package subscriber

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/hooks"
)

const HookImportRow = "subscriber.import_row"

type ImportRow struct {
	Email string
	Name  string
}

type ImportResult struct {
	Imported int
	Skipped  int
	Failed   int
}

// ImportCSV reads a header + rows, runs each row through the import_row filter,
// and adds it (deduping). Rows the filter blanks (empty Email) are skipped.
func (s *Service) ImportCSV(ctx context.Context, owner, listID uuid.UUID, r io.Reader) (ImportResult, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	header, err := cr.Read()
	if err != nil {
		return ImportResult{}, errors.New("empty or invalid CSV")
	}
	emailCol, nameCol := -1, -1
	for i, h := range header {
		switch strings.ToLower(strings.TrimSpace(h)) {
		case "email":
			emailCol = i
		case "name":
			nameCol = i
		}
	}
	if emailCol < 0 {
		return ImportResult{}, errors.New("CSV must have an 'email' column")
	}

	var res ImportResult
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			res.Failed++
			continue
		}
		row := ImportRow{Email: field(rec, emailCol)}
		if nameCol >= 0 {
			row.Name = field(rec, nameCol)
		}
		row, ferr := hooks.Filter(ctx, s.h, HookImportRow, row)
		if ferr != nil {
			res.Failed++
			continue
		}
		if strings.TrimSpace(row.Email) == "" {
			res.Skipped++
			continue
		}
		_, created, aerr := s.Add(ctx, owner, listID, SubscriberInput{Email: row.Email, Name: row.Name})
		switch {
		case aerr != nil:
			res.Failed++
		case created:
			res.Imported++
		default:
			res.Skipped++ // duplicate
		}
	}
	return res, nil
}

func field(rec []string, i int) string {
	if i < 0 || i >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[i])
}
