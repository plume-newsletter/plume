package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/brand"
)

type brandHandlers struct{ svc *brand.Service }

type brandBody struct {
	Name      string `json:"name"`
	FromName  string `json:"fromName"`
	FromEmail string `json:"fromEmail"`
	ReplyTo   string `json:"replyTo"`
}

func (in brandBody) toInput() brand.BrandInput {
	return brand.BrandInput{Name: in.Name, FromName: in.FromName, FromEmail: in.FromEmail, ReplyTo: in.ReplyTo}
}

func (h brandHandlers) list(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	brands, err := h.svc.List(r.Context(), owner)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, brands)
}

func (h brandHandlers) create(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	var body brandBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	created, err := h.svc.Create(r.Context(), owner, body.toInput())
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, created)
}

func (h brandHandlers) get(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	b, err := h.svc.Get(r.Context(), owner, id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, b)
}

func (h brandHandlers) update(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var body brandBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	b, err := h.svc.Update(r.Context(), owner, id, body.toInput())
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, b)
}

func (h brandHandlers) delete(w http.ResponseWriter, r *http.Request) {
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
