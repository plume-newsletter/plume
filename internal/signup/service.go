// Package signup implements the public double opt-in flow: a visitor
// subscribes to a list, a transactional confirmation email is sent directly
// via the email.Provider (never through the bulk campaign worker), and a
// confirm click activates the subscriber.
package signup

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/plume-newsletter/plume/internal/email"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
)

// HookConfirmed fires (fire-and-react) once a pending subscriber confirms.
const HookConfirmed = "subscriber.confirmed"

// ErrListNotFound is returned by Subscribe when listID does not resolve.
var ErrListNotFound = errors.New("list not found")

// ConfirmedPayload is the Action payload for subscriber.confirmed.
type ConfirmedPayload struct{ Subscriber gen.Subscriber }

// Service implements the public subscribe/confirm flow.
type Service struct {
	q        *gen.Queries
	h        *hooks.Hooks
	resolver email.Resolver
	baseURL  string
}

// New builds a Service. resolver and baseURL are the same values the
// sending.Worker uses — the confirmation email is transactional and shares
// only the email.Resolver seam with bulk sending, never the worker/queue.
func New(q *gen.Queries, h *hooks.Hooks, resolver email.Resolver, baseURL string) *Service {
	return &Service{q: q, h: h, resolver: resolver, baseURL: baseURL}
}

func normalizeEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

// Subscribe resolves listID (-> owner_id, brand_id), dedupes by (list, email),
// and ensures a pending subscriber exists, (re)sending the confirmation
// email. An already-active subscriber is a silent no-op. The method always
// returns nil on success (regardless of which branch ran) so the public
// handler's response never leaks subscription membership. Unknown list ->
// ErrListNotFound. Real DB errors propagate.
func (s *Service) Subscribe(ctx context.Context, listID uuid.UUID, addr, name string) error {
	l, err := s.q.GetListByID(ctx, listID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrListNotFound
		}
		return err
	}
	addr = normalizeEmail(addr)

	existing, err := s.q.GetSubscriberInListByEmail(ctx, gen.GetSubscriberInListByEmailParams{ListID: listID, Email: addr})
	switch {
	case err == nil:
		if existing.Status == "active" {
			return nil // already subscribed: silent no-op, no leak
		}
		// pending (or any other non-active status): resend confirmation below.
	case errors.Is(err, pgx.ErrNoRows):
		existing, err = s.q.CreateSubscriber(ctx, gen.CreateSubscriberParams{
			ID: uuid.New(), OwnerID: l.OwnerID, ListID: listID, Email: addr, Name: name, Status: "pending",
		})
		if err != nil {
			return err
		}
		if err := s.h.DoAction(ctx, subscriber.HookSubscriberAdded, subscriber.AddedPayload{Subscriber: existing}); err != nil {
			// Actions are fire-and-react: a handler failure must not fail the
			// subscribe (the row is already committed). Log and continue.
			log.Printf("subscriber.added action error for %s: %v", existing.Email, err)
		}
	default:
		return err
	}

	brandRow, err := s.q.GetBrandByID(ctx, l.BrandID)
	if err != nil {
		return err
	}
	if err := s.sendConfirmEmail(ctx, brandRow, existing); err != nil {
		// The confirmation email failing must not surface to the public
		// caller (leak-safe, generic response); re-subscribing resends it.
		log.Printf("signup: send confirm email to %s: %v", existing.Email, err)
	}
	return nil
}

func (s *Service) sendConfirmEmail(ctx context.Context, b gen.Brand, sub gen.Subscriber) error {
	provider, err := s.resolver.Provider(ctx)
	if err != nil {
		// Resolver failures must not fail Subscribe (transactional,
		// fire-and-forget): log and let the caller's no-op path proceed; the
		// pending subscriber row already exists and re-subscribing resends it.
		log.Printf("signup: resolve provider for %s: %v", sub.Email, err)
		return nil
	}
	link := fmt.Sprintf("%s/confirm/%s", s.baseURL, sub.ID.String())
	msg := email.Message{
		From:     b.FromEmail,
		FromName: b.FromName,
		ReplyTo:  b.ReplyTo,
		To:       sub.Email,
		Subject:  "Confirm your subscription",
		// SECURITY: only the trusted confirm URL (baseURL + UUID) is interpolated here. Do NOT interpolate subscriber-supplied name/email into this HTML without html.EscapeString — it would be an injection vector.
		HTML: fmt.Sprintf(`<p>Please confirm your subscription by clicking the link below.</p><p><a href="%s">%s</a></p>`, link, link),
	}
	return provider.Send(ctx, msg)
}

// Confirm activates a pending subscriber and fires subscriber.confirmed.
// Unknown subscriber or one that is not pending (e.g. already active) is a
// silent idempotent no-op, so the public handler never leaks state.
func (s *Service) Confirm(ctx context.Context, subscriberID uuid.UUID) error {
	sub, err := s.q.GetSubscriberByID(ctx, subscriberID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if sub.Status != "pending" {
		return nil
	}
	if err := s.q.SetSubscriberStatusByID(ctx, gen.SetSubscriberStatusByIDParams{ID: sub.ID, Status: "active"}); err != nil {
		return err
	}
	// A completed double-opt-in is fresh consent: it overrides any prior
	// unsubscribe/bounce/complaint suppression recorded for this address.
	if err := s.q.DeleteSuppressionByOwnerEmail(ctx, gen.DeleteSuppressionByOwnerEmailParams{
		OwnerID: sub.OwnerID, Email: sub.Email,
	}); err != nil {
		return err
	}
	sub.Status = "active"
	if err := s.h.DoAction(ctx, HookConfirmed, ConfirmedPayload{Subscriber: sub}); err != nil {
		// Fire-and-react: a handler failure must not fail the confirm (the
		// status is already committed). Log and continue.
		log.Printf("subscriber.confirmed action error for subscriber %s: %v", sub.ID, err)
	}
	return nil
}
