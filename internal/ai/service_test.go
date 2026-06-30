package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type stubMessager struct {
	gotModel, gotSystem, gotUser string
	gotMsgs                      []Message
	reply                        string
	err                          error
}

func (s *stubMessager) complete(_ context.Context, _ /*apiKey*/, model, system, user string) (string, error) {
	s.gotModel, s.gotSystem, s.gotUser = model, system, user
	return s.reply, s.err
}

func (s *stubMessager) completeChat(_ context.Context, _ /*apiKey*/, model, system string, msgs []Message) (string, error) {
	s.gotModel, s.gotSystem, s.gotMsgs = model, system, msgs
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

func TestChatHappyPathReturnsTrimmedReply(t *testing.T) {
	stub := &stubMessager{reply: "  Here is a draft.  "}
	out, err := New(stub).Chat(context.Background(), Config{APIKey: "k"},
		[]Message{{Role: "user", Content: "write me an email"}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if out != "Here is a draft." {
		t.Fatalf("out = %q", out)
	}
	if stub.gotModel != DefaultModel {
		t.Fatalf("model = %q, want default", stub.gotModel)
	}
	if len(stub.gotMsgs) != 1 || stub.gotMsgs[0].Content != "write me an email" {
		t.Fatalf("msgs = %+v", stub.gotMsgs)
	}
}

func TestChatValidation(t *testing.T) {
	svc := New(&stubMessager{reply: "x"})
	if _, err := svc.Chat(context.Background(), Config{APIKey: "k"}, nil); !errors.Is(err, ErrNoMessages) {
		t.Fatalf("empty slice: err = %v, want ErrNoMessages", err)
	}
	big := []Message{{Role: "user", Content: strings.Repeat("a", 12001)}}
	if _, err := svc.Chat(context.Background(), Config{APIKey: "k"}, big); !errors.Is(err, ErrTooLong) {
		t.Fatalf("oversize: err = %v, want ErrTooLong", err)
	}
}

func TestChatEmptyReplyIsRefusal(t *testing.T) {
	svc := New(&stubMessager{reply: "  "})
	if _, err := svc.Chat(context.Background(), Config{APIKey: "k"},
		[]Message{{Role: "user", Content: "hi"}}); !errors.Is(err, ErrRefused) {
		t.Fatalf("want ErrRefused, got %v", err)
	}
}

func TestSuggestParsesThreeSubjectLines(t *testing.T) {
	stub := &stubMessager{reply: "First subject\n\"Second subject\"\nThird subject\nExtra ignored"}
	opts, err := New(stub).Suggest(context.Background(), Config{APIKey: "k"}, "subject", "Our new feature launched today.")
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	want := []string{"First subject", "Second subject", "Third subject"}
	if len(opts) != 3 || opts[0] != want[0] || opts[1] != want[1] || opts[2] != want[2] {
		t.Fatalf("opts = %#v, want %#v", opts, want)
	}
}

func TestSuggestRejectsUnknownKindAndEmptyContext(t *testing.T) {
	svc := New(&stubMessager{reply: "a\nb\nc"})
	if _, err := svc.Suggest(context.Background(), Config{APIKey: "k"}, "preheader", "body"); !errors.Is(err, ErrBadAction) {
		t.Fatalf("unknown kind: err = %v, want ErrBadAction", err)
	}
	if _, err := svc.Suggest(context.Background(), Config{APIKey: "k"}, "subject", "   "); !errors.Is(err, ErrEmpty) {
		t.Fatalf("empty context: err = %v, want ErrEmpty", err)
	}
}
