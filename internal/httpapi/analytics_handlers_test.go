package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/plume-newsletter/plume/internal/analytics"
	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/httpapi"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestAnalyticsOverview(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	cookie := auth.NewCookie([]byte("a-32-byte-or-longer-test-secret!!"))
	_, session := testsupport.SeedAdmin(t, pool, cookie, "analytics@plume.test", "pw-12345678")

	q := gen.New(pool)
	router := httpapi.NewRouter(httpapi.AuthDeps{
		Queries:   q,
		Cookie:    cookie,
		Brands:    brand.New(q),
		Analytics: analytics.New(q),
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/analytics/overview", nil)
	req.AddCookie(session)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body struct {
		Subscribers int `json:"subscribers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Subscribers != 0 {
		t.Errorf("subscribers = %d, want 0", body.Subscribers)
	}
}
