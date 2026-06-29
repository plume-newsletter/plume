package httpapi_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/campaign"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/httpapi"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
	"github.com/plume-newsletter/plume/internal/testsupport"
	"github.com/plume-newsletter/plume/internal/tracking"
)

func TestTrackingOpenAlwaysReturnsPixel(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	h := hooks.New()

	router := httpapi.NewRouter(httpapi.AuthDeps{Tracking: tracking.New(q, h)})
	srv := httptest.NewServer(router)
	defer srv.Close()

	// Unknown recipient id still returns 200 + the pixel (no leak).
	resp, err := http.Get(srv.URL + "/t/" + uuid.New().String())
	if err != nil {
		t.Fatalf("GET /t/{id}: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/gif" {
		t.Fatalf("Content-Type = %q, want image/gif", ct)
	}
	body := new(bytes.Buffer)
	_, _ = body.ReadFrom(resp.Body)
	if body.Len() != 43 || !bytes.HasPrefix(body.Bytes(), []byte("GIF89a")) {
		t.Fatalf("body len=%d prefix=%q, want 43-byte GIF89a", body.Len(), body.Bytes()[:6])
	}
}

func TestTrackingClickRedirectsAndUnknownLinkIs404(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	h := hooks.New()
	ctx := context.Background()

	owner := uuid.New()
	b, err := brand.New(q).Create(ctx, owner, brand.BrandInput{
		Name: "Acme", FromName: "Acme News", FromEmail: "n@acme.test", ReplyTo: "r@acme.test",
	})
	if err != nil {
		t.Fatalf("seed brand: %v", err)
	}
	l, err := list.New(q).Create(ctx, owner, b.ID, "Main")
	if err != nil {
		t.Fatalf("seed list: %v", err)
	}
	sub, _, err := subscriber.New(q, h).Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "x@test.com", Status: "active"})
	if err != nil {
		t.Fatalf("seed subscriber: %v", err)
	}
	c, err := campaign.New(q).Create(ctx, owner, b.ID, campaign.CampaignInput{Subject: "S", HtmlBody: "<p>hi</p>"})
	if err != nil {
		t.Fatalf("seed campaign: %v", err)
	}
	link, err := q.CreateLink(ctx, gen.CreateLinkParams{ID: uuid.New(), CampaignID: c.ID, Url: "https://dest.test/page"})
	if err != nil {
		t.Fatalf("seed link: %v", err)
	}
	recipient, err := q.CreateRecipient(ctx, gen.CreateRecipientParams{ID: uuid.New(), CampaignID: c.ID, SubscriberID: sub.ID})
	if err != nil {
		t.Fatalf("seed recipient: %v", err)
	}

	router := httpapi.NewRouter(httpapi.AuthDeps{Tracking: tracking.New(q, h)})
	srv := httptest.NewServer(router)
	defer srv.Close()

	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	resp, err := client.Get(srv.URL + "/l/" + link.ID.String() + "/" + recipient.ID.String())
	if err != nil {
		t.Fatalf("GET /l/{linkId}/{recipientId}: %v", err)
	}
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want 302", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "https://dest.test/page" {
		t.Fatalf("Location = %q, want https://dest.test/page", loc)
	}

	resp2, err := client.Get(srv.URL + "/l/" + uuid.New().String() + "/" + recipient.ID.String())
	if err != nil {
		t.Fatalf("GET /l/{unknown}/{recipientId}: %v", err)
	}
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown link status = %d, want 404", resp2.StatusCode)
	}
}

func TestTrackingSESWebhookAlwaysReturns200(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	h := hooks.New()

	router := httpapi.NewRouter(httpapi.AuthDeps{Tracking: tracking.New(q, h)})
	srv := httptest.NewServer(router)
	defer srv.Close()

	// Malformed body still returns 200 so SNS does not retry-storm.
	resp, err := http.Post(srv.URL+"/webhook/ses", "application/json", bytes.NewBufferString(`not json`))
	if err != nil {
		t.Fatalf("POST /webhook/ses: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	// SubscriptionConfirmation envelope also returns 200.
	resp2, err := http.Post(srv.URL+"/webhook/ses", "application/json",
		bytes.NewBufferString(`{"Type":"SubscriptionConfirmation","SubscribeURL":"https://sns.test/confirm"}`))
	if err != nil {
		t.Fatalf("POST /webhook/ses (confirmation): %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("confirmation status = %d, want 200", resp2.StatusCode)
	}
}
