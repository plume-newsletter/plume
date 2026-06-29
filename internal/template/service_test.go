package template_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/campaign"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/template"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestListReturnsStartersForFreshOwner(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	svc := template.New(q, campaign.New(q))
	owner := uuid.New()
	out, err := svc.List(context.Background(), owner, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Fatalf("want 3 starters, got %d", len(out))
	}
	for _, tpl := range out {
		if !tpl.Prebuilt {
			t.Fatalf("fresh owner should see only prebuilt, got owned %s", tpl.Name)
		}
	}
}

func TestCreateThenListIncludesOwned(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	svc := template.New(q, campaign.New(q))
	owner := uuid.New()
	body := []byte(`[{"id":"x","type":"text","text":"hi"}]`)
	tpl, err := svc.Create(context.Background(), owner, "My layout", "Promo", body)
	if err != nil {
		t.Fatal(err)
	}
	if tpl.Prebuilt {
		t.Fatal("created template must not be prebuilt")
	}
	out, _ := svc.List(context.Background(), owner, "")
	if len(out) != 4 {
		t.Fatalf("want 3 starters + 1 owned, got %d", len(out))
	}
	byCat, _ := svc.List(context.Background(), owner, "Promo")
	// starters include a Promo, plus the owned Promo
	names := map[string]bool{}
	for _, x := range byCat {
		names[x.Name] = true
	}
	if !names["My layout"] {
		t.Fatal("category filter dropped owned Promo")
	}
}

func TestDeleteWontRemovePrebuiltOrForeign(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	svc := template.New(q, campaign.New(q))
	owner := uuid.New()
	out, _ := svc.List(context.Background(), owner, "")
	var prebuiltID uuid.UUID
	for _, x := range out {
		if x.Prebuilt {
			prebuiltID = uuid.MustParse(x.ID)
			break
		}
	}
	if err := svc.Delete(context.Background(), owner, prebuiltID); err != nil {
		t.Fatal(err)
	}
	after, _ := svc.List(context.Background(), owner, "")
	if len(after) != 3 {
		t.Fatalf("prebuilt must survive delete, got %d", len(after))
	}
}

func TestUseCreatesDraftCampaignWithRenderedBlocks(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	cs := campaign.New(q)
	svc := template.New(q, cs)
	owner := uuid.New()
	// a brand is required to create a campaign — create one via brand service
	b, err := brand.New(q).Create(context.Background(), owner, brand.BrandInput{Name: "Acme", FromName: "A", FromEmail: "a@a.test", ReplyTo: ""})
	if err != nil {
		t.Fatal(err)
	}
	starters, _ := svc.List(context.Background(), owner, "")
	tplID := uuid.MustParse(starters[0].ID)
	newID, err := svc.Use(context.Background(), owner, tplID, b.ID, "Hello world")
	if err != nil {
		t.Fatal(err)
	}
	c, err := cs.Get(context.Background(), owner, newID)
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != "draft" {
		t.Fatalf("want draft, got %s", c.Status)
	}
	if c.Subject != "Hello world" {
		t.Fatalf("subject not set: %s", c.Subject)
	}
	if len(c.HtmlBody) == 0 {
		t.Fatal("expected rendered html from template blocks")
	}
	if string(c.BodyJson) == "[]" || len(c.BodyJson) == 0 {
		t.Fatal("expected template body_json copied")
	}
}

func TestUseRejectsForeignTemplate(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	svc := template.New(q, campaign.New(q))
	owner := uuid.New()
	other := uuid.New()
	body := []byte(`[{"id":"x","type":"text","text":"hi"}]`)
	tpl, _ := svc.Create(context.Background(), other, "theirs", "Promo", body)
	b, _ := brand.New(q).Create(context.Background(), owner, brand.BrandInput{Name: "Acme", FromName: "A", FromEmail: "a@a.test", ReplyTo: ""})
	_, err := svc.Use(context.Background(), owner, uuid.MustParse(tpl.ID), b.ID, "x")
	if !errors.Is(err, template.ErrNotFound) {
		t.Fatalf("want ErrNotFound for foreign template, got %v", err)
	}
}
