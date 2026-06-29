package brand_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestBrandCRUDIsOwnerScoped(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	svc := brand.New(gen.New(pool))
	ctx := context.Background()
	owner := uuid.New()
	other := uuid.New()

	created, err := svc.Create(ctx, owner, brand.BrandInput{Name: "Acme", FromName: "Acme", FromEmail: "n@acme.test", ReplyTo: "r@acme.test"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := svc.Get(ctx, owner, created.ID)
	if err != nil || got.Name != "Acme" {
		t.Fatalf("Get: got=%+v err=%v", got, err)
	}

	// A different owner must not see or fetch it.
	if list, _ := svc.List(ctx, other); len(list) != 0 {
		t.Fatalf("other owner List = %d, want 0", len(list))
	}
	if _, err := svc.Get(ctx, other, created.ID); err == nil {
		t.Fatal("other owner Get should fail")
	}

	updated, err := svc.Update(ctx, owner, created.ID, brand.BrandInput{Name: "Acme2", FromName: "A", FromEmail: "n@acme.test", ReplyTo: "r@acme.test"})
	if err != nil || updated.Name != "Acme2" {
		t.Fatalf("Update: got=%+v err=%v", updated, err)
	}

	// Cross-owner update must not mutate the record.
	if _, err := svc.Update(ctx, other, created.ID, brand.BrandInput{Name: "Hijacked", FromName: "A", FromEmail: "n@acme.test", ReplyTo: "r@acme.test"}); err == nil {
		t.Fatal("cross-owner Update should fail (no matching row)")
	}
	if still, err := svc.Get(ctx, owner, created.ID); err != nil || still.Name != "Acme2" {
		t.Fatalf("cross-owner Update mutated record: name=%q err=%v", still.Name, err)
	}

	// Cross-owner delete must be a no-op (DeleteBrand is :exec, idempotent) — record survives.
	if err := svc.Delete(ctx, other, created.ID); err != nil {
		t.Fatalf("cross-owner Delete errored: %v", err)
	}
	if _, err := svc.Get(ctx, owner, created.ID); err != nil {
		t.Fatal("record was deleted by the wrong owner")
	}

	if err := svc.Delete(ctx, owner, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if list, _ := svc.List(ctx, owner); len(list) != 0 {
		t.Fatalf("after delete List = %d, want 0", len(list))
	}
}
