package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/ai"
	"github.com/plume-newsletter/plume/internal/analytics"
	"github.com/plume-newsletter/plume/internal/segment"
)

// aiConfigGetter is satisfied by *settings.Service (decrypts the stored key).
type aiConfigGetter interface {
	GetAIConfig(ctx context.Context, adminID uuid.UUID) (apiKey, model string, err error)
}

// aiService is satisfied by *ai.Service.
type aiService interface {
	Rewrite(ctx context.Context, cfg ai.Config, action, text string) (string, error)
	Chat(ctx context.Context, cfg ai.Config, msgs []ai.Message) (string, error)
	Suggest(ctx context.Context, cfg ai.Config, kind, context string) ([]string, error)
	Insights(ctx context.Context, cfg ai.Config, analyticsJSON string) ([]ai.Insight, error)
	SegmentRules(ctx context.Context, cfg ai.Config, prompt string, availableFields []string) (ai.SegmentRules, error)
}

type aiHandlers struct {
	ai        aiService
	cfg       aiConfigGetter
	analytics *analytics.Service
	segments  *segment.Service
}

func (h aiHandlers) rewrite(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	var body struct {
		Action string `json:"action"`
		Text   string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	apiKey, model, err := h.cfg.GetAIConfig(r.Context(), owner)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if apiKey == "" {
		http.Error(w, "AI not configured", http.StatusBadRequest)
		return
	}
	out, err := h.ai.Rewrite(r.Context(), ai.Config{APIKey: apiKey, Model: model}, body.Action, body.Text)
	if err != nil {
		switch {
		case errors.Is(err, ai.ErrEmpty), errors.Is(err, ai.ErrTooLong), errors.Is(err, ai.ErrBadAction):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "AI request failed", http.StatusBadGateway)
		}
		return
	}
	writeJSON(w, map[string]string{"text": out})
}

func (h aiHandlers) chat(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	var body struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	apiKey, model, err := h.cfg.GetAIConfig(r.Context(), owner)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if apiKey == "" {
		http.Error(w, "AI not configured", http.StatusBadRequest)
		return
	}
	msgs := make([]ai.Message, len(body.Messages))
	for i, m := range body.Messages {
		msgs[i] = ai.Message{Role: m.Role, Content: m.Content}
	}
	out, err := h.ai.Chat(r.Context(), ai.Config{APIKey: apiKey, Model: model}, msgs)
	if err != nil {
		switch {
		case errors.Is(err, ai.ErrNoMessages), errors.Is(err, ai.ErrEmpty), errors.Is(err, ai.ErrTooLong):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "AI request failed", http.StatusBadGateway)
		}
		return
	}
	writeJSON(w, map[string]string{"reply": out})
}

func (h aiHandlers) suggest(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	var body struct {
		Kind    string `json:"kind"`
		Context string `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	apiKey, model, err := h.cfg.GetAIConfig(r.Context(), owner)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if apiKey == "" {
		http.Error(w, "AI not configured", http.StatusBadRequest)
		return
	}
	options, err := h.ai.Suggest(r.Context(), ai.Config{APIKey: apiKey, Model: model}, body.Kind, body.Context)
	if err != nil {
		switch {
		case errors.Is(err, ai.ErrEmpty), errors.Is(err, ai.ErrTooLong), errors.Is(err, ai.ErrBadAction):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "AI request failed", http.StatusBadGateway)
		}
		return
	}
	writeJSON(w, map[string][]string{"options": options})
}

func (h aiHandlers) insights(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	apiKey, model, err := h.cfg.GetAIConfig(r.Context(), owner)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if apiKey == "" {
		http.Error(w, "AI not configured", http.StatusBadRequest)
		return
	}
	ov, err := h.analytics.Overview(r.Context(), owner, 30)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	snapshot, err := json.Marshal(ov)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	insights, err := h.ai.Insights(r.Context(), ai.Config{APIKey: apiKey, Model: model}, string(snapshot))
	if err != nil {
		http.Error(w, "AI request failed", http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string][]ai.Insight{"insights": insights})
}

func (h aiHandlers) segmentRules(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	var body struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	apiKey, model, err := h.cfg.GetAIConfig(r.Context(), owner)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if apiKey == "" {
		http.Error(w, "AI not configured", http.StatusBadRequest)
		return
	}
	fields, err := h.segments.FieldNames(r.Context(), owner)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	rules, err := h.ai.SegmentRules(r.Context(), ai.Config{APIKey: apiKey, Model: model}, body.Prompt, fields)
	if err != nil {
		switch {
		case errors.Is(err, ai.ErrEmpty), errors.Is(err, ai.ErrTooLong):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "AI request failed", http.StatusBadGateway)
		}
		return
	}

	// Validate the model's output against the real compiler: convert to the
	// segment condition type and run a preview. Garbage rules are rejected
	// rather than handed to the builder.
	conds := make([]segment.Condition, len(rules.Conditions))
	for i, c := range rules.Conditions {
		conds[i] = segment.Condition{Type: c.Type, Op: c.Op, Days: c.Days, Field: c.Field, Value: c.Value}
	}
	preview, err := h.segments.Preview(r.Context(), owner, rules.Match, conds)
	if err != nil {
		// ErrInvalidCondition or a DB error: the rules aren't usable.
		http.Error(w, "AI produced invalid rules", http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]any{"match": rules.Match, "conditions": rules.Conditions, "count": preview.Count})
}
