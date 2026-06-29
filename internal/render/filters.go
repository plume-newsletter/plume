package render

import (
	"context"
	"strings"

	"github.com/plume-newsletter/plume/internal/hooks"
)

// Register wires the built-in render filters onto the render.email_html hook.
func Register(h *hooks.Hooks) {
	h.AddFilter(HookName, 10, openPixel)
	h.AddFilter(HookName, 20, unsubscribe)
	h.AddFilter(HookName, 30, clickRewrite)
}

func openPixel(_ context.Context, v any) (any, error) {
	c := v.(Context)
	img := `<img src="` + c.BaseURL + "/t/" + c.RecipientID.String() + `" width="1" height="1" alt="">`
	c.HTML = injectBeforeBodyEnd(c.HTML, img)
	return c, nil
}

func unsubscribe(_ context.Context, v any) (any, error) {
	c := v.(Context)
	link := `<div><a href="` + c.BaseURL + "/u/" + c.RecipientID.String() + `">Unsubscribe</a></div>`
	c.HTML = injectBeforeBodyEnd(c.HTML, link)
	return c, nil
}

func clickRewrite(_ context.Context, v any) (any, error) {
	c := v.(Context)
	for _, l := range c.Links {
		from := `href="` + l.URL + `"`
		to := `href="` + c.BaseURL + "/l/" + l.ID.String() + "/" + c.RecipientID.String() + `"`
		c.HTML = strings.ReplaceAll(c.HTML, from, to)
	}
	return c, nil
}

func injectBeforeBodyEnd(html, snippet string) string {
	if i := strings.LastIndex(strings.ToLower(html), "</body>"); i >= 0 {
		return html[:i] + snippet + html[i:]
	}
	return html + snippet
}
