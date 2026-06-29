package ai

import (
	"context"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// anthropicMessager is the real messager: one Claude Messages call per Complete.
// Thin glue at the SDK boundary — covered by `go build` + manual verification,
// not a unit test (the core logic is tested via the stub in service_test.go).
// ponytail: no `effort: low` set — the Go SDK field for output-config effort is
// unconfirmed; default effort is fine for a 1024-token copy task. Add when verified.
type anthropicMessager struct{}

func (anthropicMessager) complete(ctx context.Context, apiKey, model, system, user string) (string, error) {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 1024,
		System:    []anthropic.TextBlockParam{{Text: system}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(user)),
		},
	})
	if err != nil {
		return "", err
	}
	if resp.StopReason == anthropic.StopReasonRefusal {
		return "", nil // empty → Rewrite maps to ErrRefused
	}
	var sb strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			sb.WriteString(t.Text)
		}
	}
	return sb.String(), nil
}

// NewAnthropic builds a Service backed by the real Anthropic SDK.
func NewAnthropic() *Service { return New(anthropicMessager{}) }
