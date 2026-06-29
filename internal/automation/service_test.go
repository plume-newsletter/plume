package automation_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/automation"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestAutomationServiceCRUDStepsEnroll(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()
	h := hooks.New()
	b, _ := brand.New(q).Create(ctx, owner, brand.BrandInput{Name: "Acme", FromName: "A", FromEmail: "a@a.test", ReplyTo: ""})
	l, _ := list.New(q).Create(ctx, owner, b.ID, "Main")
	sub, _, _ := subscriber.New(q, h).Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: uuid.New().String() + "@t.com", Status: "active"})

	svc := automation.New(pool, q, h)

	a, err := svc.Create(ctx, owner, "Welcome", l.ID)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	id := uuid.MustParse(a.ID)
	if a.Status != "draft" {
		t.Errorf("status=%q", a.Status)
	}

	// steps: send then wait; validation rejects a bad step
	if err := svc.ReplaceSteps(ctx, owner, id, []automation.Step{{Kind: "send", Subject: ""}}); err != automation.ErrInvalid {
		t.Errorf("empty-subject send = %v, want ErrInvalid", err)
	}
	if err := svc.ReplaceSteps(ctx, owner, id, []automation.Step{
		{Kind: "send", Subject: "Hi", HTML: "<p>Hi</p>"}, {Kind: "wait", WaitDays: 2},
	}); err != nil {
		t.Fatalf("replace steps: %v", err)
	}

	got, _ := svc.Get(ctx, owner, id)
	if len(got.Steps) != 2 || got.StepSends != 1 {
		t.Fatalf("steps=%+v sends=%d", got.Steps, got.StepSends)
	}

	// enroll only when live
	if err := svc.Enroll(ctx, sub); err != nil {
		t.Fatalf("enroll(draft): %v", err)
	}
	after, _ := svc.Get(ctx, owner, id)
	if after.InFlow != 0 {
		t.Errorf("inFlow before live = %d, want 0", after.InFlow)
	}

	if err := svc.SetStatus(ctx, owner, id, "live"); err != nil {
		t.Fatalf("set live: %v", err)
	}
	if err := svc.Enroll(ctx, sub); err != nil {
		t.Fatalf("enroll(live): %v", err)
	}
	if err := svc.Enroll(ctx, sub); err != nil {
		t.Fatalf("enroll(dup): %v", err)
	} // idempotent
	live, _ := svc.Get(ctx, owner, id)
	if live.InFlow != 1 {
		t.Errorf("inFlow = %d, want 1", live.InFlow)
	}
}
