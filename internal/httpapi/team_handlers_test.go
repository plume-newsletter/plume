package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/email"
	"github.com/plume-newsletter/plume/internal/httpapi"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/team"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestTeamEndpoints(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	cookie := auth.NewCookie([]byte("a-32-byte-or-longer-test-secret!!"))
	_, session := testsupport.SeedAdmin(t, pool, cookie, "owner@plume.test", "pw-12345678")

	q := gen.New(pool)
	router := httpapi.NewRouter(httpapi.AuthDeps{
		Queries: q,
		Cookie:  cookie,
		Brands:  brand.New(q),
		Team:    team.New(q, email.NoopResolver(), "http://localhost:8080"),
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	// --- authed GET /api/team → 200 with >=1 member ---
	t.Run("list members requires auth and returns owner", func(t *testing.T) {
		// Unauthenticated → 401.
		resp, _ := http.Get(srv.URL + "/api/team")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("no auth: status = %d, want 401", resp.StatusCode)
		}

		// Authenticated → 200 with at least one member.
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/team", nil)
		req.AddCookie(session)
		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("list members: status=%v err=%v", resp.StatusCode, err)
		}
		var members []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
			t.Fatalf("decode members: %v", err)
		}
		if len(members) < 1 {
			t.Fatalf("members = %d, want >=1", len(members))
		}
	})

	// --- authed POST /api/team/invites → 200 with acceptUrl ---
	t.Run("invite as owner returns acceptUrl", func(t *testing.T) {
		body := bytes.NewBufferString(`{"email":"editor@x.test","role":"editor"}`)
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/team/invites", body)
		req.AddCookie(session)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("invite: status=%v err=%v", resp.StatusCode, err)
		}
		var result map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode invite response: %v", err)
		}
		if _, ok := result["acceptUrl"]; !ok {
			t.Fatalf("invite response missing acceptUrl: %+v", result)
		}
		if result["acceptUrl"] == "" {
			t.Fatal("acceptUrl is empty")
		}
	})

	// --- public (no cookie) GET /api/invites/badtoken → 404 ---
	t.Run("public invite info with bad token returns 404", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/invites/badtoken")
		if err != nil {
			t.Fatalf("get invite info: %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("bad token: status = %d, want 404", resp.StatusCode)
		}
	})
}
