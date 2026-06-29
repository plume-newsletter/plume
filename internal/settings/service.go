// Package settings manages admin-level configuration (currently SES credentials).
package settings

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/plume-newsletter/plume/internal/crypto"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

// ErrNotFound is returned when the admin does not exist.
var ErrNotFound = errors.New("admin not found")

type Status struct {
	SESConfigured bool   `json:"sesConfigured"`
	SESRegion     string `json:"sesRegion"`
	AIConfigured  bool   `json:"aiConfigured"`
	AIModel       string `json:"aiModel"`
}

type SESInput struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
}

type Service struct {
	q      *gen.Queries
	cipher *crypto.Cipher
}

func New(q *gen.Queries, cipher *crypto.Cipher) *Service { return &Service{q: q, cipher: cipher} }

func (s *Service) Get(ctx context.Context, adminID uuid.UUID) (Status, error) {
	admin, err := s.q.GetAdminByID(ctx, adminID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Status{}, ErrNotFound
		}
		return Status{}, err
	}
	return Status{
		SESConfigured: admin.SesAccessKeyID != "",
		SESRegion:     admin.SesRegion,
		AIConfigured:  admin.AnthropicApiKey != "",
		AIModel:       admin.AiModel,
	}, nil
}

func (s *Service) SetSES(ctx context.Context, adminID uuid.UUID, in SESInput) error {
	enc, err := s.cipher.Encrypt(in.SecretAccessKey)
	if err != nil {
		return err
	}
	return s.q.SetAdminSESCreds(ctx, gen.SetAdminSESCredsParams{
		ID:                 adminID,
		SesAccessKeyID:     in.AccessKeyID,
		SesSecretAccessKey: enc,
		SesRegion:          in.Region,
	})
}

// SetAI stores the Anthropic API key (encrypted) and chosen model.
func (s *Service) SetAI(ctx context.Context, adminID uuid.UUID, apiKey, model string) error {
	enc, err := s.cipher.Encrypt(apiKey)
	if err != nil {
		return err
	}
	return s.q.SetAdminAIConfig(ctx, gen.SetAdminAIConfigParams{
		ID:              adminID,
		AnthropicApiKey: enc,
		AiModel:         model,
	})
}

// GetAIConfig returns the decrypted API key and model. apiKey is "" when unset.
func (s *Service) GetAIConfig(ctx context.Context, adminID uuid.UUID) (apiKey, model string, err error) {
	admin, err := s.q.GetAdminByID(ctx, adminID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrNotFound
		}
		return "", "", err
	}
	if admin.AnthropicApiKey == "" {
		return "", admin.AiModel, nil
	}
	key, err := s.cipher.Decrypt(admin.AnthropicApiKey)
	if err != nil {
		return "", "", err
	}
	return key, admin.AiModel, nil
}
