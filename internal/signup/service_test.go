package signup_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/email"
	"github.com/plume-newsletter/plume/internal/email/logprovider"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/signup"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

const baseURL = "https://send.example.test"

func seed(t *testing.T, ctx context.Context, q *gen.Queries) (owner uuid.UUID, l gen.List) {
	t.Helper()
	owner = uuid.New()

	b, err := brand.New(q).Create(ctx, owner, brand.BrandInput{
		Name: "Acme", FromName: "Acme News", FromEmail: "news@acme.test", ReplyTo: "reply@acme.test",
	})
	if err != nil {
		t.Fatalf("seed brand: %v", err)
	}
	l, err = list.New(q).Create(ctx, owner, b.ID, "Main List")
	if err != nil {
		t.Fatalf("seed list: %v", err)
	}
	return owner, l
}

func TestSubscribeCreatesPendingAndSendsConfirmEmail(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()
	provider := logprovider.New(io.Discard)

	_, l := seed(t, ctx, q)

	var addedCount, confirmedCount int
	h.AddAction(subscriber.HookSubscriberAdded, 10, func(_ context.Context, _ any) error {
		addedCount++
		return nil
	})
	h.AddAction(signup.HookConfirmed, 10, func(_ context.Context, _ any) error {
		confirmedCount++
		return nil
	})

	svc := signup.New(q, h, email.NewStaticResolver(provider), baseURL)

	if err := svc.Subscribe(ctx, l.ID, "new@x.test", "New"); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	sub, err := q.GetSubscriberInListByEmail(ctx, gen.GetSubscriberInListByEmailParams{ListID: l.ID, Email: "new@x.test"})
	if err != nil {
		t.Fatalf("GetSubscriberInListByEmail: %v", err)
	}
	if sub.Status != "pending" {
		t.Fatalf("status = %q, want pending", sub.Status)
	}
	if addedCount != 1 {
		t.Fatalf("subscriber.added fired %d times, want 1", addedCount)
	}

	sent := provider.Sent()
	if len(sent) != 1 {
		t.Fatalf("len(sent) = %d, want 1", len(sent))
	}
	if sent[0].To != "new@x.test" {
		t.Fatalf("sent[0].To = %q, want new@x.test", sent[0].To)
	}
	wantLink := baseURL + "/confirm/" + sub.ID.String()
	if !strings.Contains(sent[0].HTML, wantLink) {
		t.Fatalf("sent[0].HTML = %q, want it to contain %q", sent[0].HTML, wantLink)
	}

	// Re-subscribing the same email: still pending, no duplicate row, a
	// resend confirmation email, and subscriber.added must NOT fire again.
	if err := svc.Subscribe(ctx, l.ID, "new@x.test", "New"); err != nil {
		t.Fatalf("second Subscribe: %v", err)
	}
	if addedCount != 1 {
		t.Fatalf("subscriber.added fired %d times after resubscribe, want 1", addedCount)
	}
	subs, err := q.ListSubscribersInList(ctx, gen.ListSubscribersInListParams{ListID: l.ID, OwnerID: sub.OwnerID})
	if err != nil {
		t.Fatalf("ListSubscribersInList: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("len(subs) = %d, want 1 (no duplicate row)", len(subs))
	}
	if subs[0].Status != "pending" {
		t.Fatalf("status after resubscribe = %q, want pending", subs[0].Status)
	}
	if len(provider.Sent()) != 2 {
		t.Fatalf("len(sent) after resubscribe = %d, want 2 (resend)", len(provider.Sent()))
	}

	// Confirm activates and fires subscriber.confirmed exactly once.
	if err := svc.Confirm(ctx, sub.ID); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	confirmed, err := q.GetSubscriberByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetSubscriberByID: %v", err)
	}
	if confirmed.Status != "active" {
		t.Fatalf("status after Confirm = %q, want active", confirmed.Status)
	}
	if confirmedCount != 1 {
		t.Fatalf("subscriber.confirmed fired %d times, want 1", confirmedCount)
	}
}

