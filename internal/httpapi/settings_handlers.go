package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/plume-newsletter/plume/internal/settings"
)

type settingsHandlers struct{ svc *settings.Service }

func (h settingsHandlers) get(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	st, err := h.svc.Get(r.Context(), owner)
	if err != nil {
		if errors.Is(err, settings.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, st)
}

func (h settingsHandlers) setSES(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r.Context(), "owner", "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	owner, _ := adminID(r.Context())
	var body struct {
		AccessKeyID     string `json:"accessKeyId"`
		SecretAccessKey string `json:"secretAccessKey"`
		Region          string `json:"region"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
		body.AccessKeyID == "" || body.SecretAccessKey == "" || body.Region == "" {
		http.Error(w, "accessKeyId, secretAccessKey and region are required", http.StatusBadRequest)
		return
	}
	if err := h.svc.SetSES(r.Context(), owner, settings.SESInput{
		AccessKeyID:     body.AccessKeyID,
		SecretAccessKey: body.SecretAccessKey,
		Region:          body.Region,
	}); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h settingsHandlers) setAI(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r.Context(), "owner", "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	owner, _ := adminID(r.Context())
	var body struct {
		APIKey string `json:"apiKey"`
		Model  string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.APIKey == "" {
		http.Error(w, "apiKey is required", http.StatusBadRequest)
		return
	}
	if err := h.svc.SetAI(r.Context(), owner, body.APIKey, body.Model); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
