// Package signupform manages owner signup forms (CRUD) and exposes a public
// fetch used to render the hosted landing page.
package signupform

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

var ErrNotFound = errors.New("form not found")
var ErrInvalid = errors.New("invalid form")

type Form struct {
	ID          string `json:"id"`
	ListID      string `json:"listId"`
	Name        string `json:"name"`
	Heading     string `json:"heading"`
	Description string `json:"description"`
	ButtonText  string `json:"buttonText"`
	CreatedAt   string `json:"createdAt"`
}
type FormInput struct {
	ListID      uuid.UUID
	Name        string
	Heading     string
	Description string
	ButtonText  string
}

type Service struct{ q *gen.Queries }

func New(q *gen.Queries) *Service { return &Service{q: q} }

func toForm(r gen.SignupForm) Form {
	return Form{
		ID: r.ID.String(), ListID: r.ListID.String(), Name: r.Name,
		Heading: r.Heading, Description: r.Description, ButtonText: r.ButtonText,
		CreatedAt: r.CreatedAt.Format(time.RFC3339),
	}
}

func (in FormInput) normalized() (FormInput, error) {
	if in.Name == "" || in.ListID == uuid.Nil {
		return in, ErrInvalid
	}
	if in.ButtonText == "" {
		in.ButtonText = "Subscribe"
	}
	return in, nil
}

func (s *Service) Create(ctx context.Context, owner uuid.UUID, in FormInput) (Form, error) {
	in, err := in.normalized()
	if err != nil {
		return Form{}, err
	}
	r, err := s.q.CreateSignupForm(ctx, gen.CreateSignupFormParams{
		ID: uuid.New(), OwnerID: owner, ListID: in.ListID, Name: in.Name,
		Heading: in.Heading, Description: in.Description, ButtonText: in.ButtonText,
	})
	if err != nil {
		return Form{}, err
	}
	return toForm(r), nil
}

func (s *Service) List(ctx context.Context, owner uuid.UUID) ([]Form, error) {
	rows, err := s.q.ListSignupFormsByOwner(ctx, owner)
	if err != nil {
		return nil, err
	}
	out := make([]Form, 0, len(rows))
	for _, r := range rows {
		out = append(out, toForm(r))
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, owner, id uuid.UUID) (Form, error) {
	r, err := s.q.GetSignupFormForOwner(ctx, gen.GetSignupFormForOwnerParams{ID: id, OwnerID: owner})
	if errors.Is(err, pgx.ErrNoRows) {
		return Form{}, ErrNotFound
	}
	if err != nil {
		return Form{}, err
	}
	return toForm(r), nil
}

func (s *Service) Update(ctx context.Context, owner, id uuid.UUID, in FormInput) (Form, error) {
	in, err := in.normalized()
	if err != nil {
		return Form{}, err
	}
	r, err := s.q.UpdateSignupForm(ctx, gen.UpdateSignupFormParams{
		ID: id, OwnerID: owner, ListID: in.ListID, Name: in.Name,
		Heading: in.Heading, Description: in.Description, ButtonText: in.ButtonText,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Form{}, ErrNotFound
	}
	if err != nil {
		return Form{}, err
	}
	return toForm(r), nil
}

func (s *Service) Delete(ctx context.Context, owner, id uuid.UUID) error {
	return s.q.DeleteSignupForm(ctx, gen.DeleteSignupFormParams{ID: id, OwnerID: owner})
}

func (s *Service) GetPublic(ctx context.Context, id uuid.UUID) (Form, error) {
	r, err := s.q.GetSignupForm(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Form{}, ErrNotFound
	}
	if err != nil {
		return Form{}, err
	}
	return toForm(r), nil
}
