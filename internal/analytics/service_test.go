package analytics_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/analytics"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/campaign"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func seed(t *testing.T, ctx context.Context, q *gen.Queries) uuid.UUID {
	t.Helper()
	owner := uuid.New()
	h := hooks.New()
	b, err := brand.New(q).Create(ctx, owner, brand.BrandInput{Name: "Acme", FromName: "Acme", FromEmail: "n@acme.test", ReplyTo: ""})
	if err != nil {
		t.Fatalf("brand: %v", err)
	}
	l, err := list.New(q).Create(ctx, owner, b.ID, "Main")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	c, err := campaign.New(q).Create(ctx, owner, b.ID, campaign.CampaignInput{Subject: "Hello", HtmlBody: "<p>Hi</p>", PlainBody: "Hi"})
	if err != nil {
		t.Fatalf("campaign: %v", err)
	}
	subSvc := subscriber.New(q, h)
	var subs [3]gen.Subscriber
	for i := range subs {
		s, _, err := subSvc.Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: uuid.New().String() + "@test.com", Status: "active"})
		if err != nil {
			t.Fatalf("sub %d: %v", i, err)
		}
		subs[i] = s
	}
	for i, s := range subs {
		rcpt, err := q.CreateRecipient(ctx, gen.CreateRecipientParams{ID: uuid.New(), CampaignID: c.ID, SubscriberID: s.ID})
		if err != nil {
			t.Fatalf("recipient %d: %v", i, err)
		}
		if i < 2 {
			if err := q.MarkRecipientSent(ctx, rcpt.ID); err != nil {
				t.Fatalf("sent %d: %v", i, err)
			}
		}
	}
	for _, s := range []gen.Subscriber{subs[0], subs[0], subs[1]} {
		if _, err := q.InsertEmailEvent(ctx, gen.InsertEmailEventParams{ID: uuid.New(), CampaignID: c.ID, SubscriberID: s.ID, Type: "open"}); err != nil {
			t.Fatalf("open: %v", err)
		}
	}
	if _, err := q.InsertEmailEvent(ctx, gen.InsertEmailEventParams{ID: uuid.New(), CampaignID: c.ID, SubscriberID: subs[0].ID, Type: "click"}); err != nil {
		t.Fatalf("click: %v", err)
	}
	return owner
}

func TestOverviewAggregates(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := seed(t, ctx, q)

	ov, err := analytics.New(q).Overview(ctx, owner, 30)
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}

	if ov.Subscribers != 3 {
		t.Errorf("Subscribers = %d, want 3", ov.Subscribers)
	}
	if ov.NetNewSubs != 3 {
		t.Errorf("NetNewSubs = %d, want 3", ov.NetNewSubs)
	}
	// 2 distinct openers / 2 sent = 1.0
	if ov.AvgOpenRate != 1.0 {
		t.Errorf("AvgOpenRate = %v, want 1.0", ov.AvgOpenRate)
	}
	// 1 click / 2 sent = 0.5
	if ov.ClickRate != 0.5 {
		t.Errorf("ClickRate = %v, want 0.5", ov.ClickRate)
	}
	// 2 sent / 1000 * 0.10 = 0.0002
	if ov.SendCost < 0.0001 || ov.SendCost > 0.0003 {
		t.Errorf("SendCost = %v, want ~0.0002", ov.SendCost)
	}
	if len(ov.Campaigns) != 1 {
		t.Fatalf("Campaigns len = %d, want 1", len(ov.Campaigns))
	}
	if ov.Campaigns[0].Sent != 2 {
		t.Errorf("campaign Sent = %d, want 2", ov.Campaigns[0].Sent)
	}
	if ov.Campaigns[0].OpenRate != 1.0 {
		t.Errorf("campaign OpenRate = %v, want 1.0", ov.Campaigns[0].OpenRate)
	}
	if len(ov.TopCampaigns) != 1 || ov.TopCampaigns[0].Opens != 2 {
		t.Errorf("TopCampaigns = %+v, want 1 with Opens=2", ov.TopCampaigns)
	}
}
