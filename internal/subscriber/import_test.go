package subscriber_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestImportCSVDedupesAppliesFilterAndCounts(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()
	b, _ := brand.New(q).Create(ctx, owner, brand.BrandInput{Name: "Acme", FromEmail: "n@acme.test"})
	l, _ := list.New(q).Create(ctx, owner, b.ID, "News")

	h := hooks.New()
	// Filter that skips any row whose email contains "skip".
	h.AddFilter(subscriber.HookImportRow, 0, func(_ context.Context, v any) (any, error) {
		row := v.(subscriber.ImportRow)
		if strings.Contains(row.Email, "skip") {
			row.Email = ""
		}
		return row, nil
	})
	svc := subscriber.New(q, h)

	csv := "email,name\n" +
		"a@x.test,A\n" +
		"a@x.test,A again\n" + // duplicate → skipped
		"skip@x.test,S\n" + // filtered out → skipped
		"b@x.test,B\n"

	res, err := svc.ImportCSV(ctx, owner, l.ID, strings.NewReader(csv))
	if err != nil {
		t.Fatalf("ImportCSV: %v", err)
	}
	if res.Imported != 2 || res.Skipped != 2 {
		t.Fatalf("result = %+v, want Imported=2 Skipped=2", res)
	}
	subs, _ := svc.List(ctx, owner, l.ID)
	if len(subs) != 2 {
		t.Fatalf("list has %d subscribers, want 2", len(subs))
	}
}
