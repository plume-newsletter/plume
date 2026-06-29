package httpapi

import (
	"context"
	"net/http"

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

// requireAuth verifies the session cookie, loads the user, and stores the
// user's workspace id (owner scope) + the current user (id, role) in context.
func requireAuth(cookie *auth.Cookie, q *gen.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
