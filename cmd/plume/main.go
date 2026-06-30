package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/plume-newsletter/plume/internal/abtest"
	"github.com/plume-newsletter/plume/internal/ai"
	"github.com/plume-newsletter/plume/internal/analytics"
	"github.com/plume-newsletter/plume/internal/apikey"
	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/automation"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/campaign"
	"github.com/plume-newsletter/plume/internal/crypto"
	"github.com/plume-newsletter/plume/internal/email"
	"github.com/plume-newsletter/plume/internal/email/logprovider"
	"github.com/plume-newsletter/plume/internal/email/ses"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/httpapi"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/render"
	"github.com/plume-newsletter/plume/internal/report"
	"github.com/plume-newsletter/plume/internal/segment"
	"github.com/plume-newsletter/plume/internal/sending"
	"github.com/plume-newsletter/plume/internal/settings"
	"github.com/plume-newsletter/plume/internal/signup"
	"github.com/plume-newsletter/plume/internal/signupform"
	"github.com/plume-newsletter/plume/internal/store"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
	"github.com/plume-newsletter/plume/internal/team"
	"github.com/plume-newsletter/plume/internal/template"
	"github.com/plume-newsletter/plume/internal/tracking"
	"github.com/plume-newsletter/plume/internal/unsubscribe"
	"github.com/plume-newsletter/plume/internal/webhook"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("PLUME_DATABASE_URL")
	if dsn == "" {
		log.Fatal("PLUME_DATABASE_URL is required")
	}

	// *sql.DB (via pgx stdlib) is used only for goose migrations.
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := store.Migrate(ctx, db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	// gen.DBTX requires pgx interface; use *pgxpool.Pool for gen.New.
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("pgxpool: %v", err)
	}
	defer pool.Close()

	q := gen.New(pool)
	if err := auth.EnsureAdmin(ctx, q, os.Getenv("PLUME_ADMIN_EMAIL"), os.Getenv("PLUME_ADMIN_PASSWORD")); err != nil {
		log.Fatalf("bootstrap admin: %v", err)
	}

	secret := os.Getenv("PLUME_COOKIE_SECRET")
	if len(secret) < 32 {
		log.Fatal("PLUME_COOKIE_SECRET must be at least 32 bytes")
	}

	secretKey := os.Getenv("PLUME_SECRET_KEY")
	cipher, err := crypto.New([]byte(secretKey))
	if err != nil {
		log.Fatalf("PLUME_SECRET_KEY invalid: %v (must be exactly 32 bytes)", err)
	}

	h := hooks.New()
	render.Register(h)
	// (further built-in action/filter handlers register here in later phases)

	// EmailProvider seam: logprovider is the zero-config fallback so Plume
	// runs with no AWS. The AdminResolver picks the real SES provider once an
	// admin has stored credentials (internal/settings), else it returns this
	// fallback — so sends never fail just because SES isn't configured yet.
	logProvider := logprovider.New(os.Stdout)
	resolver := email.NewAdminResolver(q, cipher, logProvider, func(ctx context.Context, accessKeyID, secretAccessKey, region string) (email.Provider, error) {
		return ses.NewFromCreds(ctx, accessKeyID, secretAccessKey, region)
	})

	sendingSvc := sending.New(pool, q, h, resolver)
	trackingSvc := tracking.New(q, h)
	unsubscribeSvc := unsubscribe.New(q, h)

	// Outbound webhooks: forward existing domain events to user-configured
	// endpoints. These hook handlers are fire-and-react — Deliver returns
	// immediately and never blocks or fails the triggering action.
	webhookSvc := webhook.New(q)
	h.AddAction(subscriber.HookSubscriberAdded, 50, func(ctx context.Context, p any) error {
		if a, ok := p.(subscriber.AddedPayload); ok {
			webhookSvc.Deliver(ctx, a.Subscriber.OwnerID, "subscriber.created", subscriberData(a.Subscriber))
		}
		return nil
	})
	h.AddAction(signup.HookConfirmed, 50, func(ctx context.Context, p any) error {
		if a, ok := p.(signup.ConfirmedPayload); ok {
			webhookSvc.Deliver(ctx, a.Subscriber.OwnerID, "subscriber.confirmed", subscriberData(a.Subscriber))
		}
		return nil
	})
	h.AddAction(sending.HookCampaignSent, 50, func(ctx context.Context, p any) error {
		if a, ok := p.(sending.SendingPayload); ok {
			c := a.Campaign
			webhookSvc.Deliver(ctx, c.OwnerID, "campaign.sent", map[string]any{
				"id": c.ID.String(), "subject": c.Subject, "status": c.Status,
			})
		}
		return nil
	})

	baseURL := os.Getenv("PLUME_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	worker := sending.NewWorker(pool, q, h, resolver, baseURL, 20)
	go worker.Start(ctx, 5*time.Second)

	automationSvc := automation.New(pool, q, h)
	autoWorker := automation.NewWorker(q, resolver, baseURL, 20)
	go autoWorker.Start(ctx, 30*time.Second)

	// Signup's confirmation email is transactional: it reuses the same
	// resolver + baseURL as the bulk worker but sends directly, never
	// through the campaign queue.
	signupSvc := signup.New(q, h, resolver, baseURL)

	campaignSvc := campaign.New(q)
	deps := httpapi.AuthDeps{
		Queries:     q,
		Cookie:      auth.NewCookie([]byte(secret)),
		Secure:      os.Getenv("PLUME_SECURE_COOKIES") == "true",
		Brands:      brand.New(q),
		Lists:       list.New(q),
		Subscribers: subscriber.New(q, h),
		Campaigns:   campaignSvc,
		Sending:     sendingSvc,
		Tracking:    trackingSvc,
		Unsubscribe: unsubscribeSvc,
		Signup:      signupSvc,
		Reports:     report.New(q),
		Segments:    segment.New(pool, q),
		Settings:    settings.New(q, cipher),
		AI:          ai.NewAnthropic(),
		Analytics:   analytics.New(q),
		SignupForms: signupform.New(q),
		Team:        team.New(q, resolver, baseURL),
		ABTests:     abtest.New(q),
		Automations: automationSvc,
		Templates:   template.New(q, campaignSvc),
		APIKeys:     apikey.New(q),
		Webhooks:    webhookSvc,
	}

	addr := os.Getenv("PLUME_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("plume listening on %s", addr)
	if err := http.ListenAndServe(addr, httpapi.NewRouter(deps)); err != nil {
		log.Fatal(err)
	}
}

// subscriberData is the webhook payload shape for subscriber events — a small,
// stable subset of the row (no internal columns).
func subscriberData(s gen.Subscriber) map[string]any {
	return map[string]any{
		"id": s.ID.String(), "email": s.Email, "name": s.Name,
		"status": s.Status, "listId": s.ListID.String(),
	}
}
