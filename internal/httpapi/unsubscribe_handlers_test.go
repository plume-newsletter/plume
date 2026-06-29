package httpapi_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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
	"github.com/plume-newsletter/plume/internal/unsubscribe"
)

func TestUnsubscribeConfirmPageDoesNotMutate(t *testing.T) {
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
	recipient, err := q.CreateRecipient(ctx, gen.CreateRecipientParams{ID: uuid.New(), CampaignID: c.ID, SubscriberID: sub.ID})
	if err != nil {
		t.Fatalf("seed recipient: %v", err)
	}

	router := httpapi.NewRouter(httpapi.AuthDeps{Unsubscribe: unsubscribe.New(q, h)})
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/u/" + recipient.ID.String())
	if err != nil {
		t.Fatalf("GET /u/{id}: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	body := new(bytes.Buffer)
	_, _ = body.ReadFrom(resp.Body)
	page := body.String()
	wantAction := "/u/" + recipient.ID.String()
	if !strings.Contains(page, `method="POST"`) || !strings.Contains(page, wantAction) {
		t.Fatalf("confirm page = %q, want a POST form targeting %s", page, wantAction)
	}

	// GET must not mutate.
	unchanged, err := q.GetSubscriberByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetSubscriberByID: %v", err)
	}
	if unchanged.Status != "active" {
		t.Fatalf("subscriber status after GET = %q, want active (GET must not mutate)", unchanged.Status)
	}
}

func TestUnsubscribeActionUnsubscribesAndAlwaysReturns200(t *testing.T) {
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
	sub, _, err := subscriber.New(q, h).Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "y@test.com", Status: "active"})
	if err != nil {
		t.Fatalf("seed subscriber: %v", err)
	}
	c, err := campaign.New(q).Create(ctx, owner, b.ID, campaign.CampaignInput{Subject: "S", HtmlBody: "<p>hi</p>"})
	if err != nil {
		t.Fatalf("seed campaign: %v", err)
	}
	recipient, err := q.CreateRecipient(ctx, gen.CreateRecipientParams{ID: uuid.New(), CampaignID: c.ID, SubscriberID: sub.ID})
	if err != nil {
		t.Fatalf("seed recipient: %v", err)
	}

	router := httpapi.NewRouter(httpapi.AuthDeps{Unsubscribe: unsubscribe.New(q, h)})
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/u/"+recipient.ID.String(), "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("POST /u/{id}: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	body := new(bytes.Buffer)
	_, _ = body.ReadFrom(resp.Body)
	if !strings.Contains(body.String(), "unsubscribed") {
		t.Fatalf("body = %q, want confirmation text", body.String())
	}

	updated, err := q.GetSubscriberByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetSubscriberByID: %v", err)
	}
	if updated.Status != "unsubscribed" {
		t.Fatalf("subscriber status = %q, want unsubscribed", updated.Status)
	}

	// Bad uuid must still return 200 + generic page (leak-safe).
	resp2, err := http.Post(srv.URL+"/u/not-a-uuid", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("POST /u/{bad}: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("bad id status = %d, want 200", resp2.StatusCode)
	}
}
