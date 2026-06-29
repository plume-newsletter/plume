// Package automation runs linear trigger-based email journeys: an automation has
// an ordered list of steps (send/wait); subscribers who confirm into its list are
// enrolled (via the subscriber.confirmed hook) and advanced by the worker.
package automation

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/signup"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

var ErrNotFound = errors.New("automation not found")
var ErrInvalid = errors.New("invalid")

type Step struct {
	Kind     string `json:"kind"`
	Subject  string `json:"subject"`
	HTML     string `json:"html"`
	WaitDays int    `json:"waitDays"`
}

type Automation struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	ListID      string  `json:"listId"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"createdAt"`
	Steps       []Step  `json:"steps"`
	StepSends   int     `json:"stepSends"`
	InFlow      int     `json:"inFlow"`
	CompletePct float64 `json:"completePct"`
}

type Service struct {
	pool *pgxpool.Pool
	q    *gen.Queries
}

func New(pool *pgxpool.Pool, q *gen.Queries, h *hooks.Hooks) *Service {
	s := &Service{pool: pool, q: q}
	h.AddAction(signup.HookConfirmed, 50, func(ctx context.Context, payload any) error {
		if p, ok := payload.(signup.ConfirmedPayload); ok {
			return s.Enroll(ctx, p.Subscriber)
		}
		return nil
	})
	return s
}

func validStatus(st string) bool { return st == "draft" || st == "live" || st == "paused" }

func validateSteps(steps []Step) error {
	for _, st := range steps {
		switch st.Kind {
		case "send":
			if strings.TrimSpace(st.Subject) == "" {
				return ErrInvalid
			}
		case "wait":
			if st.WaitDays < 1 {
				return ErrInvalid
			}
		default:
			return ErrInvalid
		}
	}
	return nil
}

func (s *Service) get(ctx context.Context, owner, id uuid.UUID) (gen.Automation, error) {
	a, err := s.q.GetAutomationForOwner(ctx, gen.GetAutomationForOwnerParams{ID: id, OwnerID: owner})
	if errors.Is(err, pgx.ErrNoRows) {
		return gen.Automation{}, ErrNotFound
	}
	return a, err
}

func (s *Service) toAutomation(ctx context.Context, a gen.Automation) (Automation, error) {
	stepRows, err := s.q.ListStepsForAutomation(ctx, a.ID)
	if err != nil {
		return Automation{}, err
	}
	steps := make([]Step, 0, len(stepRows))
	sends := 0
	for _, r := range stepRows {
		if r.Kind == "send" {
			sends++
		}
		steps = append(steps, Step{Kind: r.Kind, Subject: r.Subject, HTML: r.Html, WaitDays: int(r.WaitDays)})
	}
	active, err := s.q.CountEnrollmentsByStatus(ctx, gen.CountEnrollmentsByStatusParams{AutomationID: a.ID, Status: "active"})
	if err != nil {
		return Automation{}, err
	}
	complete, err := s.q.CountEnrollmentsByStatus(ctx, gen.CountEnrollmentsByStatusParams{AutomationID: a.ID, Status: "complete"})
	if err != nil {
		return Automation{}, err
	}
	pct := 0.0
	if total := active + complete; total > 0 {
		pct = float64(complete) / float64(total)
	}
	return Automation{
		ID: a.ID.String(), Name: a.Name, ListID: a.ListID.String(), Status: a.Status,
		CreatedAt: a.CreatedAt.Format(time.RFC3339), Steps: steps, StepSends: sends,
		InFlow: int(active), CompletePct: pct,
	}, nil
}

func (s *Service) Create(ctx context.Context, owner uuid.UUID, name string, listID uuid.UUID) (Automation, error) {
	if strings.TrimSpace(name) == "" {
		return Automation{}, ErrInvalid
	}
	if _, err := s.q.GetListForOwner(ctx, gen.GetListForOwnerParams{ID: listID, OwnerID: owner}); err != nil {
		return Automation{}, ErrInvalid
	}
	a, err := s.q.CreateAutomation(ctx, gen.CreateAutomationParams{ID: uuid.New(), OwnerID: owner, Name: name, ListID: listID})
	if err != nil {
		return Automation{}, err
	}
	return s.toAutomation(ctx, a)
}

func (s *Service) List(ctx context.Context, owner uuid.UUID) ([]Automation, error) {
	rows, err := s.q.ListAutomationsByOwner(ctx, owner)
	if err != nil {
		return nil, err
	}
	out := make([]Automation, 0, len(rows))
	for _, a := range rows {
		am, err := s.toAutomation(ctx, a)
		if err != nil {
			return nil, err
		}
		out = append(out, am)
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, owner, id uuid.UUID) (Automation, error) {
	a, err := s.get(ctx, owner, id)
	if err != nil {
		return Automation{}, err
	}
	return s.toAutomation(ctx, a)
}

func (s *Service) Update(ctx context.Context, owner, id uuid.UUID, name string, listID uuid.UUID) (Automation, error) {
	if strings.TrimSpace(name) == "" {
		return Automation{}, ErrInvalid
	}
	if _, err := s.q.GetListForOwner(ctx, gen.GetListForOwnerParams{ID: listID, OwnerID: owner}); err != nil {
		return Automation{}, ErrInvalid
	}
	a, err := s.q.UpdateAutomation(ctx, gen.UpdateAutomationParams{ID: id, OwnerID: owner, Name: name, ListID: listID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Automation{}, ErrNotFound
	}
	if err != nil {
		return Automation{}, err
	}
	return s.toAutomation(ctx, a)
}

func (s *Service) Delete(ctx context.Context, owner, id uuid.UUID) error {
	return s.q.DeleteAutomation(ctx, gen.DeleteAutomationParams{ID: id, OwnerID: owner})
}

func (s *Service) ReplaceSteps(ctx context.Context, owner, id uuid.UUID, steps []Step) error {
	if _, err := s.get(ctx, owner, id); err != nil {
		return err
	}
	if err := validateSteps(steps); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	qtx := s.q.WithTx(tx)
	if err := qtx.DeleteStepsForAutomation(ctx, id); err != nil {
		return err
	}
	for i, st := range steps {
		if err := qtx.CreateStep(ctx, gen.CreateStepParams{
			ID: uuid.New(), AutomationID: id, Position: int32(i), Kind: st.Kind,
			Subject: st.Subject, Html: st.HTML, WaitDays: int32(st.WaitDays),
		}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Service) SetStatus(ctx context.Context, owner, id uuid.UUID, status string) error {
	if !validStatus(status) {
		return ErrInvalid
	}
	if _, err := s.get(ctx, owner, id); err != nil {
		return err
	}
	return s.q.SetAutomationStatus(ctx, gen.SetAutomationStatusParams{ID: id, OwnerID: owner, Status: status})
}

func (s *Service) Enroll(ctx context.Context, sub gen.Subscriber) error {
	autos, err := s.q.ListLiveAutomationsForList(ctx, gen.ListLiveAutomationsForListParams{ListID: sub.ListID, OwnerID: sub.OwnerID})
	if err != nil {
		return err
	}
	for _, a := range autos {
		if err := s.q.CreateEnrollment(ctx, gen.CreateEnrollmentParams{ID: uuid.New(), AutomationID: a.ID, SubscriberID: sub.ID}); err != nil {
			return err
		}
	}
	return nil
}
