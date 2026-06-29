package settings_test

import (
	"context"
	"testing"

	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/crypto"
	"github.com/plume-newsletter/plume/internal/settings"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestSetAndGetSESStatusNeverExposesSecret(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	if err := auth.EnsureAdmin(ctx, q, "a@plume.test", "pw-12345678"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	admin, _, _ := auth.Validate(ctx, q, "a@plume.test", "pw-12345678")

	cipher, err := crypto.New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	svc := settings.New(q, cipher)

	// Initially not configured.
	st, err := svc.Get(ctx, admin.ID)
	if err != nil || st.SESConfigured {
		t.Fatalf("initial Get: %+v err=%v (want not configured)", st, err)
	}

	if err := svc.SetSES(ctx, admin.ID, settings.SESInput{
		AccessKeyID: "AKIATEST", SecretAccessKey: "super-secret", Region: "us-east-1",
	}); err != nil {
		t.Fatalf("SetSES: %v", err)
	}

	st, err = svc.Get(ctx, admin.ID)
	if err != nil || !st.SESConfigured || st.SESRegion != "us-east-1" {
		t.Fatalf("after SetSES: %+v err=%v", st, err)
	}

	// The stored secret must be encrypted (not equal to plaintext).
	reloaded, _ := q.GetAdminByID(ctx, admin.ID)
	if reloaded.SesSecretAccessKey == "super-secret" {
		t.Fatal("secret stored in plaintext")
	}
	if reloaded.SesSecretAccessKey == "" {
		t.Fatal("secret not stored")
	}
}

func TestSetAndGetAIConfigEncryptsAndRoundTrips(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	if err := auth.EnsureAdmin(ctx, q, "ai@plume.test", "pw-12345678"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	admin, _, _ := auth.Validate(ctx, q, "ai@plume.test", "pw-12345678")

	cipher, err := crypto.New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	svc := settings.New(q, cipher)

	st, _ := svc.Get(ctx, admin.ID)
	if st.AIConfigured {
		t.Fatal("should start unconfigured")
	}

	if err := svc.SetAI(ctx, admin.ID, "sk-ant-secret", "claude-haiku-4-5"); err != nil {
		t.Fatalf("SetAI: %v", err)
	}

	key, model, err := svc.GetAIConfig(ctx, admin.ID)
	if err != nil || key != "sk-ant-secret" || model != "claude-haiku-4-5" {
		t.Fatalf("GetAIConfig = %q,%q err=%v", key, model, err)
	}

	st, _ = svc.Get(ctx, admin.ID)
	if !st.AIConfigured || st.AIModel != "claude-haiku-4-5" {
		t.Fatalf("status = %+v", st)
	}

	reloaded, _ := q.GetAdminByID(ctx, admin.ID)
	if reloaded.AnthropicApiKey == "sk-ant-secret" || reloaded.AnthropicApiKey == "" {
		t.Fatalf("key not encrypted/stored: %q", reloaded.AnthropicApiKey)
	}
}
