package auth_test

import (
	"context"
	"database/sql"
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

func newTestQueries(t *testing.T) *gen.Queries {
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
		t.Fatalf("open: %v", err)
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

	return gen.New(pool)
}

func TestEnsureAdminIsIdempotentAndValidateChecksCredentials(t *testing.T) {
	ctx := context.Background()
	q := newTestQueries(t)

	if err := auth.EnsureAdmin(ctx, q, "boot@plume.test", "pw-123456"); err != nil {
		t.Fatalf("EnsureAdmin: %v", err)
	}
	// Second call with different creds must be a no-op.
	if err := auth.EnsureAdmin(ctx, q, "other@plume.test", "different"); err != nil {
		t.Fatalf("EnsureAdmin (2nd): %v", err)
	}

	if _, ok, _ := auth.Validate(ctx, q, "BOOT@plume.test", "pw-123456"); !ok {
		t.Fatal("correct creds (case-insensitive email) should validate")
	}
	if _, ok, _ := auth.Validate(ctx, q, "boot@plume.test", "wrong"); ok {
		t.Fatal("wrong password must not validate")
	}
	if _, ok, _ := auth.Validate(ctx, q, "other@plume.test", "different"); ok {
		t.Fatal("second admin must never have been created")
	}
}

func TestEnsureAdminCreatesWorkspaceAndOwner(t *testing.T) {
	ctx := context.Background()
	q := newTestQueries(t)
	if err := auth.EnsureAdmin(ctx, q, "boss@x.test", "pw123456"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	u, err := q.GetAdminByEmail(ctx, "boss@x.test")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if u.Role != "owner" {
		t.Errorf("role = %q, want owner", u.Role)
	}
	if u.WorkspaceID == uuid.Nil {
		t.Fatal("workspace_id is nil")
	}
	if _, err := q.GetWorkspace(ctx, u.WorkspaceID); err != nil {
		t.Errorf("workspace not created: %v", err)
	}
	// idempotent
	if err := auth.EnsureAdmin(ctx, q, "boss@x.test", "pw123456"); err != nil {
		t.Fatalf("re-ensure: %v", err)
	}
	n, _ := q.CountAdmins(ctx)
	if n != 1 {
		t.Errorf("admins = %d, want 1", n)
	}
}

// TestValidateUnknownEmailIsNotAnError confirms that a completely unknown email
// returns ok=false AND err=nil — not a DB error. This distinguishes the
// not-found path from a real database failure.
func TestValidateUnknownEmailIsNotAnError(t *testing.T) {
	ctx := context.Background()
	q := newTestQueries(t)

	_, ok, err := auth.Validate(ctx, q, "nobody@plume.test", "irrelevant")
	if err != nil {
		t.Fatalf("unknown email must not return a DB error, got: %v", err)
	}
	if ok {
		t.Fatal("unknown email must not validate")
	}
}
