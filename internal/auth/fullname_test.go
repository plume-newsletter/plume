package auth_test

import (
	"context"
	"testing"

	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestAdminHasFullNameColumnDefaultingEmpty(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	if err := auth.EnsureAdmin(ctx, q, "u@plume.test", "pw-12345678"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	admin, _, _ := auth.Validate(ctx, q, "u@plume.test", "pw-12345678")
	got, err := q.GetAdminByID(ctx, admin.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.FullName != "" {
		t.Fatalf("FullName default should be empty, got %q", got.FullName)
	}
}
