package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/plume-newsletter/plume/internal/subscriber"
)

type subscriberHandlers struct{ svc *subscriber.Service }

func (h subscriberHandlers) add(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	listID, err := uuid.Parse(chiURLParam(r, "listId"))
	if err != nil {
		http.Error(w, "bad listId", http.StatusBadRequest)
		return
	}
	var body struct {
		Email  string `json:"email"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}
	sub, created, err := h.svc.Add(r.Context(), owner, listID, subscriber.SubscriberInput{
		Email:  body.Email,
		Name:   body.Name,
		Status: body.Status,
	})
	if errors.Is(err, subscriber.ErrListNotFound) {
		http.Error(w, "list not found", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if created {
		w.WriteHeader(http.StatusCreated)
	}
	writeJSON(w, sub)
}

func (h subscriberHandlers) list(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	listID, err := uuid.Parse(chiURLParam(r, "listId"))
	if err != nil {
		http.Error(w, "bad listId", http.StatusBadRequest)
		return
	}
	subs, err := h.svc.List(r.Context(), owner, listID)
	if errors.Is(err, subscriber.ErrListNotFound) {
		http.Error(w, "list not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, subs)
}

func (h subscriberHandlers) setStatus(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}
	sub, err := h.svc.SetStatus(r.Context(), owner, id, body.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, sub)
}

func (h subscriberHandlers) delete(w http.ResponseWriter, r *http.Request) {
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
