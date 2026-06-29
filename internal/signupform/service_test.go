package signupform_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/signupform"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestSignupFormCRUD(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()
	b, _ := brand.New(q).Create(ctx, owner, brand.BrandInput{Name: "Acme", FromName: "A", FromEmail: "a@a.test", ReplyTo: ""})
	l, _ := list.New(q).Create(ctx, owner, b.ID, "Main")

	svc := signupform.New(q)
	f, err := svc.Create(ctx, owner, signupform.FormInput{ListID: l.ID, Name: "Hero", Heading: "Join us", Description: "Weekly", ButtonText: ""})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if f.Name != "Hero" || f.Heading != "Join us" || f.ButtonText != "Subscribe" {
		t.Fatalf("bad form: %+v", f)
	}
	if f.ListID != l.ID.String() {
		t.Errorf("listId = %s", f.ListID)
	}

	forms, _ := svc.List(ctx, owner)
	if len(forms) != 1 {
		t.Fatalf("list len = %d", len(forms))
	}

	id := uuid.MustParse(f.ID)
	pub, err := svc.GetPublic(ctx, id)
	if err != nil || pub.Heading != "Join us" {
		t.Errorf("getpublic: %v / %q", err, pub.Heading)
	}

	upd, err := svc.Update(ctx, owner, id, signupform.FormInput{ListID: l.ID, Name: "Hero2", Heading: "Join 1000s", Description: "d", ButtonText: "Join"})
	if err != nil || upd.Heading != "Join 1000s" || upd.ButtonText != "Join" {
		t.Errorf("update: %v / %+v", err, upd)
	}

	if _, err := svc.Get(ctx, uuid.New(), id); err != signupform.ErrNotFound {
		t.Errorf("other owner get = %v, want ErrNotFound", err)
	}
	if _, err := svc.Create(ctx, owner, signupform.FormInput{ListID: l.ID, Name: ""}); err != signupform.ErrInvalid {
		t.Errorf("empty name = %v, want ErrInvalid", err)
	}

	if err := svc.Delete(ctx, owner, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	forms, _ = svc.List(ctx, owner)
	if len(forms) != 0 {
		t.Errorf("after delete len = %d", len(forms))
	}
}
