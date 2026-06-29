// Package abtest runs subject-line A/B tests: a split of a list is sent two
// subjects via the normal send pipeline (per-recipient subject override), and
// the owner finalizes by sending the winning subject to the holdout.
package abtest

import (
	"context"
	"errors"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/plume-newsletter/plume/internal/render"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

var ErrNotFound = errors.New("ab test not found")
var ErrInvalid = errors.New("invalid")
var ErrState = errors.New("wrong state")

// Test is the JSON-friendly view of a gen.AbTest.
type Test struct {
	ID          string `json:"id"`
	CampaignID  string `json:"campaignId"`
	ListID      string `json:"listId"`
	SubjectA    string `json:"subjectA"`
	SubjectB    string `json:"subjectB"`
	TestPercent int    `json:"testPercent"`
	Status      string `json:"status"`
	Winner      string `json:"winner"`
	CreatedAt   string `json:"createdAt"`
}

// VariantResult holds per-variant aggregated metrics.
type VariantResult struct {
	Variant   string  `json:"variant"`
	Subject   string  `json:"subject"`
	Sent      int     `json:"sent"`
	OpenRate  float64 `json:"openRate"`
	ClickRate float64 `json:"clickRate"`
}

// Results holds the overall test status plus both variant results.
type Results struct {
	Status   string          `json:"status"`
	Winner   string          `json:"winner"`
	Variants []VariantResult `json:"variants"`
}

// Input is the payload for creating a new A/B test.
type Input struct {
	CampaignID, ListID uuid.UUID
	SubjectA, SubjectB string
	TestPercent        int
}

// Service orchestrates A/B test lifecycle.
type Service struct{ q *gen.Queries }

// New returns a new Service.
func New(q *gen.Queries) *Service { return &Service{q: q} }

func clampPercent(p int) int {
	if p < 5 {
		return 5
	}
	if p > 50 {
		return 50
	}
	return p
}

func toTest(t gen.AbTest) Test {
	return Test{
		ID: t.ID.String(), CampaignID: t.CampaignID.String(), ListID: t.ListID.String(),
		SubjectA: t.SubjectA, SubjectB: t.SubjectB, TestPercent: int(t.TestPercent),
		Status: t.Status, Winner: t.Winner, CreatedAt: t.CreatedAt.Format(time.RFC3339),
	}
}

func rate(num, den int64) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}

// Create validates and persists a new A/B test in draft status.
func (s *Service) Create(ctx context.Context, owner uuid.UUID, in Input) (Test, error) {
	if in.SubjectA == "" || in.SubjectB == "" {
		return Test{}, ErrInvalid
	}
	c, err := s.q.GetCampaignForOwner(ctx, gen.GetCampaignForOwnerParams{ID: in.CampaignID, OwnerID: owner})
	if err != nil || c.Status != "draft" {
		return Test{}, ErrInvalid // must be an owned, still-draft campaign
	}
	if _, err := s.q.GetListForOwner(ctx, gen.GetListForOwnerParams{ID: in.ListID, OwnerID: owner}); err != nil {
		return Test{}, ErrInvalid
	}
	row, err := s.q.CreateABTest(ctx, gen.CreateABTestParams{
		ID: uuid.New(), OwnerID: owner, CampaignID: in.CampaignID, ListID: in.ListID,
		SubjectA: in.SubjectA, SubjectB: in.SubjectB, TestPercent: int32(clampPercent(in.TestPercent)),
	})
	if err != nil {
		return Test{}, err
	}
	return toTest(row), nil
}

// List returns all A/B tests for the given owner.
func (s *Service) List(ctx context.Context, owner uuid.UUID) ([]Test, error) {
	rows, err := s.q.ListABTestsByOwner(ctx, owner)
	if err != nil {
		return nil, err
	}
	out := make([]Test, 0, len(rows))
	for _, r := range rows {
		out = append(out, toTest(r))
	}
	return out, nil
}

func (s *Service) get(ctx context.Context, owner, id uuid.UUID) (gen.AbTest, error) {
	row, err := s.q.GetABTestForOwner(ctx, gen.GetABTestForOwnerParams{ID: id, OwnerID: owner})
	if errors.Is(err, pgx.ErrNoRows) {
		return gen.AbTest{}, ErrNotFound
	}
	return row, err
}

// Get returns a single A/B test by id (scoped to owner).
func (s *Service) Get(ctx context.Context, owner, id uuid.UUID) (Test, error) {
	row, err := s.get(ctx, owner, id)
	if err != nil {
		return Test{}, err
	}
	return toTest(row), nil
}

// Delete removes a draft A/B test.
func (s *Service) Delete(ctx context.Context, owner, id uuid.UUID) error {
	return s.q.DeleteABTest(ctx, gen.DeleteABTestParams{ID: id, OwnerID: owner})
}

