package httpapi_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/httpapi"
	"github.com/plume-newsletter/plume/internal/store"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func newTestDeps(t *testing.T) httpapi.AuthDeps {
	t.Helper()
	ctx := context.Background()

	pg, err := tcpostgres.Run(ctx, "postgres:17",
		tcpostgres.WithDatabase("plume"),
		tcpostgres.WithUsername("plume"),
		tcpostgres.WithPassword("plume"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(pg) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("conn string: %v", err)
	}

	// *sql.DB (via pgx stdlib) is used only for goose migrations.
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := store.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// gen.DBTX requires pgx interface; use *pgxpool.Pool for gen.New.
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	q := gen.New(pool)
	if err := auth.EnsureAdmin(ctx, q, "admin@plume.test", "secure-password-123"); err != nil {
		t.Fatalf("EnsureAdmin: %v", err)
	}

	return httpapi.AuthDeps{
		Queries: q,
		Cookie:  auth.NewCookie([]byte("a-32-byte-or-longer-test-secret!!")),
		Secure:  false,
	}
}

func TestAuthHandlers(t *testing.T) {
	deps := newTestDeps(t)
	srv := httptest.NewServer(httpapi.NewRouter(deps))
	defer srv.Close()

	client := &http.Client{}

	// POST /api/login with correct creds → expect 200 and plume_session cookie.
	t.Run("login with correct creds returns 200 and session cookie", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":    "admin@plume.test",
			"password": "secure-password-123",
		})
		resp, err := client.Post(srv.URL+"/api/login", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("POST /api/login: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}

		var sessionCookie *http.Cookie
		for _, c := range resp.Cookies() {
			if c.Name == "plume_session" {
				sessionCookie = c
				break
			}
		}
		if sessionCookie == nil {
			t.Fatal("expected plume_session cookie in response, got none")
		}
	})

	// GET /api/me WITH the session cookie → expect 200 and admin email.
	t.Run("me with valid session cookie returns 200 and email", func(t *testing.T) {
		// First login to get a cookie.
		body, _ := json.Marshal(map[string]string{
			"email":    "admin@plume.test",
			"password": "secure-password-123",
		})
		loginResp, err := client.Post(srv.URL+"/api/login", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("POST /api/login: %v", err)
		}
		loginResp.Body.Close()

		var sessionCookie *http.Cookie
		for _, c := range loginResp.Cookies() {
			if c.Name == "plume_session" {
				sessionCookie = c
				break
			}
		}
		if sessionCookie == nil {
			t.Fatal("expected plume_session cookie from login")
		}

		// Now call /api/me with the cookie.
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/me", nil)
		req.AddCookie(sessionCookie)
		meResp, err := client.Do(req)
		if err != nil {
			t.Fatalf("GET /api/me: %v", err)
		}
		defer meResp.Body.Close()

		if meResp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", meResp.StatusCode)
		}

		var result map[string]string
		if err := json.NewDecoder(meResp.Body).Decode(&result); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if result["email"] != "admin@plume.test" {
			t.Fatalf("email = %q, want %q", result["email"], "admin@plume.test")
		}
		if result["role"] != "owner" {
			t.Fatalf("role = %q, want %q", result["role"], "owner")
		}
		if result["workspaceName"] == "" {
			t.Fatal("workspaceName must be non-empty")
		}
	})

	// POST /api/login with wrong password → expect 401.
	t.Run("login with wrong password returns 401", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":    "admin@plume.test",
			"password": "wrong-password",
		})
		resp, err := client.Post(srv.URL+"/api/login", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("POST /api/login: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})

	// GET /api/me with NO cookie → expect 401.
	t.Run("me with no cookie returns 401", func(t *testing.T) {
		resp, err := client.Get(srv.URL + "/api/me")
		if err != nil {
			t.Fatalf("GET /api/me: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})
}
