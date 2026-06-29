package automation

import (
	"context"
	"log"
	"time"

	"github.com/plume-newsletter/plume/internal/email"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

// Exactly ONE Worker goroutine must run at a time: ClaimDueEnrollments is a
// non-atomic SELECT, so concurrent workers would double-send. ponytail:
// single global worker; move to FOR UPDATE SKIP LOCKED if you need multiple.
type Worker struct {
	q        *gen.Queries
	resolver email.Resolver
	baseURL  string
	batch    int32
}

func NewWorker(q *gen.Queries, resolver email.Resolver, baseURL string, batch int) *Worker {
	return &Worker{q: q, resolver: resolver, baseURL: baseURL, batch: int32(batch)}
}

// dueNow returns a timestamp guaranteed to satisfy `<= now()` on the very next
// DB query, even when the Go clock is a few milliseconds ahead of the Postgres
// clock (common in Docker Desktop / VM setups). Using -1 second is safe:
// any enrollment marked "immediately due" is functionally still processed on
// the next RunOnce tick.
func dueNow() time.Time { return time.Now().Add(-time.Second) }

// RunOnce claims due enrollments (active, due, automation live) and advances each
// one step. Returns the number processed.
func (w *Worker) RunOnce(ctx context.Context) (int, error) {
	enrollments, err := w.q.ClaimDueEnrollments(ctx, w.batch)
	if err != nil {
		return 0, err
	}
	for _, e := range enrollments {
		if err := w.process(ctx, e); err != nil {
			log.Printf("automation: enrollment %s: %v", e.ID, err)
		}
	}
	return len(enrollments), nil
}

func (w *Worker) process(ctx context.Context, e gen.AutomationEnrollment) error {
	steps, err := w.q.ListStepsForAutomation(ctx, e.AutomationID)
	if err != nil {
		return err
	}
	i := int(e.StepIndex)
	if i >= len(steps) {
		return w.q.MarkEnrollmentComplete(ctx, e.ID)
	}
	step := steps[i]
	switch step.Kind {
	case "send":
		w.send(ctx, e, step) // best-effort; never wedge the enrollment
		return w.q.AdvanceEnrollment(ctx, gen.AdvanceEnrollmentParams{ID: e.ID, StepIndex: int32(i + 1), NextRunAt: dueNow()})
	case "wait":
		// The wait has not elapsed yet; next_run_at was just claimed. Schedule
		// the next tick at now+WaitDays so the worker skips until the delay expires.
		nextRun := time.Now().Add(time.Duration(step.WaitDays) * 24 * time.Hour)
		return w.q.AdvanceEnrollment(ctx, gen.AdvanceEnrollmentParams{ID: e.ID, StepIndex: int32(i + 1), NextRunAt: nextRun})
	default:
		return w.q.AdvanceEnrollment(ctx, gen.AdvanceEnrollmentParams{ID: e.ID, StepIndex: int32(i + 1), NextRunAt: dueNow()})
	}
}

func (w *Worker) send(ctx context.Context, e gen.AutomationEnrollment, step gen.AutomationStep) {
	a, err := w.q.GetAutomationByID(ctx, e.AutomationID)
	if err != nil {
		log.Printf("automation send: get automation: %v", err)
		return
	}
	l, err := w.q.GetListByID(ctx, a.ListID)
	if err != nil {
		return
	}
	b, err := w.q.GetBrandByID(ctx, l.BrandID)
	if err != nil {
		return
	}
	sub, err := w.q.GetSubscriberByID(ctx, e.SubscriberID)
	if err != nil {
		return
	}
	provider, err := w.resolver.Provider(ctx)
	if err != nil {
		log.Printf("automation send: resolve provider: %v", err)
		return
	}
	// v1: raw step HTML, no per-recipient tracking or unsubscribe (like the confirm email).
	msg := email.Message{
		From: b.FromEmail, FromName: b.FromName, ReplyTo: b.ReplyTo,
		To: sub.Email, ToName: sub.Name, Subject: step.Subject, HTML: step.Html,
	}
	if err := provider.Send(ctx, msg); err != nil {
		log.Printf("automation send: %v", err)
	}
}

// Start runs RunOnce on a ticker until ctx is canceled (mirrors sending.Worker.Start).
func (w *Worker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := w.RunOnce(ctx); err != nil {
				log.Printf("automation: RunOnce error: %v", err)
			}
		}
	}
}
