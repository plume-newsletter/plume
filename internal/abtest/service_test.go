package abtest_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/abtest"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/campaign"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestABTestStartResultsWinner(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()
	h := hooks.New()
	b, _ := brand.New(q).Create(ctx, owner, brand.BrandInput{Name: "Acme", FromName: "A", FromEmail: "a@a.test", ReplyTo: ""})
	l, _ := list.New(q).Create(ctx, owner, b.ID, "Main")
	c, _ := campaign.New(q).Create(ctx, owner, b.ID, campaign.CampaignInput{Subject: "ignored", HtmlBody: "<p>Hi</p>", PlainBody: "Hi"})
	subSvc := subscriber.New(q, h)
	var subs []gen.Subscriber
	for i := 0; i < 10; i++ {
		sub, _, err := subSvc.Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: uuid.New().String() + "@t.com", Status: "active"})
		if err != nil {
			t.Fatalf("sub: %v", err)
		}
		subs = append(subs, sub)
	}
	_ = subs

	svc := abtest.New(q)
	tst, err := svc.Create(ctx, owner, abtest.Input{CampaignID: c.ID, ListID: l.ID, SubjectA: "A subj", SubjectB: "B subj", TestPercent: 40})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	id := uuid.MustParse(tst.ID)

	if err := svc.Start(ctx, owner, id); err != nil {
		t.Fatalf("start: %v", err)
	}
	// 40% of 10 = 4 test recipients, split 2 a / 2 b
	aCount, _ := q.CountVariantRecipients(ctx, gen.CountVariantRecipientsParams{CampaignID: c.ID, Variant: "a"})
	bCount, _ := q.CountVariantRecipients(ctx, gen.CountVariantRecipientsParams{CampaignID: c.ID, Variant: "b"})
	if aCount != 2 || bCount != 2 {
		t.Fatalf("split a=%d b=%d, want 2/2", aCount, bCount)
	}

	// give variant A two opens (its two recipients), variant B none
	recips, _ := q.ListRecipientSubscriberIDs(ctx, c.ID)
	// open events for the 'a' recipients: query them
	// (simplest: insert an open for each subscriber whose recipient variant is 'a')
	// Use a direct query or iterate; here insert opens for the first 2 subscribers of the test group.
	aSubs := abtestTestVariantSubs(t, ctx, q, c.ID, "a")
	for _, sid := range aSubs {
		_, _ = q.InsertEmailEvent(ctx, gen.InsertEmailEventParams{ID: uuid.New(), CampaignID: c.ID, SubscriberID: sid, Type: "open"})
	}
	_ = recips

	res, err := svc.Results(ctx, owner, id)
	if err != nil {
		t.Fatalf("results: %v", err)
	}
	if len(res.Variants) != 2 {
		t.Fatalf("variants=%d", len(res.Variants))
	}
	if res.Variants[0].OpenRate != 1.0 {
		t.Errorf("A openRate=%v want 1.0", res.Variants[0].OpenRate)
	}
	if res.Variants[1].OpenRate != 0.0 {
		t.Errorf("B openRate=%v want 0.0", res.Variants[1].OpenRate)
	}

	if err := svc.SendWinner(ctx, owner, id, "a"); err != nil {
		t.Fatalf("winner: %v", err)
	}
	got, _ := svc.Get(ctx, owner, id)
	if got.Status != "complete" || got.Winner != "a" {
		t.Fatalf("after winner: %+v", got)
	}
	// holdout: 10 total - 4 test = 6 new recipients with subject 'A subj'
	allRecips, _ := q.ListRecipientSubscriberIDs(ctx, c.ID)
	if len(allRecips) != 10 {
		t.Errorf("recipients=%d, want 10", len(allRecips))
	}
}

// helper: subscriber ids whose recipient on this campaign has the given variant
func abtestTestVariantSubs(t *testing.T, ctx context.Context, q *gen.Queries, campaignID uuid.UUID, variant string) []uuid.UUID {
	t.Helper()
	rows, err := q.ListRecipientsForCampaign(ctx, campaignID) // add this query if needed, or query inline
	if err != nil {
		t.Fatalf("list recips: %v", err)
	}
	var out []uuid.UUID
	for _, r := range rows {
		if r.Variant == variant {
			out = append(out, r.SubscriberID)
		}
	}
	return out
}

// TestStartGuardsAgainstDoubleSend: once a campaign is started (→queued), a
// second A/B test on the same campaign cannot Start, and Create rejects a
// non-draft campaign — closing the double-send vector (final-review C1/I1).
func TestStartGuardsAgainstDoubleSend(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()
	h := hooks.New()
	b, _ := brand.New(q).Create(ctx, owner, brand.BrandInput{Name: "Acme", FromName: "A", FromEmail: "a@a.test", ReplyTo: ""})
	l, _ := list.New(q).Create(ctx, owner, b.ID, "Main")
	c, _ := campaign.New(q).Create(ctx, owner, b.ID, campaign.CampaignInput{Subject: "x", HtmlBody: "<p>Hi</p>", PlainBody: "Hi"})
	subSvc := subscriber.New(q, h)
	for i := 0; i < 4; i++ {
		if _, _, err := subSvc.Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: uuid.New().String() + "@t.com", Status: "active"}); err != nil {
			t.Fatalf("sub: %v", err)
		}
	}
	svc := abtest.New(q)
	in := abtest.Input{CampaignID: c.ID, ListID: l.ID, SubjectA: "A", SubjectB: "B", TestPercent: 50}
	t1, _ := svc.Create(ctx, owner, in)
	t2, _ := svc.Create(ctx, owner, in)
	if err := svc.Start(ctx, owner, uuid.MustParse(t1.ID)); err != nil {
		t.Fatalf("start t1: %v", err)
	}
	// campaign is now 'queued' → starting a second test on it must fail
	if err := svc.Start(ctx, owner, uuid.MustParse(t2.ID)); err != abtest.ErrState {
		t.Errorf("start t2 = %v, want ErrState", err)
	}
	// Create against the now-non-draft campaign is rejected
	if _, err := svc.Create(ctx, owner, in); err != abtest.ErrInvalid {
		t.Errorf("create on non-draft = %v, want ErrInvalid", err)
	}
}
