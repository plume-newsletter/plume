package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/campaign"
	"github.com/plume-newsletter/plume/internal/httpapi"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/template"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestTemplatesRoundTrip(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	cookie := auth.NewCookie([]byte("a-32-byte-or-longer-test-secret!!"))
	_, session := testsupport.SeedAdmin(t, pool, cookie, "templates@plume.test", "pw-12345678")

	q := gen.New(pool)
	router := httpapi.NewRouter(httpapi.AuthDeps{
		Queries:   q,
		Cookie:    cookie,
		Brands:    brand.New(q),
		Campaigns: campaign.New(q),
		Templates: template.New(q, campaign.New(q)),
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	authedGet := func(path string) *http.Response {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		req.AddCookie(session)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		return resp
	}
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
	authedDelete := func(path string) *http.Response {
		req, _ := http.NewRequest(http.MethodDelete, srv.URL+path, nil)
		req.AddCookie(session)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("DELETE %s: %v", path, err)
		}
		return resp
	}
	_ = authedDelete // available for future use
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

	// 1. GET /api/templates → 200, 3 prebuilt starters.
	listResp1 := authedGet("/api/templates")
	if listResp1.StatusCode != http.StatusOK {
		t.Fatalf("list templates (1): status=%d, want 200", listResp1.StatusCode)
	}
	var list1 []map[string]any
	if err := json.NewDecoder(listResp1.Body).Decode(&list1); err != nil {
		t.Fatalf("decode list1: %v", err)
	}
	if len(list1) != 3 {
		t.Fatalf("template count = %d, want 3 prebuilt starters", len(list1))
	}
	starterID, _ := list1[0]["id"].(string)
	if starterID == "" {
		t.Fatalf("no id in first template: %v", list1[0])
	}

	// 2. POST /api/templates → 200, prebuilt==false.
	createResp := authedPost("/api/templates", `{"name":"Mine","category":"Promo","bodyJson":[{"id":"x","type":"text","html":"hi"}]}`)
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("create template: status=%d, want 200", createResp.StatusCode)
	}
	var created map[string]any
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created template: %v", err)
	}
	if prebuilt, _ := created["prebuilt"].(bool); prebuilt {
		t.Errorf("created template prebuilt=%v, want false", prebuilt)
	}

	// 3. GET /api/templates → 200, now 4.
	listResp2 := authedGet("/api/templates")
	if listResp2.StatusCode != http.StatusOK {
		t.Fatalf("list templates (2): status=%d, want 200", listResp2.StatusCode)
	}
	var list2 []map[string]any
	if err := json.NewDecoder(listResp2.Body).Decode(&list2); err != nil {
		t.Fatalf("decode list2: %v", err)
	}
	if len(list2) != 4 {
		t.Fatalf("template count = %d, want 4", len(list2))
	}

	// 4. POST /api/templates/{starterID}/use → 200, decode campaignId.
	useResp := authedPost(
		fmt.Sprintf("/api/templates/%s/use", starterID),
		fmt.Sprintf(`{"brandId":%q,"subject":"Hi"}`, brandID),
	)
	if useResp.StatusCode != http.StatusOK {
		t.Fatalf("use template: status=%d, want 200", useResp.StatusCode)
	}
	campaignID := decodeID(t, useResp, "campaignId")

	// 5. GET /api/campaigns/{campaignId} → 200, status "draft", body_json non-empty.
	campResp := authedGet(fmt.Sprintf("/api/campaigns/%s", campaignID))
	if campResp.StatusCode != http.StatusOK {
		t.Fatalf("get campaign: status=%d, want 200", campResp.StatusCode)
	}
	var camp map[string]any
	if err := json.NewDecoder(campResp.Body).Decode(&camp); err != nil {
		t.Fatalf("decode campaign: %v", err)
	}
	if camp["status"] != "draft" {
		t.Errorf("campaign status = %v, want draft", camp["status"])
	}
	bodyJSON, _ := camp["body_json"].(string)
	if bodyJSON == "" || bodyJSON == "[]" {
		t.Errorf("campaign body_json = %q, want non-empty blocks", bodyJSON)
	}
}