// Start enqueues the test split. It takes max(2, ceil(N*pct/100)) subscribers
// ordered by UUID, splits them A=first half / B=second half (odd remainder → A),
// creates queued campaign_recipient rows, and marks the campaign queued + test running.
// Requires the campaign to be in draft status.
func (s *Service) Start(ctx context.Context, owner, id uuid.UUID) error {
	t, err := s.get(ctx, owner, id)
	if err != nil {
		return err
	}
	if t.Status != "draft" {
		return ErrState
	}
	c, err := s.q.GetCampaignForOwner(ctx, gen.GetCampaignForOwnerParams{ID: t.CampaignID, OwnerID: owner})
	if err != nil {
		return ErrInvalid
	}
	// The campaign must still be draft. Starting flips it to 'queued', so a
	// second test on the same campaign (or a re-send of an already-sent one)
	// can never queue overlapping recipients — closes the double-send vector.
	if c.Status != "draft" {
		return ErrState
	}
	for _, url := range render.ExtractLinks(c.HtmlBody) {
		if _, err := s.q.CreateLink(ctx, gen.CreateLinkParams{ID: uuid.New(), CampaignID: t.CampaignID, Url: url}); err != nil {
			return err
		}
	}
	ids, err := s.q.ListActiveSubscriberIDsInList(ctx, t.ListID)
	if err != nil {
		return err
	}
	if len(ids) < 2 {
		return ErrInvalid
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	k := int(math.Ceil(float64(len(ids)) * float64(t.TestPercent) / 100.0))
	if k < 2 {
		k = 2
	}
	if k > len(ids) {
		k = len(ids)
	}
	test := ids[:k]
	half := (k + 1) / 2 // odd → A gets more
	for i, sid := range test {
		variant, subject := "a", t.SubjectA
		if i >= half {
			variant, subject = "b", t.SubjectB
		}
		if _, err := s.q.CreateRecipientVariant(ctx, gen.CreateRecipientVariantParams{
			ID: uuid.New(), CampaignID: t.CampaignID, SubscriberID: sid, Variant: variant, Subject: subject,
		}); err != nil {
			return err
		}
	}
	if _, err := s.q.SetCampaignStatusByID(ctx, gen.SetCampaignStatusByIDParams{ID: t.CampaignID, Status: "queued"}); err != nil {
		return err
	}
	return s.q.UpdateABTestStatus(ctx, gen.UpdateABTestStatusParams{ID: id, Status: "running"})
}

func (s *Service) variantResult(ctx context.Context, campaignID uuid.UUID, variant, subject string) (VariantResult, error) {
	sent, err := s.q.CountVariantRecipients(ctx, gen.CountVariantRecipientsParams{CampaignID: campaignID, Variant: variant})
	if err != nil {
		return VariantResult{}, err
	}
	ev, err := s.q.CountVariantEvents(ctx, gen.CountVariantEventsParams{CampaignID: campaignID, Variant: variant})
	if err != nil {
		return VariantResult{}, err
	}
	return VariantResult{
		Variant: variant, Subject: subject, Sent: int(sent),
		OpenRate: rate(ev.Opens, sent), ClickRate: rate(ev.Clicks, sent),
	}, nil
}

// Results aggregates per-variant open/click rates for the test.
func (s *Service) Results(ctx context.Context, owner, id uuid.UUID) (Results, error) {
	t, err := s.get(ctx, owner, id)
	if err != nil {
		return Results{}, err
	}
	a, err := s.variantResult(ctx, t.CampaignID, "a", t.SubjectA)
	if err != nil {
		return Results{}, err
	}
	b, err := s.variantResult(ctx, t.CampaignID, "b", t.SubjectB)
	if err != nil {
		return Results{}, err
	}
	return Results{Status: t.Status, Winner: t.Winner, Variants: []VariantResult{a, b}}, nil
}

// SendWinner sends the winning subject to the holdout (active subscribers not yet
// in the test), marks the campaign queued, and marks the A/B test complete with winner.
// Requires the test to be running and winner to be "a" or "b".
func (s *Service) SendWinner(ctx context.Context, owner, id uuid.UUID, winner string) error {
	if winner != "a" && winner != "b" {
		return ErrInvalid
	}
	t, err := s.get(ctx, owner, id)
	if err != nil {
		return err
	}
	if t.Status != "running" {
		return ErrState
	}
	subject := t.SubjectA
	if winner == "b" {
		subject = t.SubjectB
	}
	active, err := s.q.ListActiveSubscriberIDsInList(ctx, t.ListID)
	if err != nil {
		return err
	}
	existing, err := s.q.ListRecipientSubscriberIDs(ctx, t.CampaignID)
	if err != nil {
		return err
	}
	seen := make(map[uuid.UUID]bool, len(existing))
	for _, sid := range existing {
		seen[sid] = true
	}
	for _, sid := range active {
		if seen[sid] {
			continue
		}
		if _, err := s.q.CreateRecipientVariant(ctx, gen.CreateRecipientVariantParams{
			ID: uuid.New(), CampaignID: t.CampaignID, SubscriberID: sid, Variant: "", Subject: subject,
		}); err != nil {
			return err
		}
	}
	if _, err := s.q.SetCampaignStatusByID(ctx, gen.SetCampaignStatusByIDParams{ID: t.CampaignID, Status: "queued"}); err != nil {
		return err
	}
	return s.q.SetABTestWinner(ctx, gen.SetABTestWinnerParams{ID: id, Winner: winner})
}
