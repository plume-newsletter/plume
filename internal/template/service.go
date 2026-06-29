// Package template manages reusable email layouts: global prebuilt starters
// plus user-saved designs. "Use" spins a template's blocks into a draft campaign.
package template

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/plume-newsletter/plume/internal/campaign"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

var ErrInvalid = errors.New("invalid")
var ErrNotFound = errors.New("template not found")

type Template struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Category  string          `json:"category"`
	BodyJSON  json.RawMessage `json:"bodyJson"`
	Prebuilt  bool            `json:"prebuilt"`
	CreatedAt string          `json:"createdAt"`
}

type Service struct {
	q         *gen.Queries
	campaigns *campaign.Service
}

func New(q *gen.Queries, campaigns *campaign.Service) *Service {
	return &Service{q: q, campaigns: campaigns}
}

var validCategory = map[string]bool{"Newsletter": true, "Product": true, "Promo": true, "Transactional": true}

func toTemplate(t gen.Template) Template {
	body := t.BodyJson
	if len(body) == 0 {
		body = []byte("[]")
	}
	return Template{
		ID: t.ID.String(), Name: t.Name, Category: t.Category,
		BodyJSON: json.RawMessage(body), Prebuilt: t.Prebuilt,
		CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (s *Service) List(ctx context.Context, owner uuid.UUID, category string) ([]Template, error) {
	var rows []gen.Template
	var err error
	if category == "" {
		rows, err = s.q.ListTemplatesForOwner(ctx, owner)
	} else {
		rows, err = s.q.ListTemplatesForOwnerByCategory(ctx, gen.ListTemplatesForOwnerByCategoryParams{OwnerID: owner, Category: category})
	}
	if err != nil {
		return nil, err
	}
	out := make([]Template, 0, len(rows))
	for _, r := range rows {
		out = append(out, toTemplate(r))
	}
	return out, nil
}

func (s *Service) Create(ctx context.Context, owner uuid.UUID, name, category string, bodyJSON []byte) (Template, error) {
	if name == "" {
		return Template{}, ErrInvalid
	}
	if !validCategory[category] {
		category = "Newsletter"
	}
	if len(bodyJSON) == 0 {
		bodyJSON = []byte("[]")
	}
	row, err := s.q.CreateTemplate(ctx, gen.CreateTemplateParams{
		ID: uuid.New(), OwnerID: owner, Name: name, Category: category, BodyJson: bodyJSON,
	})
	if err != nil {
		return Template{}, err
	}
	return toTemplate(row), nil
}

func (s *Service) Delete(ctx context.Context, owner, id uuid.UUID) error {
	return s.q.DeleteTemplateForOwner(ctx, gen.DeleteTemplateForOwnerParams{ID: id, OwnerID: owner})
}

func (s *Service) Use(ctx context.Context, owner, templateID, brandID uuid.UUID, subject string) (uuid.UUID, error) {
	if subject == "" {
		return uuid.Nil, ErrInvalid
	}
	tpl, err := s.q.GetTemplateForUse(ctx, gen.GetTemplateForUseParams{ID: templateID, OwnerID: owner})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}
	c, err := s.campaigns.Create(ctx, owner, brandID, campaign.CampaignInput{Subject: subject})
	if err != nil {
		return uuid.Nil, err // includes campaign.ErrBrandNotFound for a foreign/bad brand
	}
	if _, err := s.campaigns.Update(ctx, owner, c.ID, subject, tpl.BodyJson); err != nil {
		return uuid.Nil, err
	}
	return c.ID, nil
}
