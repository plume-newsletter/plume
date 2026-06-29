package list

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

var ErrBrandNotFound = errors.New("brand not found for owner")

type Service struct{ q *gen.Queries }

func New(q *gen.Queries) *Service { return &Service{q: q} }

func (s *Service) Create(ctx context.Context, owner, brandID uuid.UUID, name string) (gen.List, error) {
	if _, err := s.q.GetBrandForOwner(ctx, gen.GetBrandForOwnerParams{ID: brandID, OwnerID: owner}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.List{}, ErrBrandNotFound
		}
		return gen.List{}, err
	}
	return s.q.CreateList(ctx, gen.CreateListParams{
		ID: uuid.New(), OwnerID: owner, BrandID: brandID, Name: name,
	})
}

func (s *Service) List(ctx context.Context, owner uuid.UUID) ([]gen.List, error) {
	return s.q.ListListsByOwner(ctx, owner)
}

func (s *Service) Get(ctx context.Context, owner, id uuid.UUID) (gen.List, error) {
	return s.q.GetListForOwner(ctx, gen.GetListForOwnerParams{ID: id, OwnerID: owner})
}

func (s *Service) Update(ctx context.Context, owner, id uuid.UUID, name string) (gen.List, error) {
	return s.q.UpdateList(ctx, gen.UpdateListParams{ID: id, OwnerID: owner, Name: name})
}

func (s *Service) Delete(ctx context.Context, owner, id uuid.UUID) error {
	return s.q.DeleteList(ctx, gen.DeleteListParams{ID: id, OwnerID: owner})
}
