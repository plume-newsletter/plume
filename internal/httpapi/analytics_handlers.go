package httpapi

import (
	"net/http"

	"github.com/plume-newsletter/plume/internal/analytics"
)

type analyticsHandlers struct{ svc *analytics.Service }

func (h analyticsHandlers) overview(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	window := 30
	if r.URL.Query().Get("window") == "90" {
		window = 90
	}
	ov, err := h.svc.Overview(r.Context(), owner, window)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, ov)
}

func (h analyticsHandlers) deliverability(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	window := 30
	if r.URL.Query().Get("window") == "90" {
		window = 90
	}
	d, err := h.svc.Deliverability(r.Context(), owner, window)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, d)
}
