package analytics

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

// Suppression is one suppressed address (bounced/complained/unsubscribed).
type Suppression struct {
	Email  string `json:"email"`
	Reason string `json:"reason"`
	Date   string `json:"date"`
}

// Deliverability is the sender-health payload for GET /api/analytics/deliverability.
type Deliverability struct {
	Sent          int           `json:"sent"`
	Bounces       int           `json:"bounces"`
	Complaints    int           `json:"complaints"`
	BounceRate    float64       `json:"bounceRate"`
	ComplaintRate float64       `json:"complaintRate"`
	Suppressed    int           `json:"suppressed"`
	Suppressions  []Suppression `json:"suppressions"`
}

// Deliverability returns bounce/complaint rates over the window plus the
// current suppression list. Bounce/complaint counts come from email_event
// (same source as the engagement rates); the suppression_entry table is the
// authoritative list of who is currently suppressed and why.
func (s *Service) Deliverability(ctx context.Context, owner uuid.UUID, windowDays int) (Deliverability, error) {
	if windowDays != 90 {
		windowDays = 30
	}
	since := time.Now().AddDate(0, 0, -windowDays)
	var d Deliverability

	sent, err := s.q.CountSentForOwnerSince(ctx, gen.CountSentForOwnerSinceParams{OwnerID: owner, SentAt: since})
	if err != nil {
		return d, err
	}
	bounces, err := s.q.CountEventsForOwnerSince(ctx, gen.CountEventsForOwnerSinceParams{OwnerID: owner, Type: "bounce", CreatedAt: since})
	if err != nil {
		return d, err
	}
	complaints, err := s.q.CountEventsForOwnerSince(ctx, gen.CountEventsForOwnerSinceParams{OwnerID: owner, Type: "complaint", CreatedAt: since})
	if err != nil {
		return d, err
	}
	d.Sent = int(sent)
	d.Bounces = int(bounces)
	d.Complaints = int(complaints)
	d.BounceRate = rate(bounces, sent)
	d.ComplaintRate = rate(complaints, sent)

	total, err := s.q.CountSuppressionsForOwner(ctx, owner)
	if err != nil {
		return d, err
	}
	d.Suppressed = int(total)

	rows, err := s.q.RecentSuppressionsForOwner(ctx, owner)
	if err != nil {
		return d, err
	}
	d.Suppressions = make([]Suppression, 0, len(rows))
	for _, r := range rows {
		reason := r.Reason
		if reason == "" {
			reason = "unknown"
		}
		d.Suppressions = append(d.Suppressions, Suppression{
			Email:  r.Email,
			Reason: reason,
			Date:   r.CreatedAt.Format("2006-01-02"),
		})
	}
	return d, nil
}
