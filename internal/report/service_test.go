package report_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/campaign"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/report"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

// seed builds a campaign with 3 recipients (2 marked sent) and the event mix
// described in the task: 3 opens from 2 distinct subscribers, 1 click, 1
// bounce, 1 unsubscribe.
func seed(t *testing.T, ctx context.Context, q *gen.Queries) (owner uuid.UUID, c gen.Campaign) {
	t.Helper()
	owner = uuid.New()
	h := hooks.New()

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
	c, err = campaign.New(q).Create(ctx, owner, b.ID, campaign.CampaignInput{
		Subject:   "Hello",
		HtmlBody:  "<p>Hi</p>",
		PlainBody: "Hi",
	})
	if err != nil {
		t.Fatalf("seed campaign: %v", err)
	}

	subSvc := subscriber.New(q, h)
	var subs [3]gen.Subscriber
	for i := range subs {
		sub, _, err := subSvc.Add(ctx, owner, l.ID, subscriber.SubscriberInput{
			Email: uuid.New().String() + "@test.com", Status: "active",
		})
		if err != nil {
			t.Fatalf("seed subscriber %d: %v", i, err)
		}
		subs[i] = sub
	}

	for i, sub := range subs {
		recipient, err := q.CreateRecipient(ctx, gen.CreateRecipientParams{ID: uuid.New(), CampaignID: c.ID, SubscriberID: sub.ID})
		if err != nil {
			t.Fatalf("seed recipient %d: %v", i, err)
		}
		if i < 2 {
			if err := q.MarkRecipientSent(ctx, recipient.ID); err != nil {
				t.Fatalf("MarkRecipientSent %d: %v", i, err)
			}
		}
	}

	// 3 opens from 2 distinct subscribers: subs[0] opens twice, subs[1] opens once.
	for _, sub := range []gen.Subscriber{subs[0], subs[0], subs[1]} {
		if _, err := q.InsertEmailEvent(ctx, gen.InsertEmailEventParams{
			ID: uuid.New(), CampaignID: c.ID, SubscriberID: sub.ID, Type: "open",
		}); err != nil {
			t.Fatalf("seed open event: %v", err)
		}
	}
	if _, err := q.InsertEmailEvent(ctx, gen.InsertEmailEventParams{
		ID: uuid.New(), CampaignID: c.ID, SubscriberID: subs[0].ID, Type: "click",
	}); err != nil {
		t.Fatalf("seed click event: %v", err)
	}
	if _, err := q.InsertEmailEvent(ctx, gen.InsertEmailEventParams{
		ID: uuid.New(), CampaignID: c.ID, SubscriberID: subs[1].ID, Type: "bounce",
	}); err != nil {
		t.Fatalf("seed bounce event: %v", err)
	}
	if _, err := q.InsertEmailEvent(ctx, gen.InsertEmailEventParams{
		ID: uuid.New(), CampaignID: c.ID, SubscriberID: subs[2].ID, Type: "unsubscribe",
	}); err != nil {
		t.Fatalf("seed unsubscribe event: %v", err)
	}

	return owner, c
}

func TestCampaignAggregatesReport(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()

	owner, c := seed(t, ctx, q)

	svc := report.New(q)
	summary, err := svc.Campaign(ctx, owner, c.ID)
	if err != nil {
		t.Fatalf("Campaign: %v", err)
	}

	if summary.Recipients != 3 {
		t.Errorf("Recipients = %d, want 3", summary.Recipients)
	}
	if summary.Sent != 2 {
		t.Errorf("Sent = %d, want 2", summary.Sent)
	}
	if summary.Opens.Total != 3 {
		t.Errorf("Opens.Total = %d, want 3", summary.Opens.Total)
	}
	if summary.Opens.Unique != 2 {
		t.Errorf("Opens.Unique = %d, want 2", summary.Opens.Unique)
	}
	if summary.Clicks.Total != 1 {
		t.Errorf("Clicks.Total = %d, want 1", summary.Clicks.Total)
	}
	if summary.Clicks.Unique != 1 {
		t.Errorf("Clicks.Unique = %d, want 1", summary.Clicks.Unique)
	}
	if summary.Bounces != 1 {
		t.Errorf("Bounces = %d, want 1", summary.Bounces)
	}
	if summary.Complaints != 0 {
		t.Errorf("Complaints = %d, want 0", summary.Complaints)
	}
	if summary.Unsubscribes != 1 {
		t.Errorf("Unsubscribes = %d, want 1", summary.Unsubscribes)
	}
}

func TestCampaignNotOwnedReturnsErrNotFound(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()

	_, c := seed(t, ctx, q)

	svc := report.New(q)
	_, err := svc.Campaign(ctx, uuid.New(), c.ID)
	if !errors.Is(err, report.ErrNotFound) {
		t.Fatalf("Campaign with wrong owner: err = %v, want ErrNotFound", err)
	}
}
