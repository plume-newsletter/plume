// Package testsupport provides shared helpers for integration tests. It is
// imported only by _test code, so its Testcontainers dependency never enters
// the plume binary.
package testsupport

import (
	"context"
	"database/sql"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/store"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// NewPostgres starts a migrated postgres:17 container and returns a ready pool.
func NewPostgres(t *testing.T) *pgxpool.Pool {
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

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open sql: %v", err)
	}
	if err := store.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	_ = db.Close()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// SeedAdmin bootstraps one admin and returns its id plus a valid session cookie.
func SeedAdmin(t *testing.T, pool *pgxpool.Pool, cookie *auth.Cookie, email, password string) (uuid.UUID, *http.Cookie) {
	t.Helper()
	ctx := context.Background()
	q := gen.New(pool)
	if err := auth.EnsureAdmin(ctx, q, email, password); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	user, ok, err := auth.Validate(ctx, q, email, password)
	if err != nil || !ok {
		t.Fatalf("validate seeded admin: ok=%v err=%v", ok, err)
	}
	// NOTE: "plume_session" must match httpapi.sessionCookieName
	return user.ID, &http.Cookie{Name: "plume_session", Value: cookie.Sign(user.ID)}
}
