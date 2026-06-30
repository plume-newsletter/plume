package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/plume-newsletter/plume/internal/abtest"
	"github.com/plume-newsletter/plume/internal/ai"
	"github.com/plume-newsletter/plume/internal/analytics"
	"github.com/plume-newsletter/plume/internal/apikey"
	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/automation"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/campaign"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/report"
	"github.com/plume-newsletter/plume/internal/segment"
	"github.com/plume-newsletter/plume/internal/sending"
	"github.com/plume-newsletter/plume/internal/settings"
	"github.com/plume-newsletter/plume/internal/signup"
	"github.com/plume-newsletter/plume/internal/signupform"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
	"github.com/plume-newsletter/plume/internal/team"
	"github.com/plume-newsletter/plume/internal/template"
	"github.com/plume-newsletter/plume/internal/tracking"
	"github.com/plume-newsletter/plume/internal/unsubscribe"
	"github.com/plume-newsletter/plume/internal/webhook"
)

const sessionCookieName = "plume_session"

// AuthDeps carries what the auth handlers need.
type AuthDeps struct {
	Queries     *gen.Queries
	Cookie      *auth.Cookie
	Secure      bool
	Brands      *brand.Service
	Lists       *list.Service
	Subscribers *subscriber.Service
	Campaigns   *campaign.Service
	Sending     *sending.Service
	Tracking    *tracking.Service
	Unsubscribe *unsubscribe.Service
	Signup      *signup.Service
	Reports     *report.Service
	Settings    *settings.Service
	AI          *ai.Service
	Analytics   *analytics.Service
	Segments    *segment.Service
	SignupForms *signupform.Service
	Team        *team.Service
	ABTests     *abtest.Service
	Automations *automation.Service
	Templates   *template.Service
	APIKeys     *apikey.Service
	Webhooks    *webhook.Service
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (d AuthDeps) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	user, ok, err := auth.Validate(r.Context(), d.Queries, req.Email, req.Password)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    d.Cookie.Sign(user.ID),
		Path:     "/",
		HttpOnly: true,
		Secure:   d.Secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(auth.SessionTTL),
	})
	writeJSON(w, map[string]string{"email": user.Email})
}

func (d AuthDeps) logout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   d.Secure,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (d AuthDeps) me(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := d.Cookie.Verify(c.Value)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := d.Queries.GetAdminByID(r.Context(), id)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	wsName := ""
	if ws, err := d.Queries.GetWorkspace(r.Context(), user.WorkspaceID); err == nil {
		wsName = ws.Name
	}
	writeJSON(w, map[string]string{"email": user.Email, "fullName": user.FullName, "role": user.Role, "workspaceName": wsName})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
