// Package tracking records opens, clicks, bounces, and complaints against
// email_event, fires the corresponding Actions, and (for bounces/complaints)
// suppresses the subscriber so future sends skip them.
package tracking

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

const (
	HookOpened     = "email.opened"
	HookClicked    = "link.clicked"
	HookBounced    = "email.bounced"
	HookComplained = "email.complained"
)

// ErrLinkNotFound is returned when RecordClick is given an unknown link id.
var ErrLinkNotFound = errors.New("link not found")

// EventPayload is the Action payload for email.opened / link.clicked /
// email.bounced / email.complained.
type EventPayload struct {
	Event gen.EmailEvent
}

type Service struct {
	q *gen.Queries
	h *hooks.Hooks
}

func New(q *gen.Queries, h *hooks.Hooks) *Service { return &Service{q: q, h: h} }

// RecordOpen looks up the recipient and inserts an 'open' email_event, firing
// email.opened. An unknown recipient id is treated as a no-op (no error) so
// the public pixel handler never leaks whether an id is valid.
func (s *Service) RecordOpen(ctx context.Context, recipientID uuid.UUID) error {
	recipient, err := s.q.GetRecipientByID(ctx, recipientID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	event, err := s.q.InsertEmailEvent(ctx, gen.InsertEmailEventParams{
		ID:           uuid.New(),
		CampaignID:   recipient.CampaignID,
		SubscriberID: recipient.SubscriberID,
		Type:         "open",
	})
	if err != nil {
		return err
	}

	if err := s.h.DoAction(ctx, HookOpened, EventPayload{Event: event}); err != nil {
		// Actions are fire-and-react: a handler failure must not fail the
		// request (the event row is already committed). Log and continue.
		log.Printf("email.opened action error for recipient %s: %v", recipientID, err)
	}
	return nil
}

// RecordClick looks up the link and recipient, inserts a 'click' email_event,
// fires link.clicked, and returns the link's destination URL for the handler
// to redirect to. An unknown link returns ErrLinkNotFound (handler maps this
// to 404 — there is nothing to redirect to). An unknown recipient cannot be
// attributed to a subscriber, so the event is skipped, but the destination
// URL is still returned so the visitor's click is not broken.
func (s *Service) RecordClick(ctx context.Context, linkID, recipientID uuid.UUID) (string, error) {
	link, err := s.q.GetLinkByID(ctx, linkID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrLinkNotFound
		}
		return "", err
	}

	recipient, err := s.q.GetRecipientByID(ctx, recipientID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Unknown recipient: still redirect (don't break the click for the
			// visitor), but there is no subscriber to attribute the event to.
			return link.Url, nil
		}
		return "", err
	}

	event, err := s.q.InsertEmailEvent(ctx, gen.InsertEmailEventParams{
		ID:           uuid.New(),
		CampaignID:   recipient.CampaignID,
		SubscriberID: recipient.SubscriberID,
		LinkID:       pgtype.UUID{Bytes: linkID, Valid: true},
		Type:         "click",
	})
	if err != nil {
		return "", err
	}

	if err := s.h.DoAction(ctx, HookClicked, EventPayload{Event: event}); err != nil {
		// Actions are fire-and-react: a handler failure must never break the
		// visitor's click (the event row is already committed). Log and
		// continue, still returning the destination URL.
		log.Printf("link.clicked action error for link %s: %v", linkID, err)
	}
	return link.Url, nil
}

// RecordBounce records an email.bounced event for every subscriber matching
// email, sets their status to 'bounced', and adds a suppression_entry scoped
// to each subscriber's owner. An email matching no subscriber is a no-op.
func (s *Service) RecordBounce(ctx context.Context, email string) error {
	return s.recordSuppressingEvent(ctx, email, "bounce", "bounced", HookBounced)
}

// RecordComplaint records an email.complained event for every subscriber
// matching email, sets their status to 'complained', and adds a
// suppression_entry scoped to each subscriber's owner. An email matching no
// subscriber is a no-op.
func (s *Service) RecordComplaint(ctx context.Context, email string) error {
	return s.recordSuppressingEvent(ctx, email, "complaint", "complained", HookComplained)
}

func (s *Service) recordSuppressingEvent(ctx context.Context, email, eventType, status, hookName string) error {
	subs, err := s.q.ListSubscribersByEmail(ctx, email)
	if err != nil {
		return err
	}

	for _, sub := range subs {
		// SES bounce/complaint notifications are not tied to a specific
		// campaign send, but email_event.campaign_id is NOT NULL, so we
		// attribute the event to the subscriber's most recent campaign send,
		// if any. A subscriber with no send history (e.g. never sent to) gets
		// its status/suppression updated but no email_event row.
		campaignID, err := s.q.GetMostRecentCampaignIDForSubscriber(ctx, sub.ID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err == nil {
			event, err := s.q.InsertEmailEvent(ctx, gen.InsertEmailEventParams{
				ID:           uuid.New(),
				CampaignID:   campaignID,
				SubscriberID: sub.ID,
				Type:         eventType,
			})
			if err != nil {
				return err
			}
			if err := s.h.DoAction(ctx, hookName, EventPayload{Event: event}); err != nil {
				// Actions are fire-and-react: a handler failure must not fail
				// the suppression flow (the event row is already committed).
				// Log and continue.
				log.Printf("%s action error for subscriber %s: %v", hookName, sub.ID, err)
			}
		}

		if err := s.q.SetSubscriberStatusByID(ctx, gen.SetSubscriberStatusByIDParams{ID: sub.ID, Status: status}); err != nil {
			return err
		}

		if err := s.q.InsertSuppression(ctx, gen.InsertSuppressionParams{
			ID:      uuid.New(),
			OwnerID: sub.OwnerID,
			Email:   sub.Email,
			Reason:  eventType,
		}); err != nil {
			return err
		}
	}

	return nil
}
