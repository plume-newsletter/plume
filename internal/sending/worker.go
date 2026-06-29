package sending

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/plume-newsletter/plume/internal/email"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/render"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

// Worker drains the global campaign_recipient queue: render + send each
// queued recipient through provider, recording state and firing Actions.
// It is system-wide and intentionally NOT owner-scoped — it processes any
// queued row regardless of which admin owns the campaign.
//
// Exactly ONE Worker goroutine must run at a time: ClaimQueuedRecipients is a
// non-atomic read, so concurrent workers would double-send. ponytail:
// single global worker; move to FOR UPDATE SKIP LOCKED if you need multiple.
type Worker struct {
	pool     *pgxpool.Pool
	q        *gen.Queries
	h        *hooks.Hooks
	resolver email.Resolver
	baseURL  string
	batch    int32
}

func NewWorker(pool *pgxpool.Pool, q *gen.Queries, h *hooks.Hooks, resolver email.Resolver, baseURL string, batch int) *Worker {
	return &Worker{pool: pool, q: q, h: h, resolver: resolver, baseURL: baseURL, batch: int32(batch)}
}

// RunOnce claims up to batch queued recipients, renders and sends each, and
// marks the affected campaigns sent (firing campaign.sent once) when their
// queue is drained. It returns the number of recipients processed.
func (w *Worker) RunOnce(ctx context.Context) (int, error) {
	recipients, err := w.q.ClaimQueuedRecipients(ctx, w.batch)
	if err != nil {
		return 0, err
	}
	if len(recipients) == 0 {
		return 0, nil
	}

	affectedCampaigns := map[uuid.UUID]bool{}
	for _, r := range recipients {
		affectedCampaigns[r.CampaignID] = true
		if err := w.sendOne(ctx, r); err != nil {
			// sendOne already recorded the failure on the recipient row; log and
			// continue draining the rest of the batch.
			log.Printf("sending: recipient %s failed: %v", r.ID, err)
		}
	}

	for campaignID := range affectedCampaigns {
		if err := w.maybeFinishCampaign(ctx, campaignID); err != nil {
			log.Printf("sending: finishing campaign %s: %v", campaignID, err)
		}
	}

	return len(recipients), nil
}

func (w *Worker) sendOne(ctx context.Context, r gen.CampaignRecipient) error {
	c, err := w.q.GetCampaignByID(ctx, r.CampaignID)
	if err != nil {
		return w.fail(ctx, r.ID, err)
	}
	b, err := w.q.GetBrandByID(ctx, c.BrandID)
	if err != nil {
		return w.fail(ctx, r.ID, err)
	}
	sub, err := w.q.GetSubscriberByID(ctx, r.SubscriberID)
	if err != nil {
		return w.fail(ctx, r.ID, err)
	}

	suppressed, err := w.q.IsSuppressed(ctx, gen.IsSuppressedParams{OwnerID: sub.OwnerID, Email: sub.Email})
	if err != nil {
		return w.fail(ctx, r.ID, err)
	}
	if suppressed {
		// Hard-bounce/complaint suppression is authoritative regardless of the
		// subscriber's own status: do not send, mark failed, and let the batch
		// continue (no error returned).
		if markErr := w.q.MarkRecipientFailed(ctx, gen.MarkRecipientFailedParams{
			ID:    r.ID,
			Error: pgtype.Text{String: "suppressed", Valid: true},
		}); markErr != nil {
			return markErr
		}
		return nil
	}

	linkRows, err := w.q.ListLinksForCampaign(ctx, r.CampaignID)
	if err != nil {
		return w.fail(ctx, r.ID, err)
	}

	links := make([]render.Link, 0, len(linkRows))
	for _, l := range linkRows {
		links = append(links, render.Link{ID: l.ID, URL: l.Url})
	}

	html, err := render.Render(ctx, w.h, render.Context{
		HTML:        c.HtmlBody,
		BaseURL:     w.baseURL,
		RecipientID: r.ID,
		Links:       links,
	})
	if err != nil {
		return w.fail(ctx, r.ID, err)
	}

	subject := c.Subject
	if r.Subject != "" {
		subject = r.Subject
	}
	msg := email.Message{
		From:     b.FromEmail,
		FromName: b.FromName,
		ReplyTo:  b.ReplyTo,
		To:       sub.Email,
		ToName:   sub.Name,
		Subject:  subject,
		HTML:     html,
		Text:     c.PlainBody,
	}

	provider, err := w.resolver.Provider(ctx)
	if err != nil {
		return w.fail(ctx, r.ID, err)
	}
	if err := provider.Send(ctx, msg); err != nil {
		return w.fail(ctx, r.ID, err)
	}

	return w.q.MarkRecipientSent(ctx, r.ID)
}

func (w *Worker) fail(ctx context.Context, recipientID uuid.UUID, sendErr error) error {
	if markErr := w.q.MarkRecipientFailed(ctx, gen.MarkRecipientFailedParams{
		ID:    recipientID,
		Error: pgtype.Text{String: sendErr.Error(), Valid: true},
	}); markErr != nil {
		return markErr
	}
	return sendErr
}

// maybeFinishCampaign sets campaign status to sent and fires campaign.sent
// exactly once, when no queued recipients remain for it. It checks the
// campaign's current status first so a campaign already marked sent (by an
// earlier RunOnce batch) is never re-transitioned or double-fired.
func (w *Worker) maybeFinishCampaign(ctx context.Context, campaignID uuid.UUID) error {
	c, err := w.q.GetCampaignByID(ctx, campaignID)
	if err != nil {
		return err
	}
	if c.Status == "sent" {
		return nil // already finished by a prior batch; do not re-fire
	}

	remaining, err := w.q.CountQueuedForCampaign(ctx, campaignID)
	if err != nil {
		return err
	}
	if remaining > 0 {
		return nil
	}

	updated, err := w.q.SetCampaignStatusByID(ctx, gen.SetCampaignStatusByIDParams{ID: campaignID, Status: "sent"})
	if err != nil {
		return err
	}
	return w.h.DoAction(ctx, HookCampaignSent, SendingPayload{Campaign: updated})
}

// Start runs RunOnce on a ticker until ctx is canceled, for rate-limited
// background draining of the send queue.
func (w *Worker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := w.RunOnce(ctx); err != nil {
				log.Printf("sending: RunOnce error: %v", err)
			}
		}
	}
}
