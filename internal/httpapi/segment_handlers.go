package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/segment"
)

type segmentHandlers struct{ svc *segment.Service }

type segmentBody struct {
	Name       string              `json:"name"`
	Match      string              `json:"match"`
	Conditions []segment.Condition `json:"conditions"`
}

func (h segmentHandlers) writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, segment.ErrInvalidCondition):
		http.Error(w, "invalid condition", http.StatusBadRequest)
	case errors.Is(err, segment.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	default:
		http.Error(w, "server error", http.StatusInternalServerError)
	}
}

func (h segmentHandlers) preview(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	var b segmentBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	p, err := h.svc.Preview(r.Context(), owner, b.Match, b.Conditions)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, p)
}

func (h segmentHandlers) list(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	segs, err := h.svc.List(r.Context(), owner)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, segs)
}

func (h segmentHandlers) create(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	var b segmentBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	seg, err := h.svc.Create(r.Context(), owner, b.Name, b.Match, b.Conditions)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, seg)
}

func (h segmentHandlers) get(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	seg, err := h.svc.Get(r.Context(), owner, id)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, seg)
}

func (h segmentHandlers) update(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var b segmentBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	seg, err := h.svc.Update(r.Context(), owner, id, b.Name, b.Match, b.Conditions)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, seg)
}

func (h segmentHandlers) delete(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := h.svc.Delete(r.Context(), owner, id); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h segmentHandlers) fields(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	names, err := h.svc.FieldNames(r.Context(), owner)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, names)
}
