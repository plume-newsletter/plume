package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/signup"
)

type signupHandlers struct{ svc *signup.Service }

// subscribe serves POST /subscribe/{listId}: it accepts either a JSON body
// ({"email","name"}) or a form-encoded body, requires email, and always
// returns a generic 200 message on success — the response never reveals
// whether the address was already subscribed (leak-safe).
func (h signupHandlers) subscribe(w http.ResponseWriter, r *http.Request) {
	listID, err := uuid.Parse(chiURLParam(r, "listId"))
	if err != nil {
		http.Error(w, "bad listId", http.StatusBadRequest)
		return
	}

	var emailAddr, name string
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		var body struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		emailAddr, name = body.Email, body.Name
	} else {
		emailAddr = r.FormValue("email")
		name = r.FormValue("name")
	}
	if emailAddr == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	err = h.svc.Subscribe(r.Context(), listID, emailAddr, name)
	if errors.Is(err, signup.ErrListNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("signup: Subscribe: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Check your email to confirm."))
}

// confirm serves GET /confirm/{subscriberId}: a bad/unknown subscriberId
// still renders the same generic page (no leak, idempotent).
func (h signupHandlers) confirm(w http.ResponseWriter, r *http.Request) {
	if id, err := uuid.Parse(chiURLParam(r, "subscriberId")); err == nil {
		if err := h.svc.Confirm(r.Context(), id); err != nil {
			log.Printf("signup: Confirm: %v", err)
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<!DOCTYPE html><html><body><p>Subscription confirmed.</p></body></html>`))
}
