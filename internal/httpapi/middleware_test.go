package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestRequireAuthRejectsAndAllows(t *testing.T) {
	cookie := auth.NewCookie([]byte("a-32-byte-or-longer-test-secret!!"))

	// Set up a real DB with a seeded admin so the valid-cookie path can
	// resolve the user's workspace id.
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	if err := auth.EnsureAdmin(ctx, q, "mw@test.test", "pw-12345678"); err != nil {
		t.Fatalf("ensure admin: %v", err)
	}
	user, ok, err := auth.Validate(ctx, q, "mw@test.test", "pw-12345678")
	if err != nil || !ok {
		t.Fatalf("validate: ok=%v err=%v", ok, err)
	}

	var sawID uuid.UUID
	protected := requireAuth(cookie, q)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := adminID(r.Context())
		if !ok {
			t.Error("adminID missing in context")
		}
		sawID = id
		w.WriteHeader(http.StatusOK)
	}))

	// No cookie → 401.
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/brands", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no cookie: status = %d, want 401", rec.Code)
	}

	// Tampered/garbage cookie → 401.
	reqBad := httptest.NewRequest(http.MethodGet, "/api/brands", nil)
	reqBad.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "not.a.valid-token"})
	recBad := httptest.NewRecorder()
	protected.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusUnauthorized {
		t.Fatalf("tampered cookie: status = %d, want 401", recBad.Code)
	}

	// Valid cookie → 200, workspace id in context (adminID now returns workspace_id).
	req := httptest.NewRequest(http.MethodGet, "/api/brands", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: cookie.Sign(user.ID)})
	rec = httptest.NewRecorder()
	protected.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("valid cookie: status = %d, want 200", rec.Code)
	}
	if sawID != user.WorkspaceID {
		t.Fatalf("context workspace id = %s, want %s", sawID, user.WorkspaceID)
	}
}
