package automation_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/automation"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/email"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

type capturingProvider struct{ rec *capturingResolver }

func (p *capturingProvider) Send(_ context.Context, msg email.Message) error {
	p.rec.subjects = append(p.rec.subjects, msg.Subject)
	return nil
}
func (p *capturingProvider) Name() string { return "capturing" }

type capturingResolver struct{ subjects []string }

func (r *capturingResolver) Provider(_ context.Context) (email.Provider, error) {
	return &capturingProvider{rec: r}, nil
}

func TestWorkerDrivesJourney(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()
	h := hooks.New()
	b, _ := brand.New(q).Create(ctx, owner, brand.BrandInput{Name: "Acme", FromName: "A", FromEmail: "a@a.test", ReplyTo: ""})
	l, _ := list.New(q).Create(ctx, owner, b.ID, "Main")
	sub, _, _ := subscriber.New(q, h).Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "x@t.com", Status: "active"})

	rec := &capturingResolver{} // records subjects sent; Provider returns a provider whose Send appends
	svc := automation.New(pool, q, h)
	a, _ := svc.Create(ctx, owner, "Welcome", l.ID)
	id := uuid.MustParse(a.ID)
	_ = svc.ReplaceSteps(ctx, owner, id, []automation.Step{
		{Kind: "send", Subject: "Welcome", HTML: "<p>hi</p>"},
		{Kind: "wait", WaitDays: 2},
		{Kind: "send", Subject: "Tips", HTML: "<p>tips</p>"},
	})
	_ = svc.SetStatus(ctx, owner, id, "live")
	_ = svc.Enroll(ctx, sub)

	w := automation.NewWorker(q, rec, "http://x", 20)

	// tick1: processes the send (step0) → advances to step1 (the wait), due now.
	if n, _ := w.RunOnce(ctx); n != 1 {
		t.Fatalf("tick1 processed %d, want 1", n)
	}
	// tick2: processes the wait (step1) → advances to step2 with next_run ≈ now+2d.
	if n, _ := w.RunOnce(ctx); n != 1 {
		t.Errorf("tick2 processed %d, want 1 (wait step consumed)", n)
	}
	// tick3: step2 is not due yet (2 days out) → returns 0.
	if n, _ := w.RunOnce(ctx); n != 0 {
		t.Errorf("tick3 processed %d, want 0 (waiting 2 days)", n)
	}
	// force the wait period to be over.
	if _, err := pool.Exec(ctx, "UPDATE automation_enrollment SET next_run_at = now() - interval '1 day' WHERE automation_id=$1", id); err != nil {
		t.Fatal(err)
	}
	// tick4: processes the send (step2 "Tips") → advances to step3 (end), due now.
	if n, _ := w.RunOnce(ctx); n != 1 {
		t.Fatalf("tick4 processed %d, want 1", n)
	}
	// tick5: step_index past end → MarkEnrollmentComplete.
	if n, _ := w.RunOnce(ctx); n != 1 {
		t.Fatalf("tick5 processed %d, want 1 (complete)", n)
	}

	if len(rec.subjects) != 2 {
		t.Fatalf("subjects=%v, want exactly 2 (Welcome + Tips)", rec.subjects)
	}
	if rec.subjects[0] != "Welcome" || rec.subjects[1] != "Tips" {
		t.Errorf("subjects=%v, want [Welcome Tips]", rec.subjects)
	}
	complete, _ := q.CountEnrollmentsByStatus(ctx, gen.CountEnrollmentsByStatusParams{AutomationID: id, Status: "complete"})
	if complete != 1 {
		t.Errorf("complete=%d, want 1", complete)
	}
}

func TestWorkerHonorsLeadingWait(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()
	h := hooks.New()
	b, _ := brand.New(q).Create(ctx, owner, brand.BrandInput{Name: "Acme", FromName: "A", FromEmail: "a@a.test", ReplyTo: ""})
	l, _ := list.New(q).Create(ctx, owner, b.ID, "Main")
	sub, _, _ := subscriber.New(q, h).Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "y@t.com", Status: "active"})

	rec := &capturingResolver{}
	svc := automation.New(pool, q, h)
	a, _ := svc.Create(ctx, owner, "LeadWait", l.ID)
	id := uuid.MustParse(a.ID)
	_ = svc.ReplaceSteps(ctx, owner, id, []automation.Step{
		{Kind: "wait", WaitDays: 5},
		{Kind: "send", Subject: "Hello", HTML: "<p>hello</p>"},
	})
	_ = svc.SetStatus(ctx, owner, id, "live")
	_ = svc.Enroll(ctx, sub)

	w := automation.NewWorker(q, rec, "http://x", 20)

	// tick1: processes the wait (step0) → advances to step1 with next_run ≈ now+5d.
	// No email should be sent.
	if n, _ := w.RunOnce(ctx); n != 1 {
		t.Fatalf("tick1 processed %d, want 1 (wait step consumed)", n)
	}
	if len(rec.subjects) != 0 {
		t.Errorf("subjects=%v after wait step, want none (leading wait must not send prematurely)", rec.subjects)
	}

	// force the 5-day wait to be over.
	if _, err := pool.Exec(ctx, "UPDATE automation_enrollment SET next_run_at = now() - interval '1 day' WHERE automation_id=$1", id); err != nil {
		t.Fatal(err)
	}

	// tick2: now processes the send (step1 "Hello").
	if n, _ := w.RunOnce(ctx); n != 1 {
		t.Fatalf("tick2 processed %d, want 1 (send step)", n)
	}
	if len(rec.subjects) != 1 || rec.subjects[0] != "Hello" {
		t.Errorf("subjects=%v, want [Hello]", rec.subjects)
	}
}
