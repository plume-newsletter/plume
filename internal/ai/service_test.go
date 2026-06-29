package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type stubMessager struct {
	gotModel, gotSystem, gotUser string
	reply                        string
	err                          error
}

func (s *stubMessager) complete(_ context.Context, _ /*apiKey*/, model, system, user string) (string, error) {
	s.gotModel, s.gotSystem, s.gotUser = model, system, user
	return s.reply, s.err
}

func TestRewriteHappyPathSendsInstructionAndReturnsText(t *testing.T) {
	stub := &stubMessager{reply: "  Tighter copy.  "}
	svc := New(stub)
	out, err := svc.Rewrite(context.Background(), Config{APIKey: "k", Model: "claude-haiku-4-5"}, "shorten", "make this shorter please")
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	if out != "Tighter copy." { // trimmed
		t.Fatalf("out = %q", out)
	}
	if stub.gotModel != "claude-haiku-4-5" {
		t.Fatalf("model = %q", stub.gotModel)
	}
	if !strings.Contains(stub.gotUser, "make this shorter please") {
		t.Fatalf("user prompt missing input text: %q", stub.gotUser)
	}
}

func TestRewriteDefaultsModelWhenEmpty(t *testing.T) {
	stub := &stubMessager{reply: "ok"}
	if _, err := New(stub).Rewrite(context.Background(), Config{APIKey: "k"}, "rewrite", "hello"); err != nil {
		t.Fatal(err)
	}
	if stub.gotModel != DefaultModel {
		t.Fatalf("model = %q, want %q", stub.gotModel, DefaultModel)
	}
}

func TestRewriteValidation(t *testing.T) {
	svc := New(&stubMessager{reply: "x"})
	cases := []struct {
		name, action, text string
		want               error
	}{
		{"empty", "rewrite", "   ", ErrEmpty},
		{"toolong", "rewrite", strings.Repeat("a", 4001), ErrTooLong},
		{"badaction", "explode", "hi", ErrBadAction},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := svc.Rewrite(context.Background(), Config{APIKey: "k"}, c.action, c.text); !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
		})
	}
}

func TestRewriteEmptyReplyIsRefusal(t *testing.T) {
	svc := New(&stubMessager{reply: "   "})
	if _, err := svc.Rewrite(context.Background(), Config{APIKey: "k"}, "rewrite", "hello"); !errors.Is(err, ErrRefused) {
		t.Fatalf("want ErrRefused, got %v", err)
	}
}
