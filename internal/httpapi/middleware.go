package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

type ctxKey int

const (
	adminIDKey     ctxKey = 0 // holds the WORKSPACE id (owner scope)
	currentUserKey ctxKey = 1
)

// CurrentUser carries the authenticated user's id and role for the current request.
type CurrentUser struct {
	ID   uuid.UUID
	Role string
}

// apiKeyAuth resolves a raw API key to its workspace id. Satisfied by *apikey.Service.
type apiKeyAuth interface {
	Authenticate(ctx context.Context, raw string) (uuid.UUID, error)
}

// requireAuth authenticates a request via either an "Authorization: Bearer
// <api key>" header or the session cookie, and stores the workspace id (owner
// scope) + current user (id, role) in context. An API key acts as the
// workspace owner, so role-gated routes treat it as role "owner".
func requireAuth(cookie *auth.Cookie, q *gen.Queries, keys apiKeyAuth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if raw, ok := bearerToken(r); ok {
				owner, err := keys.Authenticate(r.Context(), raw)
				if err != nil {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				ctx := context.WithValue(r.Context(), adminIDKey, owner)
				ctx = context.WithValue(ctx, currentUserKey, CurrentUser{ID: owner, Role: "owner"})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			c, err := r.Cookie(sessionCookieName)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			uid, ok := cookie.Verify(c.Value)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			user, err := q.GetAdminByID(r.Context(), uid)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), adminIDKey, user.WorkspaceID)
			ctx = context.WithValue(ctx, currentUserKey, CurrentUser{ID: user.ID, Role: user.Role})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func adminID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(adminIDKey).(uuid.UUID)
	return id, ok
}

func currentUser(ctx context.Context) (CurrentUser, bool) {
	u, ok := ctx.Value(currentUserKey).(CurrentUser)
	return u, ok
}

func requireRole(ctx context.Context, roles ...string) bool {
	u, ok := currentUser(ctx)
	if !ok {
		return false
	}
	for _, r := range roles {
		if u.Role == r {
			return true
		}
	}
	return false
}

func chiURLParam(r *http.Request, key string) string { return chi.URLParam(r, key) }

// bearerToken extracts the token from an "Authorization: Bearer <token>" header.
func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	const p = "Bearer "
	if len(h) > len(p) && strings.EqualFold(h[:len(p)], p) {
		return h[len(p):], true
	}
	return "", false
}
