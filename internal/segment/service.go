// Package segment compiles a JSON condition list into a parameterized SQL
// predicate over an owner's subscribers and evaluates it live (count + sample),
// plus segment CRUD.
package segment

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

var ErrInvalidCondition = errors.New("invalid condition")
var ErrNotFound = errors.New("segment not found")

type Condition struct {
	Type  string `json:"type"`
	Op    string `json:"op"`
	Days  int    `json:"days,omitempty"`
	Field string `json:"field,omitempty"`
	Value string `json:"value,omitempty"`
}
type SubscriberLite struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Status string `json:"status"`
}
type Preview struct {
	Count   int              `json:"count"`
	Total   int              `json:"total"`
	Percent float64          `json:"percent"`
	Sample  []SubscriberLite `json:"sample"`
}

type Segment struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Match      string      `json:"match"`
	Conditions []Condition `json:"conditions"`
	Count      int         `json:"count"`
	CreatedAt  string      `json:"createdAt"`
}

type Service struct {
	pool *pgxpool.Pool
	q    *gen.Queries
}

func New(pool *pgxpool.Pool, q *gen.Queries) *Service { return &Service{pool: pool, q: q} }

// add appends a bind value and returns its placeholder ($N).
func add(args *[]any, v any) string {
	*args = append(*args, v)
	return "$" + strconv.Itoa(len(*args))
}

// compile returns a SQL fragment (with $N placeholders) for one condition,
// appending bound params to args. All dynamic data is bound — only whitelisted
// type/op values choose constant SQL.
func compile(c Condition, args *[]any) (string, error) {
	switch c.Type {
	case "opened", "clicked":
		et := "open"
		if c.Type == "clicked" {
			et = "click"
		}
		base := "SELECT 1 FROM email_event e JOIN campaign c ON c.id = e.campaign_id WHERE e.subscriber_id = s.id AND c.owner_id = s.owner_id AND e.type = " + add(args, et)
		switch c.Op {
		case "in_last":
			if c.Days <= 0 {
				return "", ErrInvalidCondition
			}
			return "EXISTS (" + base + " AND e.created_at >= now() - make_interval(days => " + add(args, c.Days) + "))", nil
		case "ever":
			return "EXISTS (" + base + ")", nil
		case "never":
			return "NOT EXISTS (" + base + ")", nil
		default:
			return "", ErrInvalidCondition
		}
	case "field":
		if c.Field == "" {
			return "", ErrInvalidCondition
		}
		fname := add(args, c.Field)
		base := "SELECT 1 FROM subscriber_field_value v JOIN custom_field cf ON cf.id = v.custom_field_id WHERE v.subscriber_id = s.id AND cf.name = " + fname
		switch c.Op {
		case "equals":
			return "EXISTS (" + base + " AND v.value = " + add(args, c.Value) + ")", nil
		case "not_equals":
			return "NOT EXISTS (" + base + " AND v.value = " + add(args, c.Value) + ")", nil
		case "contains":
			return "EXISTS (" + base + " AND v.value ILIKE '%' || " + add(args, c.Value) + " || '%')", nil
		default:
			return "", ErrInvalidCondition
		}
	case "status":
		switch c.Value {
		case "active", "pending", "unsubscribed":
		default:
			return "", ErrInvalidCondition
		}
		switch c.Op {
		case "is":
			return "s.status = " + add(args, c.Value), nil
		case "is_not":
			return "s.status <> " + add(args, c.Value), nil
		default:
			return "", ErrInvalidCondition
		}
	default:
		return "", ErrInvalidCondition
	}
}

// buildWhere returns the full WHERE body (owner filter + combined conditions)
// and its args, with $1 = owner.
func buildWhere(owner uuid.UUID, match string, conds []Condition) (string, []any, error) {
	if match != "all" && match != "any" {
		match = "all"
	}
	args := []any{owner}
	frags := make([]string, 0, len(conds))
	for _, c := range conds {
		f, err := compile(c, &args)
		if err != nil {
			return "", nil, err
		}
		frags = append(frags, f)
	}
	where := "s.owner_id = $1"
	if len(frags) > 0 {
		joiner := " AND "
		if match == "any" {
			joiner = " OR "
		}
		where += " AND (" + strings.Join(frags, joiner) + ")"
	}
	return where, args, nil
}

