package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBlocksRenderHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/blocks/render",
		strings.NewReader(`{"blocks":[{"type":"heading","text":"Hi","level":1}]}`))
	blocksHandlers{}.render(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	var body struct{ HTML string }
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if !strings.Contains(body.HTML, "Hi") || !strings.Contains(body.HTML, "max-width:600px") {
		t.Fatalf("html = %q", body.HTML)
	}
}

func TestBlocksRenderHandlerBadBlock(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/blocks/render",
		strings.NewReader(`{"blocks":[{"type":"nope"}]}`))
	blocksHandlers{}.render(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rec.Code)
	}
}
