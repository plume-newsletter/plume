package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/team"
)

type teamHandlers struct {
	svc    *team.Service
	cookie *auth.Cookie
	secure bool
}

func (h teamHandlers) writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, team.ErrInvalid):
		http.Error(w, "invalid", http.StatusBadRequest)
	case errors.Is(err, team.ErrNotAllowed):
		http.Error(w, "not allowed", http.StatusForbidden)
	case errors.Is(err, team.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	default:
		http.Error(w, "server error", http.StatusInternalServerError)
	}
}

// gate returns true (and writes 403) unless the caller is owner/admin.
func (h teamHandlers) gate(w http.ResponseWriter, r *http.Request) bool {
	if !requireRole(r.Context(), "owner", "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}

func (h teamHandlers) members(w http.ResponseWriter, r *http.Request) {
	ws, _ := adminID(r.Context())
	m, err := h.svc.Members(r.Context(), ws)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, m)
}

func (h teamHandlers) invite(w http.ResponseWriter, r *http.Request) {
	if !h.gate(w, r) {
		return
	}
	ws, _ := adminID(r.Context())
	var b struct{ Email, Role string }
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	inv, url, err := h.svc.Invite(r.Context(), ws, b.Email, b.Role)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, map[string]any{"invite": inv, "acceptUrl": url})
}

func (h teamHandlers) listInvites(w http.ResponseWriter, r *http.Request) {
	if !h.gate(w, r) {
		return
	}
	ws, _ := adminID(r.Context())
	inv, err := h.svc.ListInvites(r.Context(), ws)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, inv)
}

func (h teamHandlers) revokeInvite(w http.ResponseWriter, r *http.Request) {
	if !h.gate(w, r) {
		return
	}
	ws, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	if err := h.svc.RevokeInvite(r.Context(), ws, id); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h teamHandlers) setRole(w http.ResponseWriter, r *http.Request) {
	if !h.gate(w, r) {
		return
	}
	ws, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	var b struct{ Role string }
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	if err := h.svc.SetRole(r.Context(), ws, id, b.Role); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h teamHandlers) removeMember(w http.ResponseWriter, r *http.Request) {
	if !h.gate(w, r) {
		return
	}
	ws, _ := adminID(r.Context())
	cu, _ := currentUser(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	if err := h.svc.RemoveMember(r.Context(), ws, id, cu.ID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h teamHandlers) getWorkspace(w http.ResponseWriter, r *http.Request) {
	ws, _ := adminID(r.Context())
	name, err := h.svc.Workspace(r.Context(), ws)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, map[string]string{"name": name})
}

func (h teamHandlers) renameWorkspace(w http.ResponseWriter, r *http.Request) {
	if !h.gate(w, r) {
		return
	}
	ws, _ := adminID(r.Context())
	var b struct{ Name string }
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	if err := h.svc.RenameWorkspace(r.Context(), ws, b.Name); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- public (no session) ---

func (h teamHandlers) inviteInfo(w http.ResponseWriter, r *http.Request) {
	info, err := h.svc.GetInvite(r.Context(), chiURLParam(r, "token"))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, info)
}

func (h teamHandlers) acceptInvite(w http.ResponseWriter, r *http.Request) {
	var b struct{ FullName, Password string }
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	user, err := h.svc.AcceptInvite(r.Context(), chiURLParam(r, "token"), b.FullName, b.Password)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    h.cookie.Sign(user.ID),
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(auth.SessionTTL),
	})
	writeJSON(w, map[string]string{"email": user.Email})
}
