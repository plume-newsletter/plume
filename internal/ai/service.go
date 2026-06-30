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
	ErrEmpty      = errors.New("ai: text is empty")
	ErrTooLong    = errors.New("ai: text exceeds limit")
	ErrBadAction  = errors.New("ai: unknown action")
	ErrRefused    = errors.New("ai: model returned no usable text")
	ErrNoMessages = errors.New("ai: no messages")
)

// Config carries the per-request credentials/model resolved from settings.
type Config struct {
	APIKey string
	Model  string
}

// messager performs Claude completions. Real impl: anthropicMessager (anthropic.go).
type messager interface {
	complete(ctx context.Context, apiKey, model, system, user string) (string, error)
	completeChat(ctx context.Context, apiKey, model, system string, msgs []Message) (string, error)
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

// Message is one turn in a Chat conversation.
type Message struct {
	Role    string // "user" | "assistant"
	Content string
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

// maxChatChars bounds the total conversation size sent to the model.
const maxChatChars = 12000

const chatSystemPrompt = "You are Plume AI, a helpful assistant inside the Plume email-marketing app. " +
	"You help draft campaign copy, build segment rules, and outline automations, and you give concise, " +
	"practical marketing guidance. Keep replies focused and skimmable. You cannot perform actions or read " +
	"the user's account data — you propose, and the user applies."

// Chat runs a multi-turn, non-streaming conversation and returns the assistant reply.
func (s *Service) Chat(ctx context.Context, cfg Config, msgs []Message) (string, error) {
	if len(msgs) == 0 {
		return "", ErrNoMessages
	}
	total := 0
	for _, m := range msgs {
		total += len(m.Content)
	}
	if total == 0 {
		return "", ErrEmpty
	}
	if total > maxChatChars {
		return "", ErrTooLong
	}
	model := cfg.Model
	if model == "" {
		model = DefaultModel
	}
	out, err := s.m.completeChat(ctx, cfg.APIKey, model, chatSystemPrompt, msgs)
	if err != nil {
		return "", err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return "", ErrRefused
	}
	return out, nil
}

const suggestSubjectPrompt = "You are an email subject-line assistant. Given the email body, propose exactly 3 " +
	"distinct, compelling subject lines. Return only the 3 subject lines, one per line — no numbering, no quotes, " +
	"no extra text."

// Suggest returns up to 3 suggestions for the given kind. Only "subject" is supported today.
func (s *Service) Suggest(ctx context.Context, cfg Config, kind, context string) ([]string, error) {
	if kind != "subject" {
		return nil, ErrBadAction
	}
	context = strings.TrimSpace(context)
	if context == "" {
		return nil, ErrEmpty
	}
	if len(context) > maxInputChars {
		return nil, ErrTooLong
	}
	model := cfg.Model
	if model == "" {
		model = DefaultModel
	}
	out, err := s.m.complete(ctx, cfg.APIKey, model, suggestSubjectPrompt, "Email body:\n\n"+context)
	if err != nil {
		return nil, err
	}
	options := parseSuggestionLines(out)
	if len(options) == 0 {
		return nil, ErrRefused
	}
	return options, nil
}

// parseSuggestionLines splits model output into at most 3 trimmed, de-quoted, non-empty lines.
func parseSuggestionLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		line = strings.Trim(line, `"'`)
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
		if len(out) == 3 {
			break
		}
	}
	return out
}
