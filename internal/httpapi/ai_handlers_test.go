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

// stubAI implements aiRewriter for tests.
type stubAI struct{ reply string }

func (s stubAI) Rewrite(_ context.Context, _ ai.Config, _, _ string) (string, error) {
	return s.reply, nil
}

// stubCfg implements aiConfigGetter.
type stubCfg struct{ key, model string }

func (s stubCfg) GetAIConfig(_ context.Context, _ uuid.UUID) (string, string, error) {
	return s.key, s.model, nil
}

func newAIHandler(reply, key string) aiHandlers {
	return aiHandlers{ai: stubAI{reply: reply}, cfg: stubCfg{key: key, model: ""}}
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
