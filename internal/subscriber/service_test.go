package subscriber_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestListUnownedListReturnsErrListNotFound(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	ownerA := uuid.New()

	b, err := brand.New(q).Create(ctx, ownerA, brand.BrandInput{Name: "Acme", FromEmail: "n@acme.test"})
	if err != nil {
		t.Fatalf("create brand: %v", err)
	}
	l, err := list.New(q).Create(ctx, ownerA, b.ID, "News")
	if err != nil {
		t.Fatalf("create list: %v", err)
	}

	svc := subscriber.New(q, hooks.New())

	otherOwner := uuid.New()
	_, listErr := svc.List(ctx, otherOwner, l.ID)
	if !errors.Is(listErr, subscriber.ErrListNotFound) {
		t.Fatalf("expected ErrListNotFound, got: %v", listErr)
	}
}

func TestAddSubscriberDedupesAndFiresHook(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()

	b, _ := brand.New(q).Create(ctx, owner, brand.BrandInput{Name: "Acme", FromEmail: "n@acme.test"})
	l, _ := list.New(q).Create(ctx, owner, b.ID, "News")

	h := hooks.New()
	var fired int
	h.AddAction(subscriber.HookSubscriberAdded, 0, func(_ context.Context, _ any) error {
		fired++
		return nil
	})
	svc := subscriber.New(q, h)

	_, created, err := svc.Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "a@x.test", Name: "A"})
	if err != nil || !created {
		t.Fatalf("first add: created=%v err=%v", created, err)
	}
	// Duplicate email in same list → not created, no second hook.
	_, created2, err := svc.Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "a@x.test"})
	if err != nil || created2 {
		t.Fatalf("dup add: created=%v err=%v (want created=false)", created2, err)
	}
	if fired != 1 {
		t.Fatalf("subscriber.added fired %d times, want 1", fired)
	}

	subs, _ := svc.List(ctx, owner, l.ID)
	if len(subs) != 1 {
		t.Fatalf("list has %d subscribers, want 1", len(subs))
	}
}

func TestAddSucceedsEvenIfActionHandlerErrors(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()

	b, _ := brand.New(q).Create(ctx, owner, brand.BrandInput{Name: "Acme", FromEmail: "n@acme.test"})
	l, _ := list.New(q).Create(ctx, owner, b.ID, "News")

	h := hooks.New()
	h.AddAction(subscriber.HookSubscriberAdded, 0, func(_ context.Context, _ any) error {
		return errors.New("boom: handler failed")
	})
	svc := subscriber.New(q, h)

	sub, created, err := svc.Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "b@x.test", Name: "B"})
	if err != nil {
		t.Fatalf("add returned err=%v, want nil (handler failure must be non-fatal)", err)
	}
	if !created {
		t.Fatalf("add returned created=false, want true (row is already committed)")
	}
	if sub.Email != "b@x.test" {
		t.Fatalf("add returned subscriber email=%q, want b@x.test", sub.Email)
	}

	subs, err := svc.List(ctx, owner, l.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(subs) != 1 || subs[0].Email != "b@x.test" {
		t.Fatalf("subscriber was not actually persisted: %+v", subs)
	}
}
