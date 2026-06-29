// src/internal/blocks/blocks_test.go
package blocks

import (
	"errors"
	"strings"
	"testing"
)

func TestRenderHeadingAndText(t *testing.T) {
	html, plain, err := Render([]Block{
		{Type: "heading", Text: "Hello", Level: 2},
		{Type: "text", HTML: "A <b>bold</b> line."},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<h2") || !strings.Contains(html, "Hello") {
		t.Fatalf("heading html: %s", html)
	}
	if !strings.Contains(html, "A <b>bold</b> line.") {
		t.Fatalf("text html: %s", html)
	}
	if !strings.Contains(html, "max-width:600px") {
		t.Fatalf("missing 600px container: %s", html)
	}
	if !strings.Contains(plain, "Hello") || !strings.Contains(plain, "A bold line.") {
		t.Fatalf("plain: %q", plain)
	}
}

func TestRenderButtonImageDividerSpacer(t *testing.T) {
	html, _, err := Render([]Block{
		{Type: "button", Label: "Shop", Href: "https://x.test", Align: "center"},
		{Type: "image", Src: "https://x.test/a.png", Alt: "pic"},
		{Type: "divider"},
		{Type: "spacer", Height: 24},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`href="https://x.test"`, "Shop", `src="https://x.test/a.png"`, `alt="pic"`, "<hr", "height:24px"} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}
}

func TestRenderColumnsAndSocialAndHTML(t *testing.T) {
	html, _, err := Render([]Block{
		{Type: "columns",
			Left:  []Block{{Type: "text", HTML: "L"}},
			Right: []Block{{Type: "text", HTML: "R"}},
		},
		{Type: "social", Items: []SocialItem{{Platform: "twitter", URL: "https://t.test"}}},
		{Type: "html", HTML: "<custom>raw</custom>"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(html, "<td") < 2 {
		t.Fatalf("columns should emit two <td: %s", html)
	}
	if !strings.Contains(html, "https://t.test") {
		t.Fatalf("social url missing: %s", html)
	}
	if !strings.Contains(html, "<custom>raw</custom>") {
		t.Fatalf("html block not passed through: %s", html)
	}
}

func TestRenderUnknownTypeErrors(t *testing.T) {
	if _, _, err := Render([]Block{{Type: "nope"}}); !errors.Is(err, ErrUnknownType) {
		t.Fatalf("want ErrUnknownType, got %v", err)
	}
}

func TestRenderJSONParses(t *testing.T) {
	html, _, err := RenderJSON([]byte(`[{"type":"heading","text":"Hi","level":1}]`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Hi") {
		t.Fatalf("json render: %s", html)
	}
}

func TestColumnsDoNotNestDocumentTable(t *testing.T) {
	html, _, err := Render([]Block{
		{Type: "columns",
			Left:  []Block{{Type: "text", HTML: "L"}},
			Right: []Block{{Type: "text", HTML: "R"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(html, "max-width:600px") != 1 {
		t.Fatalf("document container should appear exactly once, got %d:\n%s", strings.Count(html, "max-width:600px"), html)
	}
	if !strings.Contains(html, ">L<") || !strings.Contains(html, ">R<") {
		t.Fatalf("column contents missing: %s", html)
	}
}
