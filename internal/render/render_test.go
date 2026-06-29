package render_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/render"
)

func TestRenderInjectsPixelUnsubscribeAndRewritesLinks(t *testing.T) {
	h := hooks.New()
	render.Register(h)

	rid := uuid.New()
	linkID := uuid.New()
	in := render.Context{
		HTML:        `<html><body><p>Hi <a href="https://acme.test/sale">sale</a></p></body></html>`,
		BaseURL:     "https://mail.example.com",
		RecipientID: rid,
		Links:       []render.Link{{ID: linkID, URL: "https://acme.test/sale"}},
	}

	out, err := render.Render(context.Background(), h, in)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "/t/"+rid.String()) {
		t.Errorf("missing open pixel: %s", out)
	}
	if !strings.Contains(out, "/u/"+rid.String()) {
		t.Errorf("missing unsubscribe link: %s", out)
	}
	if !strings.Contains(out, "/l/"+linkID.String()+"/"+rid.String()) {
		t.Errorf("link not rewritten: %s", out)
	}
	if strings.Contains(out, `href="https://acme.test/sale"`) {
		t.Errorf("original href should have been rewritten: %s", out)
	}
}

func TestExtractLinks(t *testing.T) {
	got := render.ExtractLinks(`<a href="https://a.test/x">a</a> <a href="http://b.test">b</a> <a href="mailto:x@y">m</a>`)
	if len(got) != 2 || got[0] != "https://a.test/x" || got[1] != "http://b.test" {
		t.Fatalf("ExtractLinks = %v", got)
	}
}