func (s *Service) Preview(ctx context.Context, owner uuid.UUID, match string, conds []Condition) (Preview, error) {
	where, args, err := buildWhere(owner, match, conds)
	if err != nil {
		return Preview{}, err
	}
	var p Preview
	if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM subscriber s WHERE "+where, args...).Scan(&p.Count); err != nil {
		return Preview{}, err
	}
	if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM subscriber WHERE owner_id = $1", owner).Scan(&p.Total); err != nil {
		return Preview{}, err
	}
	if p.Total > 0 {
		p.Percent = float64(p.Count) / float64(p.Total)
	}
	rows, err := s.pool.Query(ctx, "SELECT s.id, s.email, s.name, s.status FROM subscriber s WHERE "+where+" ORDER BY s.created_at DESC LIMIT 20", args...)
	if err != nil {
		return Preview{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var sl SubscriberLite
		if err := rows.Scan(&id, &sl.Email, &sl.Name, &sl.Status); err != nil {
			return Preview{}, err
		}
		sl.ID = id.String()
		p.Sample = append(p.Sample, sl)
	}
	return p, rows.Err()
}

// countMatching runs only the count query for a segment's predicate — used when
// listing/getting saved segments, which need the count but not the total/sample
// that full Preview computes (avoids 3 queries per segment on List).
func (s *Service) countMatching(ctx context.Context, owner uuid.UUID, match string, conds []Condition) (int, error) {
	where, args, err := buildWhere(owner, match, conds)
	if err != nil {
		return 0, err
	}
	var count int
	if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM subscriber s WHERE "+where, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Service) toSegment(ctx context.Context, row gen.Segment) (Segment, error) {
	var conds []Condition
	if len(row.Conditions) > 0 {
		if err := json.Unmarshal(row.Conditions, &conds); err != nil {
			return Segment{}, err
		}
	}
	count, err := s.countMatching(ctx, row.OwnerID, row.Match, conds)
	if err != nil {
		return Segment{}, err
	}
	return Segment{
		ID: row.ID.String(), Name: row.Name, Match: row.Match,
		Conditions: conds, Count: count, CreatedAt: row.CreatedAt.Format(time.RFC3339),
	}, nil
}

func marshalConds(conds []Condition) ([]byte, error) {
	if conds == nil {
		conds = []Condition{}
	}
	return json.Marshal(conds)
}

func (s *Service) Create(ctx context.Context, owner uuid.UUID, name, match string, conds []Condition) (Segment, error) {
	if match != "all" && match != "any" {
		match = "all"
	}
	if name == "" {
		return Segment{}, ErrInvalidCondition
	}
	// validate conditions compile before persisting
	if _, _, err := buildWhere(owner, match, conds); err != nil {
		return Segment{}, err
	}
	raw, err := marshalConds(conds)
	if err != nil {
		return Segment{}, err
	}
	row, err := s.q.CreateSegment(ctx, gen.CreateSegmentParams{ID: uuid.New(), OwnerID: owner, Name: name, Match: match, Conditions: raw})
	if err != nil {
		return Segment{}, err
	}
	return s.toSegment(ctx, row)
}

func (s *Service) List(ctx context.Context, owner uuid.UUID) ([]Segment, error) {
	rows, err := s.q.ListSegmentsByOwner(ctx, owner)
	if err != nil {
		return nil, err
	}
	out := make([]Segment, 0, len(rows))
	for _, r := range rows {
		seg, err := s.toSegment(ctx, r)
		if err != nil {
			return nil, err
		}
		out = append(out, seg)
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, owner, id uuid.UUID) (Segment, error) {
	row, err := s.q.GetSegmentForOwner(ctx, gen.GetSegmentForOwnerParams{ID: id, OwnerID: owner})
	if errors.Is(err, pgx.ErrNoRows) {
		return Segment{}, ErrNotFound
	}
	if err != nil {
		return Segment{}, err
	}
	return s.toSegment(ctx, row)
}

func (s *Service) Update(ctx context.Context, owner, id uuid.UUID, name, match string, conds []Condition) (Segment, error) {
	if match != "all" && match != "any" {
		match = "all"
	}
	if name == "" {
		return Segment{}, ErrInvalidCondition
	}
	if _, _, err := buildWhere(owner, match, conds); err != nil {
		return Segment{}, err
	}
	raw, err := marshalConds(conds)
	if err != nil {
		return Segment{}, err
	}
	row, err := s.q.UpdateSegment(ctx, gen.UpdateSegmentParams{ID: id, OwnerID: owner, Name: name, Match: match, Conditions: raw})
	if errors.Is(err, pgx.ErrNoRows) {
		return Segment{}, ErrNotFound
	}
	if err != nil {
		return Segment{}, err
	}
	return s.toSegment(ctx, row)
}

func (s *Service) Delete(ctx context.Context, owner, id uuid.UUID) error {
	return s.q.DeleteSegment(ctx, gen.DeleteSegmentParams{ID: id, OwnerID: owner})
}

func (s *Service) FieldNames(ctx context.Context, owner uuid.UUID) ([]string, error) {
	names, err := s.q.ListCustomFieldNamesForOwner(ctx, owner)
	if err != nil {
		return nil, err
	}
	if names == nil {
		names = []string{}
	}
	return names, nil
}
