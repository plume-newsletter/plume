package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/abtest"
)

type abtestHandlers struct{ svc *abtest.Service }

func (h abtestHandlers) writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, abtest.ErrInvalid):
		http.Error(w, "invalid", http.StatusBadRequest)
	case errors.Is(err, abtest.ErrState):
		http.Error(w, "wrong state", http.StatusConflict)
	case errors.Is(err, abtest.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	default:
		http.Error(w, "server error", http.StatusInternalServerError)
	}
}

func (h abtestHandlers) idParam(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chiURLParam(r, "id"))
}

func (h abtestHandlers) list(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	ts, err := h.svc.List(r.Context(), owner)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, ts)
}

func (h abtestHandlers) create(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	var b struct {
		CampaignID  string `json:"campaignId"`
		ListID      string `json:"listId"`
		SubjectA    string `json:"subjectA"`
		SubjectB    string `json:"subjectB"`
		TestPercent int    `json:"testPercent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	cid, err1 := uuid.Parse(b.CampaignID)
	lid, err2 := uuid.Parse(b.ListID)
	if err1 != nil || err2 != nil {
		http.Error(w, "bad ids", 400)
		return
	}
	t, err := h.svc.Create(r.Context(), owner, abtest.Input{
		CampaignID:  cid,
		ListID:      lid,
		SubjectA:    b.SubjectA,
		SubjectB:    b.SubjectB,
		TestPercent: b.TestPercent,
	})
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, t)
}

func (h abtestHandlers) get(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := h.idParam(r)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	t, err := h.svc.Get(r.Context(), owner, id)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, t)
}

func (h abtestHandlers) del(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := h.idParam(r)
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

func (h abtestHandlers) start(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := h.idParam(r)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	if err := h.svc.Start(r.Context(), owner, id); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h abtestHandlers) results(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := h.idParam(r)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	res, err := h.svc.Results(r.Context(), owner, id)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, res)
}

func (h abtestHandlers) winner(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := h.idParam(r)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	var b struct {
		Winner string `json:"winner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	if err := h.svc.SendWinner(r.Context(), owner, id, b.Winner); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
