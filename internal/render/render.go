// Package render builds the per-recipient email HTML by running the
// render.email_html hook Filter chain (open pixel, unsubscribe, click rewrite).
package render

import (
	"context"
	"regexp"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/hooks"
)

const HookName = "render.email_html"

type Link struct {
	ID  uuid.UUID
	URL string
}

type Context struct {
	HTML        string
	BaseURL     string
	RecipientID uuid.UUID
	Links       []Link
}

func Render(ctx context.Context, h *hooks.Hooks, in Context) (string, error) {
	out, err := hooks.Filter(ctx, h, HookName, in)
	if err != nil {
		return "", err
	}
	return out.HTML, nil
}

var hrefRe = regexp.MustCompile(`(?i)href="(https?://[^"]+)"`)

// ExtractLinks returns the distinct http(s) hrefs in document order.
func ExtractLinks(html string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range hrefRe.FindAllStringSubmatch(html, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	return out
}
