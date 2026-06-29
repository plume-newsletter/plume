// Package unsubscribe implements the public one-click unsubscribe flow: it
// marks the subscriber unsubscribed, suppresses the owner+email, records a
// campaign-attributed unsubscribe event, and fires subscriber.unsubscribed.
package unsubscribe

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

const HookUnsubscribed = "subscriber.unsubscribed"

// UnsubscribedPayload is the Action payload for subscriber.unsubscribed.
type UnsubscribedPayload struct {
	Subscriber gen.Subscriber
	Event      gen.EmailEvent
}

type Service struct {
	q *gen.Queries
	h *hooks.Hooks
}

func New(q *gen.Queries, h *hooks.Hooks) *Service { return &Service{q: q, h: h} }

// Unsubscribe resolves recipientID to its campaign/subscriber, marks the
// subscriber unsubscribed, records an 'unsubscribe' email_event attributed to
// the campaign, upserts a suppression_entry for the subscriber's owner+email,
// and fires subscriber.unsubscribed. An unknown recipient (or a subscriber
// that no longer exists) is treated as a no-op (no error) so the public
// handler never leaks whether an id is valid. Real DB errors propagate.
// Idempotent: the status set and suppression upsert are safe to repeat.
func (s *Service) Unsubscribe(ctx context.Context, recipientID uuid.UUID) error {
	recipient, err := s.q.GetRecipientByID(ctx, recipientID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	sub, err := s.q.GetSubscriberByID(ctx, recipient.SubscriberID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	// Already unsubscribed; skip duplicate event and redundant hook fire.
	if sub.Status == "unsubscribed" {
		return nil
	}

	if err := s.q.SetSubscriberStatusByID(ctx, gen.SetSubscriberStatusByIDParams{ID: sub.ID, Status: "unsubscribed"}); err != nil {
		return err
	}

	event, err := s.q.InsertEmailEvent(ctx, gen.InsertEmailEventParams{
		ID:           uuid.New(),
		CampaignID:   recipient.CampaignID,
		SubscriberID: sub.ID,
		Type:         "unsubscribe",
	})
	if err != nil {
		return err
	}

	if err := s.q.InsertSuppression(ctx, gen.InsertSuppressionParams{
		ID:      uuid.New(),
		OwnerID: sub.OwnerID,
		Email:   sub.Email,
		Reason:  "unsubscribe",
	}); err != nil {
		return err
	}

	if err := s.h.DoAction(ctx, HookUnsubscribed, UnsubscribedPayload{Subscriber: sub, Event: event}); err != nil {
		// Actions are fire-and-react: a handler failure must not fail the
		// unsubscribe (the rows are already committed). Log and continue.
		log.Printf("subscriber.unsubscribed action error for subscriber %s: %v", sub.ID, err)
	}
	return nil
}
