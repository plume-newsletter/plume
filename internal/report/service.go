// Package report aggregates a campaign's recipient and email_event rows into
// a per-campaign summary (recipients/sent counts plus open/click/bounce/
// complaint/unsubscribe totals).
package report

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

// ErrNotFound is returned when the campaign does not exist or is not owned
// by the caller.
var ErrNotFound = errors.New("campaign not found")

// EventStats holds a total count and a count of distinct subscribers for one
// email_event type.
type EventStats struct {
	Total  int `json:"total"`
	Unique int `json:"unique"`
}

// Summary is the per-campaign report payload.
type Summary struct {
	Recipients   int        `json:"recipients"`
	Sent         int        `json:"sent"`
	Opens        EventStats `json:"opens"`
	Clicks       EventStats `json:"clicks"`
	Bounces      int        `json:"bounces"`
	Complaints   int        `json:"complaints"`
	Unsubscribes int        `json:"unsubscribes"`
}

type Service struct{ q *gen.Queries }

func New(q *gen.Queries) *Service { return &Service{q: q} }

// Campaign verifies the campaign is owned by owner, then assembles its report
// Summary from the recipient and email_event aggregation queries.
func (s *Service) Campaign(ctx context.Context, owner, campaignID uuid.UUID) (Summary, error) {
	if _, err := s.q.GetCampaignForOwner(ctx, gen.GetCampaignForOwnerParams{ID: campaignID, OwnerID: owner}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Summary{}, ErrNotFound
		}
		return Summary{}, err
	}

	recipients, err := s.q.CountRecipients(ctx, campaignID)
	if err != nil {
		return Summary{}, err
	}
	sent, err := s.q.CountRecipientsByStatus(ctx, gen.CountRecipientsByStatusParams{CampaignID: campaignID, Status: "sent"})
	if err != nil {
		return Summary{}, err
	}

	opensTotal, err := s.q.CountEventsByType(ctx, gen.CountEventsByTypeParams{CampaignID: campaignID, Type: "open"})
	if err != nil {
		return Summary{}, err
	}
	opensUnique, err := s.q.CountDistinctSubscribersByEventType(ctx, gen.CountDistinctSubscribersByEventTypeParams{CampaignID: campaignID, Type: "open"})
	if err != nil {
		return Summary{}, err
	}

	clicksTotal, err := s.q.CountEventsByType(ctx, gen.CountEventsByTypeParams{CampaignID: campaignID, Type: "click"})
	if err != nil {
		return Summary{}, err
	}
	clicksUnique, err := s.q.CountDistinctSubscribersByEventType(ctx, gen.CountDistinctSubscribersByEventTypeParams{CampaignID: campaignID, Type: "click"})
	if err != nil {
		return Summary{}, err
	}

	bounces, err := s.q.CountEventsByType(ctx, gen.CountEventsByTypeParams{CampaignID: campaignID, Type: "bounce"})
	if err != nil {
		return Summary{}, err
	}
	complaints, err := s.q.CountEventsByType(ctx, gen.CountEventsByTypeParams{CampaignID: campaignID, Type: "complaint"})
	if err != nil {
		return Summary{}, err
	}
	unsubscribes, err := s.q.CountEventsByType(ctx, gen.CountEventsByTypeParams{CampaignID: campaignID, Type: "unsubscribe"})
	if err != nil {
		return Summary{}, err
	}

	return Summary{
		Recipients:   int(recipients),
		Sent:         int(sent),
		Opens:        EventStats{Total: int(opensTotal), Unique: int(opensUnique)},
		Clicks:       EventStats{Total: int(clicksTotal), Unique: int(clicksUnique)},
		Bounces:      int(bounces),
		Complaints:   int(complaints),
		Unsubscribes: int(unsubscribes),
	}, nil
}
