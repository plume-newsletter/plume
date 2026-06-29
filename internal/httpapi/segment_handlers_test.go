package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/httpapi"
	"github.com/plume-newsletter/plume/internal/segment"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestSegmentPreviewEndpoint(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	cookie := auth.NewCookie([]byte("a-32-byte-or-longer-test-secret!!"))
	_, session := testsupport.SeedAdmin(t, pool, cookie, "seg@plume.test", "pw-12345678")

	q := gen.New(pool)
	router := httpapi.NewRouter(httpapi.AuthDeps{
		Queries:  q,
		Cookie:   cookie,
		Brands:   brand.New(q),
		Segments: segment.New(pool, q),
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	authedPost := func(path string, body *strings.Reader) *http.Response {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+path, body)
		req.AddCookie(session)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("authedPost %s: %v", path, err)
		}
		return resp
	}

	// Valid preview with empty conditions → 200, count present (0 on empty DB).
	resp := authedPost("/api/segments/preview", strings.NewReader(`{"match":"all","conditions":[]}`))
	if resp.StatusCode != 200 {
		t.Fatalf("preview status = %d, want 200", resp.StatusCode)
	}
	var pv struct {
		Count int `json:"count"`
	}
	json.NewDecoder(resp.Body).Decode(&pv)
	if pv.Count != 0 {
		t.Errorf("count = %d, want 0", pv.Count)
	}

	// Invalid condition → 400.
	bad := authedPost("/api/segments/preview", strings.NewReader(`{"match":"all","conditions":[{"type":"bogus","op":"x"}]}`))
	if bad.StatusCode != 400 {
		t.Errorf("bad preview status = %d, want 400", bad.StatusCode)
	}
}
