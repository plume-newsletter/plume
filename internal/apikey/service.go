// Package apikey issues and verifies workspace API keys. A key authenticates
// the same /api endpoints as the session cookie (acting as the workspace
// owner); only the SHA-256 hash is stored, so a key is shown exactly once.
package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

var ErrInvalid = errors.New("invalid api key")

const keyPrefix = "plume_"

// Key is a stored key's public metadata (never the secret).
type Key struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Prefix     string  `json:"prefix"`
	CreatedAt  string  `json:"createdAt"`
	LastUsedAt *string `json:"lastUsedAt"`
}

type Service struct{ q *gen.Queries }

func New(q *gen.Queries) *Service { return &Service{q: q} }

func hashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// Create generates a new key for owner, stores its hash, and returns both the
// public metadata and the raw secret (the only time the secret is available).
func (s *Service) Create(ctx context.Context, owner uuid.UUID, name string) (Key, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "API key"
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return Key{}, "", err
	}
	raw := keyPrefix + base64.RawURLEncoding.EncodeToString(b)
	row, err := s.q.CreateApiKey(ctx, gen.CreateApiKeyParams{
		ID:          uuid.New(),
		WorkspaceID: owner,
		Name:        name,
		Prefix:      raw[:14], // "plume_" + 8 chars, enough to recognize
		Hash:        hashKey(raw),
	})
	if err != nil {
		return Key{}, "", err
	}
	return Key{ID: row.ID.String(), Name: row.Name, Prefix: row.Prefix, CreatedAt: row.CreatedAt.Format("2006-01-02")}, raw, nil
}

func (s *Service) List(ctx context.Context, owner uuid.UUID) ([]Key, error) {
	rows, err := s.q.ListApiKeysForOwner(ctx, owner)
	if err != nil {
		return nil, err
	}
	out := make([]Key, 0, len(rows))
	for _, r := range rows {
		k := Key{ID: r.ID.String(), Name: r.Name, Prefix: r.Prefix, CreatedAt: r.CreatedAt.Format("2006-01-02")}
		if r.LastUsedAt.Valid {
			d := r.LastUsedAt.Time.Format("2006-01-02")
			k.LastUsedAt = &d
		}
		out = append(out, k)
	}
	return out, nil
}

func (s *Service) Delete(ctx context.Context, owner uuid.UUID, id uuid.UUID) error {
	return s.q.DeleteApiKey(ctx, gen.DeleteApiKeyParams{ID: id, WorkspaceID: owner})
}

// Authenticate resolves a raw key to its workspace id, or ErrInvalid. It
// touches last_used_at on success (best-effort).
func (s *Service) Authenticate(ctx context.Context, raw string) (uuid.UUID, error) {
	if !strings.HasPrefix(raw, keyPrefix) {
		return uuid.Nil, ErrInvalid
	}
	row, err := s.q.GetApiKeyByHash(ctx, hashKey(raw))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrInvalid
		}
		return uuid.Nil, err
	}
	_ = s.q.TouchApiKey(ctx, row.ID) // best-effort; never block auth on this
	return row.WorkspaceID, nil
}
