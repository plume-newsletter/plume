// Package ai is Plume's shared Claude-backed AI capability. The core depends only
// on the unexported messager interface; the real Anthropic adapter lives in
// anthropic.go and is the single place that imports the SDK.
package ai

import (
	"context"
	"errors"
	"strings"
)

// DefaultModel is used when Config.Model is empty.
const DefaultModel = "claude-opus-4-8"

const maxInputChars = 4000

var (
	ErrEmpty     = errors.New("ai: text is empty")
	ErrTooLong   = errors.New("ai: text exceeds limit")
	ErrBadAction = errors.New("ai: unknown action")
	ErrRefused   = errors.New("ai: model returned no usable text")
)

// Config carries the per-request credentials/model resolved from settings.
type Config struct {
	APIKey string
	Model  string
}

// messager performs one Claude completion. Real impl: anthropicMessager (anthropic.go).
type messager interface {
	complete(ctx context.Context, apiKey, model, system, user string) (string, error)
}

type Service struct{ m messager }

func New(m messager) *Service { return &Service{m: m} }

const systemPrompt = "You are an email copywriting assistant. " +
	"Apply the requested edit to the user's text. " +
	"Return only the edited text — no preamble, no quotes, no explanation."

func instructionFor(action string) (string, bool) {
	switch action {
	case "rewrite":
		return "Rewrite the following text to improve clarity and flow, keeping the same meaning and roughly the same length:", true
	case "shorten":
		return "Make the following text more concise while keeping its meaning:", true
	case "more_casual":
		return "Rewrite the following text in a warmer, more casual tone:", true
	default:
		return "", false
	}
}

// Rewrite applies action ("rewrite"|"shorten"|"more_casual") to text.
func (s *Service) Rewrite(ctx context.Context, cfg Config, action, text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ErrEmpty
	}
	if len(text) > maxInputChars {
		return "", ErrTooLong
	}
	instr, ok := instructionFor(action)
	if !ok {
		return "", ErrBadAction
	}
	model := cfg.Model
	if model == "" {
		model = DefaultModel
	}
	out, err := s.m.complete(ctx, cfg.APIKey, model, systemPrompt, instr+"\n\n"+text)
	if err != nil {
		return "", err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return "", ErrRefused
	}
	return out, nil
}
