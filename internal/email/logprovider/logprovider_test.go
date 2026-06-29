package logprovider

import (
	"bytes"
	"context"
	"testing"

	"github.com/plume-newsletter/plume/internal/email"
)

func TestLogProviderRecordsSentMessages(t *testing.T) {
	var buf bytes.Buffer
	p := New(&buf)

	msg := email.Message{From: "n@acme.test", To: "a@x.test", Subject: "Hi", HTML: "<p>Hi</p>"}
	if err := p.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if p.Name() != "log" {
		t.Fatalf("Name = %q, want log", p.Name())
	}
	sent := p.Sent()
	if len(sent) != 1 || sent[0].To != "a@x.test" || sent[0].Subject != "Hi" {
		t.Fatalf("Sent = %+v", sent)
	}
	if buf.Len() == 0 {
		t.Fatal("expected a log line written")
	}
}
