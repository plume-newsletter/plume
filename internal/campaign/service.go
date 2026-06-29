package campaign

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/plume-newsletter/plume/internal/blocks"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

var ErrBrandNotFound = errors.New("brand not found for owner")

type CampaignInput struct {
	Subject   string
	HtmlBody  string
	PlainBody string
}

type Service struct{ q *gen.Queries }

func New(q *gen.Queries) *Service { return &Service{q: q} }

func (s *Service) Create(ctx context.Context, owner, brandID uuid.UUID, in CampaignInput) (gen.Campaign, error) {
	if _, err := s.q.GetBrandForOwner(ctx, gen.GetBrandForOwnerParams{ID: brandID, OwnerID: owner}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.Campaign{}, ErrBrandNotFound
		}
		return gen.Campaign{}, err
	}
	return s.q.CreateCampaign(ctx, gen.CreateCampaignParams{
		ID: uuid.New(), OwnerID: owner, BrandID: brandID,
		Subject: in.Subject, HtmlBody: in.HtmlBody, PlainBody: in.PlainBody,
	})
}

func (s *Service) List(ctx context.Context, owner uuid.UUID) ([]gen.Campaign, error) {
	return s.q.ListCampaignsByOwner(ctx, owner)
}

func (s *Service) Get(ctx context.Context, owner, id uuid.UUID) (gen.Campaign, error) {
	return s.q.GetCampaignForOwner(ctx, gen.GetCampaignForOwnerParams{ID: id, OwnerID: owner})
}

// Update renders bodyJSON to email-safe HTML + plain text and stores all three.
func (s *Service) Update(ctx context.Context, owner, id uuid.UUID, subject string, bodyJSON []byte) (gen.Campaign, error) {
	html, plain, err := blocks.RenderJSON(bodyJSON)
	if err != nil {
		return gen.Campaign{}, err
	}
	if len(bodyJSON) == 0 {
		bodyJSON = []byte("[]")
	}
	return s.q.UpdateCampaign(ctx, gen.UpdateCampaignParams{
		ID: id, OwnerID: owner,
		Subject: subject, HtmlBody: html, PlainBody: plain, BodyJson: bodyJSON,
	})
}

func (s *Service) Delete(ctx context.Context, owner, id uuid.UUID) error {
	return s.q.DeleteCampaign(ctx, gen.DeleteCampaignParams{ID: id, OwnerID: owner})
}
