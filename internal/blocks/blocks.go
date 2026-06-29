// Package blocks renders a campaign's block array to email-safe HTML + plain text.
// It is a leaf package (no DB/HTTP). Output uses table layout + inline styles only.
package blocks

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
)

var ErrUnknownType = errors.New("blocks: unknown block type")

type SocialItem struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

// Block is the union of all block props; only fields relevant to Type are set.
type Block struct {
	ID     string       `json:"id,omitempty"`
	Type   string       `json:"type"`
	Text   string       `json:"text,omitempty"`   // heading
	Level  int          `json:"level,omitempty"`  // heading
	HTML   string       `json:"html,omitempty"`   // text, html
	Src    string       `json:"src,omitempty"`    // image
	Alt    string       `json:"alt,omitempty"`    // image
	Href   string       `json:"href,omitempty"`   // image, button
	Label  string       `json:"label,omitempty"`  // button
	Align  string       `json:"align,omitempty"`  // button
	Height int          `json:"height,omitempty"` // spacer
	Items  []SocialItem `json:"items,omitempty"`  // social
	Left   []Block      `json:"left,omitempty"`   // columns
	Right  []Block      `json:"right,omitempty"`  // columns
}

// RenderJSON unmarshals a block array and renders it.
func RenderJSON(raw []byte) (htmlOut, plain string, err error) {
	var bs []Block
	if len(raw) == 0 {
		bs = nil
	} else if err := json.Unmarshal(raw, &bs); err != nil {
		return "", "", fmt.Errorf("blocks: bad json: %w", err)
	}
	return Render(bs)
}

// renderBody iterates over blocks and returns the accumulated row HTML and plain
// text without wrapping them in the outer document table. Both Render and the
// columns case call this so that column cells never get a nested 600px table.
func renderBody(bs []Block) (rows, plain string, err error) {
	var body, text strings.Builder
	for _, b := range bs {
		h, t, e := renderOne(b)
		if e != nil {
			return "", "", e
		}
		body.WriteString(h)
		if t != "" {
			text.WriteString(t)
			text.WriteString("\n\n")
		}
	}
	return body.String(), text.String(), nil
}

// Render produces email-safe HTML and a plain-text alternative.
func Render(bs []Block) (htmlOut, plain string, err error) {
	rows, text, err := renderBody(bs)
	if err != nil {
		return "", "", err
	}
	doc := `<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="border-collapse:collapse;"><tr><td align="center"><table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%;border-collapse:collapse;">` +
		rows +
		`</table></td></tr></table>`
	return doc, strings.TrimSpace(text), nil
}

func row(inner string) string {
	return `<tr><td style="padding:8px 24px;font-family:Arial,Helvetica,sans-serif;color:#1f2937;">` + inner + `</td></tr>`
}

var tagStrip = regexp.MustCompile(`<[^>]+>`)

func toPlain(s string) string { return strings.TrimSpace(tagStrip.ReplaceAllString(s, "")) }

func renderOne(b Block) (htmlOut, plain string, err error) {
	switch b.Type {
	case "heading":
		lvl := b.Level
		if lvl < 1 || lvl > 3 {
			lvl = 2
		}
		size := map[int]string{1: "28px", 2: "22px", 3: "18px"}[lvl]
		esc := html.EscapeString(b.Text)
		return row(fmt.Sprintf(`<h%d style="margin:0;font-size:%s;line-height:1.3;">%s</h%d>`, lvl, size, esc, lvl)), b.Text, nil
	case "text":
		return row(fmt.Sprintf(`<div style="font-size:15px;line-height:1.6;">%s</div>`, b.HTML)), toPlain(b.HTML), nil
	case "image":
		img := fmt.Sprintf(`<img src="%s" alt="%s" style="display:block;max-width:100%%;height:auto;border:0;"/>`,
			html.EscapeString(b.Src), html.EscapeString(b.Alt))
		if b.Href != "" {
			img = fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(b.Href), img)
		}
		return row(img), b.Alt, nil
	case "button":
		align := b.Align
		if align == "" {
			align = "left"
		}
		btn := fmt.Sprintf(`<table role="presentation" cellpadding="0" cellspacing="0" align="%s"><tr><td style="background:#1E40AF;border-radius:6px;"><a href="%s" style="display:inline-block;padding:10px 20px;color:#ffffff;text-decoration:none;font-family:Arial,sans-serif;font-size:15px;">%s</a></td></tr></table>`,
			html.EscapeString(align), html.EscapeString(b.Href), html.EscapeString(b.Label))
		return row(btn), b.Label + " (" + b.Href + ")", nil
	case "divider":
		return row(`<hr style="border:0;border-top:1px solid #e5e7eb;margin:0;"/>`), "", nil
	case "spacer":
		h := b.Height
		if h <= 0 {
			h = 16
		}
		return fmt.Sprintf(`<tr><td style="height:%dpx;line-height:%dpx;font-size:0;">&nbsp;</td></tr>`, h, h), "", nil
	case "social":
		var sb strings.Builder
		for _, it := range b.Items {
			sb.WriteString(fmt.Sprintf(`<a href="%s" style="margin-right:8px;color:#1E40AF;text-decoration:none;">%s</a>`,
				html.EscapeString(it.URL), html.EscapeString(it.Platform)))
		}
		return row(sb.String()), "", nil
	case "columns":
		lh, lp, e1 := renderBody(b.Left)
		rh, rp, e2 := renderBody(b.Right)
		if e1 != nil {
			return "", "", e1
		}
		if e2 != nil {
			return "", "", e2
		}
		cell := func(inner string) string {
			return `<td width="50%" valign="top" style="width:50%;"><table role="presentation" width="100%" cellpadding="0" cellspacing="0">` + inner + `</table></td>`
		}
		colPlain := strings.TrimSpace(lp + rp)
		return row(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr>` + cell(lh) + cell(rh) + `</tr></table>`), colPlain, nil
	case "html":
		return row(b.HTML), toPlain(b.HTML), nil
	default:
		return "", "", fmt.Errorf("%w: %q", ErrUnknownType, b.Type)
	}
}
