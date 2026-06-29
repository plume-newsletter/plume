package store_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/plume-newsletter/plume/internal/store"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestCreateAndGetBrand(t *testing.T) {
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
	defer func() { _ = testcontainers.TerminateContainer(pg) }()

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("conn string: %v", err)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := store.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// gen.DBTX (sql_package: pgx/v5) is satisfied by *pgxpool.Pool, not *sql.DB,
	// so queries run over a pgx pool while *sql.DB is used only for goose migrations.
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool: %v", err)
	}
	defer pool.Close()

	q := gen.New(pool)
	owner := uuid.New()
	created, err := q.CreateBrand(ctx, gen.CreateBrandParams{
		ID:        uuid.New(),
		OwnerID:   owner,
		Name:      "Acme",
		FromName:  "Acme News",
		FromEmail: "news@acme.test",
		ReplyTo:   "hello@acme.test",
	})
	if err != nil {
		t.Fatalf("CreateBrand: %v", err)
	}

	got, err := q.GetBrand(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetBrand: %v", err)
	}
	if got.Name != "Acme" || got.FromEmail != "news@acme.test" || got.OwnerID != owner {
		t.Fatalf("got = %+v, want Acme/news@acme.test/%s", got, owner)
	}
}
