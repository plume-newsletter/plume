package unsubscribe_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/campaign"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
	"github.com/plume-newsletter/plume/internal/testsupport"
	"github.com/plume-newsletter/plume/internal/unsubscribe"
)

func seed(t *testing.T, ctx context.Context, q *gen.Queries, h *hooks.Hooks) (owner uuid.UUID, c gen.Campaign, sub gen.Subscriber, recipient gen.CampaignRecipient) {
	t.Helper()
	owner = uuid.New()

	b, err := brand.New(q).Create(ctx, owner, brand.BrandInput{
		Name: "Acme", FromName: "Acme News", FromEmail: "news@acme.test", ReplyTo: "reply@acme.test",
	})
	if err != nil {
		t.Fatalf("seed brand: %v", err)
	}
	l, err := list.New(q).Create(ctx, owner, b.ID, "Main List")
	if err != nil {
		t.Fatalf("seed list: %v", err)
	}
	sub, _, err = subscriber.New(q, h).Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "reader@test.com", Status: "active"})
	if err != nil {
		t.Fatalf("seed subscriber: %v", err)
	}
	c, err = campaign.New(q).Create(ctx, owner, b.ID, campaign.CampaignInput{
		Subject:   "Hello",
		HtmlBody:  "<p>Hi</p>",
		PlainBody: "Hi",
	})
	if err != nil {
		t.Fatalf("seed campaign: %v", err)
	}
	recipient, err = q.CreateRecipient(ctx, gen.CreateRecipientParams{ID: uuid.New(), CampaignID: c.ID, SubscriberID: sub.ID})
	if err != nil {
		t.Fatalf("seed recipient: %v", err)
	}
	return owner, c, sub, recipient
}

func TestUnsubscribeSetsStatusSuppressesAndFiresHook(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()

	owner, c, sub, recipient := seed(t, ctx, q, h)

	var fired int
	h.AddAction(unsubscribe.HookUnsubscribed, 10, func(_ context.Context, _ any) error {
		fired++
		return nil
	})

	svc := unsubscribe.New(q, h)
	if err := svc.Unsubscribe(ctx, recipient.ID); err != nil {
		t.Fatalf("Unsubscribe: %v", err)
	}
	if fired != 1 {
		t.Fatalf("subscriber.unsubscribed fired %d times, want 1", fired)
	}

	updated, err := q.GetSubscriberByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetSubscriberByID: %v", err)
	}
	if updated.Status != "unsubscribed" {
		t.Fatalf("subscriber status = %q, want unsubscribed", updated.Status)
	}

	suppressed, err := q.IsSuppressed(ctx, gen.IsSuppressedParams{OwnerID: owner, Email: sub.Email})
	if err != nil {
		t.Fatalf("IsSuppressed: %v", err)
	}
	if !suppressed {
		t.Fatalf("expected suppression entry for %s", sub.Email)
	}

	events, err := q.ListEmailEventsForCampaign(ctx, c.ID)
	if err != nil {
		t.Fatalf("ListEmailEventsForCampaign: %v", err)
	}
	if len(events) != 1 || events[0].Type != "unsubscribe" || events[0].SubscriberID != sub.ID {
		t.Fatalf("events = %+v, want one unsubscribe event for subscriber %s", events, sub.ID)
	}

	// Idempotent: calling again must not error and status must stay unsubscribed.
	if err := svc.Unsubscribe(ctx, recipient.ID); err != nil {
		t.Fatalf("second Unsubscribe call: %v", err)
	}
	again, err := q.GetSubscriberByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetSubscriberByID after second call: %v", err)
	}
	if again.Status != "unsubscribed" {
		t.Fatalf("subscriber status after second call = %q, want unsubscribed", again.Status)
	}

	// Verify no duplicate event was inserted.
	eventsAfter, err := q.ListEmailEventsForCampaign(ctx, c.ID)
	if err != nil {
		t.Fatalf("ListEmailEventsForCampaign after second call: %v", err)
	}
	unsubCount := 0
	for _, e := range eventsAfter {
		if e.Type == "unsubscribe" && e.SubscriberID == sub.ID {
			unsubCount++
		}
	}
	if unsubCount != 1 {
		t.Fatalf("expected exactly 1 unsubscribe event after two calls, got %d", unsubCount)
	}
}

func TestUnsubscribeUnknownRecipientIsSilent(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()

	svc := unsubscribe.New(q, h)
	if err := svc.Unsubscribe(ctx, uuid.New()); err != nil {
		t.Fatalf("Unsubscribe with unknown recipient should not error: %v", err)
	}
}
