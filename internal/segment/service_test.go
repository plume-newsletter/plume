package segment_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/campaign"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/segment"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestPreviewConditions(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()
	h := hooks.New()

	b, _ := brand.New(q).Create(ctx, owner, brand.BrandInput{Name: "Acme", FromName: "A", FromEmail: "a@a.test", ReplyTo: ""})
	l, _ := list.New(q).Create(ctx, owner, b.ID, "Main")
	c, _ := campaign.New(q).Create(ctx, owner, b.ID, campaign.CampaignInput{Subject: "Hi", HtmlBody: "<p>Hi</p>", PlainBody: "Hi"})
	subSvc := subscriber.New(q, h)
	var subs [4]gen.Subscriber
	for i := range subs {
		s, _, err := subSvc.Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: uuid.New().String() + "@t.com", Status: "active"})
		if err != nil {
			t.Fatalf("sub %d: %v", i, err)
		}
		subs[i] = s
	}
	mustEvent := func(sub gen.Subscriber, typ string) {
		if _, err := q.InsertEmailEvent(ctx, gen.InsertEmailEventParams{ID: uuid.New(), CampaignID: c.ID, SubscriberID: sub.ID, Type: typ}); err != nil {
			t.Fatalf("event: %v", err)
		}
	}
	mustEvent(subs[0], "open")
	mustEvent(subs[1], "click")
	cf, err := q.CreateCustomField(ctx, gen.CreateCustomFieldParams{ID: uuid.New(), OwnerID: owner, ListID: l.ID, Name: "Plan"})
	if err != nil {
		t.Fatalf("field: %v", err)
	}
	if err := q.UpsertFieldValue(ctx, gen.UpsertFieldValueParams{ID: uuid.New(), SubscriberID: subs[0].ID, CustomFieldID: cf.ID, Value: "Pro"}); err != nil {
		t.Fatalf("value: %v", err)
	}

	svc := segment.New(pool, q)
	check := func(name, match string, conds []segment.Condition, want int) {
		p, err := svc.Preview(ctx, owner, match, conds)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if p.Count != want {
			t.Errorf("%s: count = %d, want %d", name, p.Count, want)
		}
	}
	check("empty=all", "all", nil, 4)
	check("opened ever", "all", []segment.Condition{{Type: "opened", Op: "ever"}}, 1)
	check("clicked never", "all", []segment.Condition{{Type: "clicked", Op: "never"}}, 3)
	check("opened in_last 7", "all", []segment.Condition{{Type: "opened", Op: "in_last", Days: 7}}, 1)
	check("field Plan=Pro", "all", []segment.Condition{{Type: "field", Op: "equals", Field: "Plan", Value: "Pro"}}, 1)
	check("status is active", "all", []segment.Condition{{Type: "status", Op: "is", Value: "active"}}, 4)
	// match=any: opened ever OR clicked ever = subs[0] + subs[1] = 2
	check("any opened/clicked", "any", []segment.Condition{{Type: "opened", Op: "ever"}, {Type: "clicked", Op: "ever"}}, 2)
	// match=all: opened ever AND field Plan=Pro = subs[0] = 1
	check("all opened+field", "all", []segment.Condition{{Type: "opened", Op: "ever"}, {Type: "field", Op: "equals", Field: "Plan", Value: "Pro"}}, 1)

	if _, err := svc.Preview(ctx, owner, "all", []segment.Condition{{Type: "bogus", Op: "x"}}); err == nil {
		t.Error("expected ErrInvalidCondition for bogus type")
	}
	// Total + percent
	p, _ := svc.Preview(ctx, owner, "all", []segment.Condition{{Type: "opened", Op: "ever"}})
	if p.Total != 4 || p.Percent != 0.25 {
		t.Errorf("total=%d percent=%v, want 4 / 0.25", p.Total, p.Percent)
	}
}

func TestSegmentCRUD(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()
	svc := segment.New(pool, q)

	seg, err := svc.Create(ctx, owner, "Engaged", "all", []segment.Condition{{Type: "opened", Op: "ever"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if seg.Name != "Engaged" || seg.Match != "all" || len(seg.Conditions) != 1 {
		t.Fatalf("bad seg: %+v", seg)
	}

	list, err := svc.List(ctx, owner)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}

	id := uuid.MustParse(seg.ID)
	got, err := svc.Get(ctx, owner, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Engaged" {
		t.Errorf("get name = %q", got.Name)
	}

	upd, err := svc.Update(ctx, owner, id, "Engaged v2", "any", []segment.Condition{{Type: "clicked", Op: "ever"}})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if upd.Name != "Engaged v2" || upd.Match != "any" {
		t.Errorf("update: %+v", upd)
	}

	// wrong owner -> ErrNotFound
	if _, err := svc.Get(ctx, uuid.New(), id); err != segment.ErrNotFound {
		t.Errorf("get other owner: err = %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, owner, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, _ = svc.List(ctx, owner)
	if len(list) != 0 {
		t.Errorf("after delete list len = %d, want 0", len(list))
	}
}
