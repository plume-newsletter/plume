package list_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestCreateListRequiresOwnedBrand(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()

	b, err := brand.New(q).Create(ctx, owner, brand.BrandInput{Name: "Acme", FromEmail: "n@acme.test"})
	if err != nil {
		t.Fatalf("seed brand: %v", err)
	}

	svc := list.New(q)
	// Brand owned → success.
	l, err := svc.Create(ctx, owner, b.ID, "Newsletter")
	if err != nil || l.Name != "Newsletter" {
		t.Fatalf("Create owned: got=%+v err=%v", l, err)
	}
	// Brand not owned (random brand id) → ErrBrandNotFound.
	if _, err := svc.Create(ctx, owner, uuid.New(), "X"); !errors.Is(err, list.ErrBrandNotFound) {
		t.Fatalf("Create unowned brand: err = %v, want ErrBrandNotFound", err)
	}
	// Other owner cannot create under this brand either.
	if _, err := svc.Create(ctx, uuid.New(), b.ID, "X"); !errors.Is(err, list.ErrBrandNotFound) {
		t.Fatalf("Create cross-owner: err = %v, want ErrBrandNotFound", err)
	}
}
