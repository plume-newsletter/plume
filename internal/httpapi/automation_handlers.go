package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/automation"
)

type automationHandlers struct{ svc *automation.Service }

func (h automationHandlers) writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, automation.ErrInvalid):
		http.Error(w, "invalid", http.StatusBadRequest)
	case errors.Is(err, automation.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	default:
		http.Error(w, "server error", http.StatusInternalServerError)
	}
}

func (h automationHandlers) id(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chiURLParam(r, "id"))
}

func (h automationHandlers) list(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	a, err := h.svc.List(r.Context(), owner)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, a)
}

func (h automationHandlers) create(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	var b struct{ Name, ListID string }
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	lid, err := uuid.Parse(b.ListID)
	if err != nil {
		http.Error(w, "bad listId", 400)
		return
	}
	a, err := h.svc.Create(r.Context(), owner, b.Name, lid)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, a)
}

func (h automationHandlers) get(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := h.id(r)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	a, err := h.svc.Get(r.Context(), owner, id)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, a)
}

func (h automationHandlers) update(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := h.id(r)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	var b struct{ Name, ListID string }
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	lid, err := uuid.Parse(b.ListID)
	if err != nil {
		http.Error(w, "bad listId", 400)
		return
	}
	a, err := h.svc.Update(r.Context(), owner, id, b.Name, lid)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, a)
}

func (h automationHandlers) del(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := h.id(r)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	if err := h.svc.Delete(r.Context(), owner, id); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h automationHandlers) steps(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := h.id(r)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	var b struct {
		Steps []automation.Step `json:"steps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	if err := h.svc.ReplaceSteps(r.Context(), owner, id, b.Steps); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h automationHandlers) status(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := h.id(r)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	var b struct{ Status string }
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	if err := h.svc.SetStatus(r.Context(), owner, id, b.Status); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