func TestSubscribeActiveSubscriberIsNoOp(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()
	provider := logprovider.New(io.Discard)

	owner, l := seed(t, ctx, q)
	if _, _, err := subscriber.New(q, h).Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "active@x.test", Status: "active"}); err != nil {
		t.Fatalf("seed active subscriber: %v", err)
	}

	svc := signup.New(q, h, email.NewStaticResolver(provider), baseURL)
	if err := svc.Subscribe(ctx, l.ID, "active@x.test", "Active"); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	if len(provider.Sent()) != 0 {
		t.Fatalf("len(sent) = %d, want 0 (no-op for already-active subscriber)", len(provider.Sent()))
	}
	sub, err := q.GetSubscriberInListByEmail(ctx, gen.GetSubscriberInListByEmailParams{ListID: l.ID, Email: "active@x.test"})
	if err != nil {
		t.Fatalf("GetSubscriberInListByEmail: %v", err)
	}
	if sub.Status != "active" {
		t.Fatalf("status = %q, want active (unchanged)", sub.Status)
	}
}

func TestSubscribeUnknownListReturnsErrListNotFound(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()
	provider := logprovider.New(io.Discard)

	svc := signup.New(q, h, email.NewStaticResolver(provider), baseURL)
	err := svc.Subscribe(ctx, uuid.New(), "ghost@x.test", "Ghost")
	if !errors.Is(err, signup.ErrListNotFound) {
		t.Fatalf("err = %v, want ErrListNotFound", err)
	}
}

func TestConfirmUnknownSubscriberIsIdempotentNoOp(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()
	provider := logprovider.New(io.Discard)

	var confirmedCount int
	h.AddAction(signup.HookConfirmed, 10, func(_ context.Context, _ any) error {
		confirmedCount++
		return nil
	})

	svc := signup.New(q, h, email.NewStaticResolver(provider), baseURL)
	if err := svc.Confirm(ctx, uuid.New()); err != nil {
		t.Fatalf("Confirm with unknown subscriber should not error: %v", err)
	}
	if confirmedCount != 0 {
		t.Fatalf("subscriber.confirmed fired %d times, want 0", confirmedCount)
	}
}

func TestConfirmClearsSuppression(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()
	provider := logprovider.New(io.Discard)

	owner, l := seed(t, ctx, q)

	const addr = "resub@x.test"
	if err := q.InsertSuppression(ctx, gen.InsertSuppressionParams{
		ID: uuid.New(), OwnerID: owner, Email: addr, Reason: "unsubscribe",
	}); err != nil {
		t.Fatalf("seed suppression: %v", err)
	}

	svc := signup.New(q, h, email.NewStaticResolver(provider), baseURL)
	if err := svc.Subscribe(ctx, l.ID, addr, "Resub"); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	sub, err := q.GetSubscriberInListByEmail(ctx, gen.GetSubscriberInListByEmailParams{ListID: l.ID, Email: addr})
	if err != nil {
		t.Fatalf("GetSubscriberInListByEmail: %v", err)
	}

	suppressedBefore, err := q.IsSuppressed(ctx, gen.IsSuppressedParams{OwnerID: owner, Email: addr})
	if err != nil {
		t.Fatalf("IsSuppressed before Confirm: %v", err)
	}
	if !suppressedBefore {
		t.Fatalf("suppressedBefore = false, want true (seeded)")
	}

	if err := svc.Confirm(ctx, sub.ID); err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	confirmed, err := q.GetSubscriberByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetSubscriberByID: %v", err)
	}
	if confirmed.Status != "active" {
		t.Fatalf("status after Confirm = %q, want active", confirmed.Status)
	}

	suppressedAfter, err := q.IsSuppressed(ctx, gen.IsSuppressedParams{OwnerID: owner, Email: addr})
	if err != nil {
		t.Fatalf("IsSuppressed after Confirm: %v", err)
	}
	if suppressedAfter {
		t.Fatalf("suppressedAfter = true, want false (Confirm should clear suppression)")
	}
}

func TestConfirmAlreadyActiveIsIdempotent(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()
	provider := logprovider.New(io.Discard)

	owner, l := seed(t, ctx, q)
	sub, _, err := subscriber.New(q, h).Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "already@x.test", Status: "active"})
	if err != nil {
		t.Fatalf("seed active subscriber: %v", err)
	}

	var confirmedCount int
	h.AddAction(signup.HookConfirmed, 10, func(_ context.Context, _ any) error {
		confirmedCount++
		return nil
	})

	svc := signup.New(q, h, email.NewStaticResolver(provider), baseURL)
	if err := svc.Confirm(ctx, sub.ID); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if confirmedCount != 0 {
		t.Fatalf("subscriber.confirmed fired %d times for already-active subscriber, want 0", confirmedCount)
	}
}
