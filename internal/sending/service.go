// Package sending enqueues a campaign send to a list and drives the
// background worker that renders and delivers each queued recipient.
package sending

import (
	"context"
	"errors"
	"log"
	"net/mail"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/plume-newsletter/plume/internal/email"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/render"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

const (
	HookCampaignSending = "campaign.sending"
	HookCampaignSent    = "campaign.sent"
)

var (
	// ErrCampaignNotFound is returned when the campaign id is not owned by owner.
	ErrCampaignNotFound = errors.New("campaign not found for owner")
	// ErrListNotFound is returned when the list id is not owned by owner.
	ErrListNotFound = errors.New("list not found for owner")
	// ErrAlreadyQueued is returned when Enqueue is called on a campaign that
	// is not in 'draft' status, to guard against double-sending the list.
	ErrAlreadyQueued = errors.New("campaign already queued or sent")
	// ErrBadEmail is returned by SendTest when the address does not parse.
	ErrBadEmail = errors.New("invalid email address")
)

// SendingPayload is the Action payload for campaign.sending / campaign.sent.
type SendingPayload struct {
	Campaign gen.Campaign
}

type Service struct {
	pool     *pgxpool.Pool
	q        *gen.Queries
	h        *hooks.Hooks
	resolver email.Resolver
}

func New(pool *pgxpool.Pool, q *gen.Queries, h *hooks.Hooks, resolver email.Resolver) *Service {
	return &Service{pool: pool, q: q, h: h, resolver: resolver}
}

// SendTest delivers the campaign's currently-saved body to a single address,
// for the composer's "Send test" action. It is owner-scoped and transactional:
// it sends the stored HtmlBody/PlainBody/Subject as-is (no open-pixel,
// unsubscribe, or click rewrite — those are per-recipient and undesirable in a
// test) and never mutates campaign status or the send queue. Works on a
// campaign in any status.
func (s *Service) SendTest(ctx context.Context, owner, campaignID uuid.UUID, addr string) error {
	parsed, err := mail.ParseAddress(addr)
	if err != nil {
		return ErrBadEmail
	}
	c, err := s.q.GetCampaignForOwner(ctx, gen.GetCampaignForOwnerParams{ID: campaignID, OwnerID: owner})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCampaignNotFound
		}
		return err
	}
	b, err := s.q.GetBrandByID(ctx, c.BrandID)
	if err != nil {
		return err
	}
	provider, err := s.resolver.Provider(ctx)
	if err != nil {
		return err
	}
	subject := c.Subject
	if subject == "" {
		subject = "(no subject)"
	}
	return provider.Send(ctx, email.Message{
		From:     b.FromEmail,
		FromName: b.FromName,
		ReplyTo:  b.ReplyTo,
		To:       parsed.Address,
		Subject:  "[Test] " + subject,
		HTML:     c.HtmlBody,
		Text:     c.PlainBody,
	})
}

// Enqueue verifies the campaign and list are owned by owner, extracts links
// from the campaign's HTML body into the link table, inserts one queued
// campaign_recipient per active subscriber in the list, marks the campaign
// queued, fires campaign.sending, and returns the recipient count.
func (s *Service) Enqueue(ctx context.Context, owner, campaignID, listID uuid.UUID) (int, error) {
	c, err := s.q.GetCampaignForOwner(ctx, gen.GetCampaignForOwnerParams{ID: campaignID, OwnerID: owner})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrCampaignNotFound
		}
		return 0, err
	}
	if _, err := s.q.GetListForOwner(ctx, gen.GetListForOwnerParams{ID: listID, OwnerID: owner}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrListNotFound
		}
		return 0, err
	}
	if c.Status != "draft" {
		return 0, ErrAlreadyQueued
	}

	for _, url := range render.ExtractLinks(c.HtmlBody) {
		if _, err := s.q.CreateLink(ctx, gen.CreateLinkParams{ID: uuid.New(), CampaignID: campaignID, Url: url}); err != nil {
			return 0, err
		}
	}

	subscriberIDs, err := s.q.ListActiveSubscriberIDsInList(ctx, listID)
	if err != nil {
		return 0, err
	}
	for _, subID := range subscriberIDs {
		if _, err := s.q.CreateRecipient(ctx, gen.CreateRecipientParams{
			ID: uuid.New(), CampaignID: campaignID, SubscriberID: subID,
		}); err != nil {
			return 0, err
		}
	}

	updated, err := s.q.SetCampaignStatusByID(ctx, gen.SetCampaignStatusByIDParams{ID: campaignID, Status: "queued"})
	if err != nil {
		return 0, err
	}

	if err := s.h.DoAction(ctx, HookCampaignSending, SendingPayload{Campaign: updated}); err != nil {
		// Actions are fire-and-react: a handler failure must not fail the send
		// or corrupt the result (recipients/links/status are already committed).
		// Log and continue.
		log.Printf("campaign.sending action error for campaign %s: %v", updated.ID, err)
	}

	return len(subscriberIDs), nil
}
