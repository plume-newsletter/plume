package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/sending"
)

type sendingHandlers struct{ svc *sending.Service }

func (h sendingHandlers) send(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	campaignID, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		ListID string `json:"listId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	listID, err := uuid.Parse(body.ListID)
	if err != nil {
		http.Error(w, "bad listId", http.StatusBadRequest)
		return
	}

	n, err := h.svc.Enqueue(r.Context(), owner, campaignID, listID)
	if errors.Is(err, sending.ErrCampaignNotFound) || errors.Is(err, sending.ErrListNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, sending.ErrAlreadyQueued) {
		http.Error(w, "campaign already queued or sent", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]int{"recipients": n})
}
