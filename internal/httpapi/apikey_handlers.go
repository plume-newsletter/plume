package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/apikey"
	"github.com/plume-newsletter/plume/internal/webhook"
)

type apikeyHandlers struct{ svc *apikey.Service }

func (h apikeyHandlers) list(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	keys, err := h.svc.List(r.Context(), owner)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, keys)
}

func (h apikeyHandlers) create(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r.Context(), "owner", "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	owner, _ := adminID(r.Context())
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	key, secret, err := h.svc.Create(r.Context(), owner, body.Name)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	// The raw secret is returned only here, once.
	writeJSON(w, map[string]any{"key": key, "secret": secret})
}

func (h apikeyHandlers) delete(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r.Context(), "owner", "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	owner, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := h.svc.Delete(r.Context(), owner, id); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type webhookHandlers struct{ svc *webhook.Service }

func (h webhookHandlers) list(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	hooks, err := h.svc.List(r.Context(), owner)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"endpoints": hooks, "events": webhook.Events})
}

func (h webhookHandlers) create(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r.Context(), "owner", "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	owner, _ := adminID(r.Context())
	var body struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	ep, err := h.svc.Create(r.Context(), owner, body.URL, body.Events)
	if err == webhook.ErrInvalid {
		http.Error(w, "a valid URL and at least one event are required", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, ep)
}

func (h webhookHandlers) delete(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r.Context(), "owner", "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	owner, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := h.svc.Delete(r.Context(), owner, id); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
