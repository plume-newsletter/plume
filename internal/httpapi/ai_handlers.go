package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/ai"
)

// aiConfigGetter is satisfied by *settings.Service (decrypts the stored key).
type aiConfigGetter interface {
	GetAIConfig(ctx context.Context, adminID uuid.UUID) (apiKey, model string, err error)
}

// aiRewriter is satisfied by *ai.Service.
type aiRewriter interface {
	Rewrite(ctx context.Context, cfg ai.Config, action, text string) (string, error)
}

type aiHandlers struct {
	ai  aiRewriter
	cfg aiConfigGetter
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
