package campaign_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/campaign"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestCreateCampaignRequiresOwnedBrand(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()

	b, err := brand.New(q).Create(ctx, owner, brand.BrandInput{Name: "Acme", FromEmail: "n@acme.test"})
	if err != nil {
		t.Fatalf("seed brand: %v", err)
	}

	svc := campaign.New(q)
	in := campaign.CampaignInput{Subject: "Hello", HtmlBody: "<p>hi</p>", PlainBody: "hi"}
	// Brand owned -> success.
	c, err := svc.Create(ctx, owner, b.ID, in)
	if err != nil || c.Subject != "Hello" {
		t.Fatalf("Create owned: got=%+v err=%v", c, err)
	}
	if c.Status != "draft" {
		t.Fatalf("Create owned: status = %q, want draft", c.Status)
	}
	// Brand not owned (random brand id) -> ErrBrandNotFound.
	if _, err := svc.Create(ctx, owner, uuid.New(), in); !errors.Is(err, campaign.ErrBrandNotFound) {
		t.Fatalf("Create unowned brand: err = %v, want ErrBrandNotFound", err)
	}
	// Other owner cannot create under this brand either.
	if _, err := svc.Create(ctx, uuid.New(), b.ID, in); !errors.Is(err, campaign.ErrBrandNotFound) {
		t.Fatalf("Create cross-owner: err = %v, want ErrBrandNotFound", err)
	}
}

func TestCampaignOwnerScoping(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()
	other := uuid.New()

	b, err := brand.New(q).Create(ctx, owner, brand.BrandInput{Name: "Acme", FromEmail: "n@acme.test"})
	if err != nil {
		t.Fatalf("seed brand: %v", err)
	}

	svc := campaign.New(q)
	c, err := svc.Create(ctx, owner, b.ID, campaign.CampaignInput{Subject: "Hello", HtmlBody: "<p>hi</p>", PlainBody: "hi"})
	if err != nil {
		t.Fatalf("seed campaign: %v", err)
	}

	// List is owner-scoped.
	list, err := svc.List(ctx, owner)
	if err != nil || len(list) != 1 {
		t.Fatalf("List owner: got=%+v err=%v", list, err)
	}
	otherList, err := svc.List(ctx, other)
	if err != nil || len(otherList) != 0 {
		t.Fatalf("List other owner: got=%+v err=%v", otherList, err)
	}

	// Get is owner-scoped.
	if _, err := svc.Get(ctx, owner, c.ID); err != nil {
		t.Fatalf("Get owner: err=%v", err)
	}
	if _, err := svc.Get(ctx, other, c.ID); err == nil {
		t.Fatalf("Get other owner: want error, got nil")
	}

	// Update is owner-scoped; cross-owner is a no-op (returns error, no row affected).
	updated, err := svc.Update(ctx, owner, c.ID, "Updated", []byte("[]"))
	if err != nil || updated.Subject != "Updated" {
		t.Fatalf("Update owner: got=%+v err=%v", updated, err)
	}
	if _, err := svc.Update(ctx, other, c.ID, "Hacked", []byte("[]")); err == nil {
		t.Fatalf("Update other owner: want error, got nil")
	}
	// Verify the cross-owner update did not mutate the row.
	unchanged, err := svc.Get(ctx, owner, c.ID)
	if err != nil || unchanged.Subject != "Updated" {
		t.Fatalf("Get after cross-owner update: got=%+v err=%v", unchanged, err)
	}

	// Delete is owner-scoped; cross-owner delete is a no-op.
	if err := svc.Delete(ctx, other, c.ID); err != nil {
		t.Fatalf("Delete other owner: unexpected err=%v", err)
	}
	if _, err := svc.Get(ctx, owner, c.ID); err != nil {
		t.Fatalf("Get after cross-owner delete: want still present, err=%v", err)
	}
	if err := svc.Delete(ctx, owner, c.ID); err != nil {
		t.Fatalf("Delete owner: err=%v", err)
	}
	if _, err := svc.Get(ctx, owner, c.ID); err == nil {
		t.Fatalf("Get after delete: want error, got nil")
	}
}

func TestUpdateRendersBlocksToHTMLAndPlain(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	if err := auth.EnsureAdmin(ctx, q, "c@plume.test", "pw-12345678"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	admin, _, _ := auth.Validate(ctx, q, "c@plume.test", "pw-12345678")

	brand, err := q.CreateBrand(ctx, gen.CreateBrandParams{
		ID: uuid.New(), OwnerID: admin.ID, Name: "B", FromName: "B", FromEmail: "b@x.test", ReplyTo: "",
	})
	if err != nil {
		t.Fatalf("brand: %v", err)
	}

	svc := campaign.New(q)
	c, err := svc.Create(ctx, admin.ID, brand.ID, campaign.CampaignInput{Subject: "S"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := svc.Update(ctx, admin.ID, c.ID, "S2",
		[]byte(`[{"type":"heading","text":"Welcome","level":1}]`))
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Subject != "S2" {
		t.Fatalf("subject = %q", updated.Subject)
	}
	if !strings.Contains(updated.HtmlBody, "Welcome") || !strings.Contains(updated.HtmlBody, "max-width:600px") {
		t.Fatalf("html_body not rendered: %s", updated.HtmlBody)
	}
	if !strings.Contains(updated.PlainBody, "Welcome") {
		t.Fatalf("plain_body not rendered: %q", updated.PlainBody)
	}
	if !strings.Contains(string(updated.BodyJson), "Welcome") {
		t.Fatalf("body_json not stored: %s", updated.BodyJson)
	}
}
