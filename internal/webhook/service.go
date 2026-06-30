// Package webhook stores outbound webhook endpoints and delivers signed event
// payloads to them. Delivery is best-effort: each POST runs in its own
// goroutine with a short timeout and is not retried.
//
// ponytail: fire-and-forget delivery, no retry/queue. Add a delivery table +
// retry worker if at-least-once delivery is ever required.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

var ErrInvalid = errors.New("invalid webhook")

// Events is the catalog of deliverable event names, shown in the UI and
// validated on create.
var Events = []string{"subscriber.created", "subscriber.confirmed", "campaign.sent"}

func validEvent(e string) bool {
	for _, v := range Events {
		if v == e {
			return true
		}
	}
	return false
}

// Endpoint is a stored webhook endpoint as returned to the UI (secret included
// so the user can verify the X-Plume-Signature on their receiver).
type Endpoint struct {
	ID        string   `json:"id"`
	URL       string   `json:"url"`
	Secret    string   `json:"secret"`
	Events    []string `json:"events"`
	Active    bool     `json:"active"`
	CreatedAt string   `json:"createdAt"`
}

type Service struct {
	q      *gen.Queries
	client *http.Client
}

func New(q *gen.Queries) *Service {
	return &Service{q: q, client: &http.Client{Timeout: 10 * time.Second}}
}

func toEndpoint(row gen.WebhookEndpoint) Endpoint {
	return Endpoint{
		ID: row.ID.String(), URL: row.Url, Secret: row.Secret,
		Events: row.Events, Active: row.Active, CreatedAt: row.CreatedAt.Format("2006-01-02"),
	}
}

func (s *Service) Create(ctx context.Context, owner uuid.UUID, url string, events []string) (Endpoint, error) {
	url = strings.TrimSpace(url)
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return Endpoint{}, ErrInvalid
	}
	clean := make([]string, 0, len(events))
	for _, e := range events {
		if validEvent(e) {
			clean = append(clean, e)
		}
	}
	if len(clean) == 0 {
		return Endpoint{}, ErrInvalid
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return Endpoint{}, err
	}
	row, err := s.q.CreateWebhook(ctx, gen.CreateWebhookParams{
		ID:          uuid.New(),
		WorkspaceID: owner,
		Url:         url,
		Secret:      "whsec_" + base64.RawURLEncoding.EncodeToString(b),
		Events:      clean,
	})
	if err != nil {
		return Endpoint{}, err
	}
	return toEndpoint(row), nil
}

func (s *Service) List(ctx context.Context, owner uuid.UUID) ([]Endpoint, error) {
	rows, err := s.q.ListWebhooksForOwner(ctx, owner)
	if err != nil {
		return nil, err
	}
	out := make([]Endpoint, 0, len(rows))
	for _, r := range rows {
		out = append(out, toEndpoint(r))
	}
	return out, nil
}

func (s *Service) Delete(ctx context.Context, owner, id uuid.UUID) error {
	return s.q.DeleteWebhook(ctx, gen.DeleteWebhookParams{ID: id, WorkspaceID: owner})
}

// Deliver sends event+data to every active endpoint of owner subscribed to
// event. It returns immediately; each POST is signed and fired in a goroutine.
func (s *Service) Deliver(ctx context.Context, owner uuid.UUID, event string, data any) {
	rows, err := s.q.ListActiveWebhooksForOwner(ctx, gen.ListActiveWebhooksForOwnerParams{WorkspaceID: owner, Event: event})
	if err != nil {
		log.Printf("webhook: list endpoints for %s/%s: %v", owner, event, err)
		return
	}
	if len(rows) == 0 {
		return
	}
	body, err := json.Marshal(map[string]any{
		"event":  event,
		"data":   data,
		"sentAt": time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		log.Printf("webhook: marshal %s payload: %v", event, err)
		return
	}
	for _, r := range rows {
		go s.post(r.Url, r.Secret, event, body)
	}
}

func (s *Service) post(url, secret, event string, body []byte) {
	// Detach from the request context: delivery outlives the triggering request.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("webhook: build request to %s: %v", url, err)
		return
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Plume-Event", event)
	req.Header.Set("X-Plume-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("webhook: POST %s: %v", url, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("webhook: POST %s returned %d", url, resp.StatusCode)
	}
}
