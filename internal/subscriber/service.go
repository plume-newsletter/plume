package subscriber

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

const HookSubscriberAdded = "subscriber.added"

var ErrListNotFound = errors.New("list not found for owner")

type AddedPayload struct{ Subscriber gen.Subscriber }

type SubscriberInput struct {
	Email  string
	Name   string
	Status string
}

type Service struct {
	q *gen.Queries
	h *hooks.Hooks
}

func New(q *gen.Queries, h *hooks.Hooks) *Service { return &Service{q: q, h: h} }

func normalizeEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

// Add inserts a subscriber, deduping by (list, email). created=false means the
// email already existed (no insert, no hook).
func (s *Service) Add(ctx context.Context, owner, listID uuid.UUID, in SubscriberInput) (gen.Subscriber, bool, error) {
	if _, err := s.q.GetListForOwner(ctx, gen.GetListForOwnerParams{ID: listID, OwnerID: owner}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.Subscriber{}, false, ErrListNotFound
		}
		return gen.Subscriber{}, false, err
	}
	email := normalizeEmail(in.Email)

	existing, err := s.q.GetSubscriberInListByEmail(ctx, gen.GetSubscriberInListByEmailParams{ListID: listID, Email: email})
	if err == nil {
		return existing, false, nil // already present → dedupe
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return gen.Subscriber{}, false, err
	}

	status := in.Status
	if status == "" {
		status = "pending"
	}
	sub, err := s.q.CreateSubscriber(ctx, gen.CreateSubscriberParams{
		ID: uuid.New(), OwnerID: owner, ListID: listID, Email: email, Name: in.Name, Status: status,
	})
	if err != nil {
		return gen.Subscriber{}, false, err
	}
	if err := s.h.DoAction(ctx, HookSubscriberAdded, AddedPayload{Subscriber: sub}); err != nil {
		// Actions are fire-and-react: a handler failure must not fail the add or
		// corrupt the result (the row is already committed). Log and continue.
		log.Printf("subscriber.added action error for %s: %v", sub.Email, err)
	}
	return sub, true, nil
}

func (s *Service) List(ctx context.Context, owner, listID uuid.UUID) ([]gen.Subscriber, error) {
	if _, err := s.q.GetListForOwner(ctx, gen.GetListForOwnerParams{ID: listID, OwnerID: owner}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrListNotFound
		}
		return nil, err
	}
	return s.q.ListSubscribersInList(ctx, gen.ListSubscribersInListParams{ListID: listID, OwnerID: owner})
}

func (s *Service) SetStatus(ctx context.Context, owner, id uuid.UUID, status string) (gen.Subscriber, error) {
	return s.q.UpdateSubscriberStatus(ctx, gen.UpdateSubscriberStatusParams{ID: id, OwnerID: owner, Status: status})
}

func (s *Service) Delete(ctx context.Context, owner, id uuid.UUID) error {
	return s.q.DeleteSubscriber(ctx, gen.DeleteSubscriberParams{ID: id, OwnerID: owner})
}
