package email_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/crypto"
	"github.com/plume-newsletter/plume/internal/email"
	"github.com/plume-newsletter/plume/internal/email/logprovider"
	"github.com/plume-newsletter/plume/internal/email/ses"
	"github.com/plume-newsletter/plume/internal/settings"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

// sesBuilder adapts ses.NewFromCreds to email.SESBuilder for tests (avoids
// an import cycle in the email package itself).
func sesBuilder(ctx context.Context, accessKeyID, secret, region string) (email.Provider, error) {
	return ses.NewFromCreds(ctx, accessKeyID, secret, region)
}

func TestAdminResolverPicksSESWhenConfigured(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	_ = auth.EnsureAdmin(ctx, q, "a@plume.test", "pw-12345678")
	admin, _, _ := auth.Validate(ctx, q, "a@plume.test", "pw-12345678")

	cipher, _ := crypto.New([]byte("0123456789abcdef0123456789abcdef"))
	fallback := logprovider.New(&bytes.Buffer{})
	r := email.NewAdminResolver(q, cipher, fallback, sesBuilder)

	// No creds yet → fallback (log).
	p, err := r.Provider(ctx)
	if err != nil || p.Name() != "log" {
		t.Fatalf("no creds: name=%q err=%v (want log)", p.Name(), err)
	}

	// After configuring SES → ses provider.
	_ = settings.New(q, cipher).SetSES(ctx, admin.ID, settings.SESInput{
		AccessKeyID: "AKIATEST", SecretAccessKey: "secret", Region: "us-east-1",
	})
	p, err = r.Provider(ctx)
	if err != nil || p.Name() != "ses" {
		t.Fatalf("with creds: name=%q err=%v (want ses)", p.Name(), err)
	}
}

func TestAdminResolverFallsBackWhenNoAdmin(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()

	cipher, _ := crypto.New([]byte("0123456789abcdef0123456789abcdef"))
	fallback := logprovider.New(&bytes.Buffer{})
	r := email.NewAdminResolver(q, cipher, fallback, sesBuilder)

	p, err := r.Provider(ctx)
	if err != nil || p.Name() != "log" {
		t.Fatalf("no admin: name=%q err=%v (want log, no error)", p.Name(), err)
	}
}
