package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/plume-newsletter/plume/internal/abtest"
	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/campaign"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/httpapi"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestABTestEndpoints(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	cookie := auth.NewCookie([]byte("a-32-byte-or-longer-test-secret!!"))
	_, session := testsupport.SeedAdmin(t, pool, cookie, "abtest@plume.test", "pw-12345678")

	q := gen.New(pool)
	h := hooks.New()
	router := httpapi.NewRouter(httpapi.AuthDeps{
		Queries:     q,
		Cookie:      cookie,
		Brands:      brand.New(q),
		Lists:       list.New(q),
		Campaigns:   campaign.New(q),
		Subscribers: subscriber.New(q, h),
		ABTests:     abtest.New(q),
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	authedPost := func(path, body string) *http.Response {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewBufferString(body))
		req.AddCookie(session)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		return resp
	}
	authedGet := func(path string) *http.Response {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		req.AddCookie(session)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		return resp
	}
	decodeID := func(t *testing.T, r *http.Response, field string) string {
		t.Helper()
		var v map[string]any
		if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		id, _ := v[field].(string)
		if id == "" {
			t.Fatalf("missing %q in response %v", field, v)
		}
		return id
	}

	// --- seed brand ---
	bResp := authedPost("/api/brands", `{"name":"Test Co","fromEmail":"n@test.co"}`)
	if bResp.StatusCode != http.StatusCreated {
		t.Fatalf("create brand: status=%d", bResp.StatusCode)
	}
	brandID := decodeID(t, bResp, "id")

	// --- seed list ---
	lResp := authedPost("/api/lists", fmt.Sprintf(`{"brandId":%q,"name":"Main"}`, brandID))
	if lResp.StatusCode != http.StatusCreated {
		t.Fatalf("create list: status=%d", lResp.StatusCode)
	}
	listID := decodeID(t, lResp, "id")

	// --- seed campaign ---
	cResp := authedPost("/api/campaigns", fmt.Sprintf(`{"brandId":%q,"subject":"Test Campaign","htmlBody":"<p>Hi</p>","plainBody":"Hi"}`, brandID))
	if cResp.StatusCode != http.StatusCreated {
		t.Fatalf("create campaign: status=%d", cResp.StatusCode)
	}
	campaignID := decodeID(t, cResp, "id")

	// --- seed 2 active subscribers ---
	for i := 0; i < 2; i++ {
		sResp := authedPost(fmt.Sprintf("/api/lists/%s/subscribers", listID),
			fmt.Sprintf(`{"email":"sub%d@example.com","status":"active"}`, i))
		if sResp.StatusCode != http.StatusCreated {
			t.Fatalf("add subscriber %d: status=%d", i, sResp.StatusCode)
		}
	}

	// --- POST /api/ab-tests → 200 with id and status "draft" ---
	atResp := authedPost("/api/ab-tests", fmt.Sprintf(
		`{"campaignId":%q,"listId":%q,"subjectA":"Hello A","subjectB":"Hello B","testPercent":30}`,
		campaignID, listID,
	))
	if atResp.StatusCode != http.StatusOK {
		t.Fatalf("create ab-test: status=%d, want 200", atResp.StatusCode)
	}
	var created map[string]any
	if err := json.NewDecoder(atResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created ab-test: %v", err)
	}
	testID, _ := created["id"].(string)
	if testID == "" {
		t.Fatalf("missing id in created ab-test: %v", created)
	}
	if created["status"] != "draft" {
		t.Errorf("status = %v, want draft", created["status"])
	}

	// --- GET /api/ab-tests → includes the created test ---
	listResp := authedGet("/api/ab-tests")
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list ab-tests: status=%d", listResp.StatusCode)
	}
	var listed []map[string]any
	if err := json.NewDecoder(listResp.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	found := false
	for _, item := range listed {
		if item["id"] == testID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("created test %s not found in list %v", testID, listed)
	}

	// --- POST /api/ab-tests/{id}/start → 204 ---
	startResp := authedPost(fmt.Sprintf("/api/ab-tests/%s/start", testID), "")
	if startResp.StatusCode != http.StatusNoContent {
		t.Fatalf("start ab-test: status=%d, want 204", startResp.StatusCode)
	}

	// --- GET /api/ab-tests/{id}/results → 200 with two variants ---
	resResp := authedGet(fmt.Sprintf("/api/ab-tests/%s/results", testID))
	if resResp.StatusCode != http.StatusOK {
		t.Fatalf("results ab-test: status=%d, want 200", resResp.StatusCode)
	}
	var results map[string]any
	if err := json.NewDecoder(resResp.Body).Decode(&results); err != nil {
		t.Fatalf("decode results: %v", err)
	}
	variants, _ := results["variants"].([]any)
	if len(variants) != 2 {
		t.Errorf("variants count = %d, want 2; results: %v", len(variants), results)
	}
}
