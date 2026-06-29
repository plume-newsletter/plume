package email

import (
	"context"
	"errors"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/plume-newsletter/plume/internal/crypto"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

// SESBuilder constructs a Provider from static SES credentials. It exists so
// this package (which the ses package imports, for the Provider/Message
// seam) does not itself import ses — that would be an import cycle. Callers
// (main.go) inject ses.NewFromCreds.
type SESBuilder func(ctx context.Context, accessKeyID, secretAccessKey, region string) (Provider, error)

// Resolver returns the Provider to send through right now. Implementations
// must be safe for concurrent use: both the sending.Worker goroutine and
// signup HTTP handlers call Provider concurrently.
type Resolver interface {
	Provider(ctx context.Context) (Provider, error)
}

// AdminResolver returns a SES provider built from the single admin's stored
// credentials when present, else a fallback (log) provider. Safe for
// concurrent use.
type AdminResolver struct {
	q        *gen.Queries
	cipher   *crypto.Cipher
	fallback Provider
	build    SESBuilder

	mu        sync.Mutex
	cachedKey string   // ses_access_key_id+"|"+region the cached provider was built from
	cached    Provider // last built SES provider
}

// NewAdminResolver builds an AdminResolver. fallback is returned whenever no
// admin exists yet or the admin has no SES credentials stored. build
// constructs the real SES-backed Provider (inject ses.NewFromCreds).
func NewAdminResolver(q *gen.Queries, cipher *crypto.Cipher, fallback Provider, build SESBuilder) *AdminResolver {
	return &AdminResolver{q: q, cipher: cipher, fallback: fallback, build: build}
}

// Provider resolves the current send provider. A missing admin row or absent
// credentials are not errors: sends must not fail just because SES has not
// been configured yet, so both cases return the fallback provider with a nil
// error. Only a real failure (decrypt failure, SES client construction
// failure, or a non-ErrNoRows DB error) returns a non-nil error.
func (r *AdminResolver) Provider(ctx context.Context) (Provider, error) {
	admin, err := r.q.GetSingleAdmin(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.fallback, nil // no admin yet → fall back, don't fail sends
		}
		return nil, err
	}
	if admin.SesAccessKeyID == "" {
		return r.fallback, nil
	}
	key := admin.SesAccessKeyID + "|" + admin.SesRegion

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cached != nil && r.cachedKey == key {
		return r.cached, nil
	}
	secret, err := r.cipher.Decrypt(admin.SesSecretAccessKey)
	if err != nil {
		return nil, err
	}
	p, err := r.build(ctx, admin.SesAccessKeyID, secret, admin.SesRegion)
	if err != nil {
		return nil, err
	}
	r.cached, r.cachedKey = p, key
	return p, nil
}

// staticResolver always returns the same Provider. Useful for tests and
// call sites that have not (yet) configured admin-based resolution.
type staticResolver struct{ p Provider }

// NewStaticResolver wraps a fixed Provider as a Resolver.
func NewStaticResolver(p Provider) Resolver { return staticResolver{p: p} }

func (s staticResolver) Provider(context.Context) (Provider, error) { return s.p, nil }

// NoopResolver returns a Resolver whose Provider always returns a no-op
// Provider that silently discards messages. Useful in tests.
func NoopResolver() Resolver { return NewStaticResolver(noopProvider{}) }

type noopProvider struct{}

func (noopProvider) Send(context.Context, Message) error { return nil }
func (noopProvider) Name() string                        { return "noop" }
