package brand

import (
	"context"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

type BrandInput struct {
	Name      string
	FromName  string
	FromEmail string
	ReplyTo   string
}

type Service struct{ q *gen.Queries }

func New(q *gen.Queries) *Service { return &Service{q: q} }

func (s *Service) Create(ctx context.Context, owner uuid.UUID, in BrandInput) (gen.Brand, error) {
	return s.q.CreateBrand(ctx, gen.CreateBrandParams{
		ID:        uuid.New(),
		OwnerID:   owner,
		Name:      in.Name,
		FromName:  in.FromName,
		FromEmail: in.FromEmail,
		ReplyTo:   in.ReplyTo,
	})
}

func (s *Service) List(ctx context.Context, owner uuid.UUID) ([]gen.Brand, error) {
	return s.q.ListBrandsByOwner(ctx, owner)
}

func (s *Service) Get(ctx context.Context, owner, id uuid.UUID) (gen.Brand, error) {
	return s.q.GetBrandForOwner(ctx, gen.GetBrandForOwnerParams{ID: id, OwnerID: owner})
}

func (s *Service) Update(ctx context.Context, owner, id uuid.UUID, in BrandInput) (gen.Brand, error) {
	return s.q.UpdateBrand(ctx, gen.UpdateBrandParams{
		ID: id, OwnerID: owner,
		Name: in.Name, FromName: in.FromName, FromEmail: in.FromEmail, ReplyTo: in.ReplyTo,
	})
}

func (s *Service) Delete(ctx context.Context, owner, id uuid.UUID) error {
	return s.q.DeleteBrand(ctx, gen.DeleteBrandParams{ID: id, OwnerID: owner})
}
