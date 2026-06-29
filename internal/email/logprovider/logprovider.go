// Package logprovider is the zero-config default EmailProvider: it records and
// logs messages instead of sending them, so Plume runs and tests with no AWS.
package logprovider

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/plume-newsletter/plume/internal/email"
)

type Provider struct {
	w   io.Writer
	mu  sync.Mutex
	out []email.Message
}

func New(w io.Writer) *Provider { return &Provider{w: w} }

func (p *Provider) Name() string { return "log" }

func (p *Provider) Send(_ context.Context, msg email.Message) error {
	p.mu.Lock()
	p.out = append(p.out, msg)
	p.mu.Unlock()
	_, err := fmt.Fprintf(p.w, "[email:log] to=%s subject=%q bytes=%d\n", msg.To, msg.Subject, len(msg.HTML))
	return err
}

// Sent returns a copy of all recorded messages (test/inspection helper).
func (p *Provider) Sent() []email.Message {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]email.Message(nil), p.out...)
}
