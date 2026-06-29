package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/template"
)

type templateHandlers struct{ svc *template.Service }

func (h templateHandlers) writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, template.ErrInvalid):
		http.Error(w, "invalid", http.StatusBadRequest)
	case errors.Is(err, template.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	default:
		http.Error(w, "server error", http.StatusInternalServerError)
	}
}

func (h templateHandlers) list(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	ts, err := h.svc.List(r.Context(), owner, r.URL.Query().Get("category"))
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, ts)
}

func (h templateHandlers) create(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	var b struct {
		Name     string          `json:"name"`
		Category string          `json:"category"`
		BodyJSON json.RawMessage `json:"bodyJson"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	tpl, err := h.svc.Create(r.Context(), owner, b.Name, b.Category, []byte(b.BodyJSON))
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, tpl)
}

func (h templateHandlers) del(w http.ResponseWriter, r *http.Request) {
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

func (h templateHandlers) use(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var b struct {
		BrandID string `json:"brandId"`
		Subject string `json:"subject"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	brandID, err := uuid.Parse(b.BrandID)
	if err != nil {
		http.Error(w, "bad brand", http.StatusBadRequest)
		return
	}
	cid, err := h.svc.Use(r.Context(), owner, id, brandID, b.Subject)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, map[string]string{"campaignId": cid.String()})
}
