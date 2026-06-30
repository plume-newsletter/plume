package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/ai"
)

// stubAI implements aiService for tests.
type stubAI struct {
	reply   string
	options []string
}

func (s stubAI) Rewrite(_ context.Context, _ ai.Config, _, _ string) (string, error) { return s.reply, nil }
func (s stubAI) Chat(_ context.Context, _ ai.Config, _ []ai.Message) (string, error) { return s.reply, nil }
func (s stubAI) Suggest(_ context.Context, _ ai.Config, _, _ string) ([]string, error) {
	return s.options, nil
}
func (s stubAI) Insights(_ context.Context, _ ai.Config, _ string) ([]ai.Insight, error) {
	return nil, nil
}
func (s stubAI) SegmentRules(_ context.Context, _ ai.Config, _ string, _ []string) (ai.SegmentRules, error) {
	return ai.SegmentRules{}, nil
}

// stubCfg implements aiConfigGetter.
type stubCfg struct{ key, model string }

func (s stubCfg) GetAIConfig(_ context.Context, _ uuid.UUID) (string, string, error) {
	return s.key, s.model, nil
}

func newAIHandler(reply, key string) aiHandlers {
	return aiHandlers{ai: stubAI{reply: reply, options: []string{"A", "B", "C"}}, cfg: stubCfg{key: key, model: ""}}
}

// withOwner injects an admin id the same way requireAuth would.
func withOwner(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), adminIDKey, uuid.New()))
}

func TestRewriteHandlerHappyPath(t *testing.T) {
	h := newAIHandler("Edited copy.", "sk-ant-key")
	req := withOwner(httptest.NewRequest("POST", "/api/ai/rewrite",
		strings.NewReader(`{"action":"rewrite","text":"hello world"}`)))
	rec := httptest.NewRecorder()
	h.rewrite(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	var body struct{ Text string }
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body.Text != "Edited copy." {
		t.Fatalf("text = %q", body.Text)
	}
}

func TestRewriteHandlerNotConfigured(t *testing.T) {
	h := newAIHandler("x", "") // no key
	req := withOwner(httptest.NewRequest("POST", "/api/ai/rewrite",
		strings.NewReader(`{"action":"rewrite","text":"hello"}`)))
	rec := httptest.NewRecorder()
	h.rewrite(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rec.Code)
	}
}

func TestChatHandlerHappyPath(t *testing.T) {
	h := newAIHandler("Here is a draft.", "sk-ant-key")
	req := withOwner(httptest.NewRequest("POST", "/api/ai/chat",
		strings.NewReader(`{"messages":[{"role":"user","content":"hi"}]}`)))
	rec := httptest.NewRecorder()
	h.chat(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	var body struct{ Reply string }
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body.Reply != "Here is a draft." {
		t.Fatalf("reply = %q", body.Reply)
	}
}

func TestChatHandlerNotConfigured(t *testing.T) {
	h := newAIHandler("x", "")
	req := withOwner(httptest.NewRequest("POST", "/api/ai/chat",
		strings.NewReader(`{"messages":[{"role":"user","content":"hi"}]}`)))
	rec := httptest.NewRecorder()
	h.chat(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rec.Code)
	}
}

func TestSuggestHandlerHappyPath(t *testing.T) {
	h := newAIHandler("x", "sk-ant-key")
	req := withOwner(httptest.NewRequest("POST", "/api/ai/suggest",
		strings.NewReader(`{"kind":"subject","context":"our launch"}`)))
	rec := httptest.NewRecorder()
	h.suggest(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	var body struct{ Options []string }
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if len(body.Options) != 3 || body.Options[0] != "A" {
		t.Fatalf("options = %#v", body.Options)
	}
}

func TestSuggestHandlerNotConfigured(t *testing.T) {
	h := newAIHandler("x", "")
	req := withOwner(httptest.NewRequest("POST", "/api/ai/suggest",
		strings.NewReader(`{"kind":"subject","context":"our launch"}`)))
	rec := httptest.NewRecorder()
	h.suggest(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rec.Code)
	}
}
