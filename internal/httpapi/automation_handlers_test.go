package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/automation"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/httpapi"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestAutomationEndpoints(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	cookie := auth.NewCookie([]byte("a-32-byte-or-longer-test-secret!!"))
	adminID, session := testsupport.SeedAdmin(t, pool, cookie, "auto@plume.test", "pw-12345678")

	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()

	// Seed brand + list.
	b, err := brand.New(q).Create(ctx, adminID, brand.BrandInput{Name: "Acme", FromName: "A", FromEmail: "a@a.test", ReplyTo: ""})
	if err != nil {
		t.Fatalf("seed brand: %v", err)
	}
	l, err := list.New(q).Create(ctx, adminID, b.ID, "Main")
	if err != nil {
		t.Fatalf("seed list: %v", err)
	}

	autoSvc := automation.New(pool, q, h)

	router := httpapi.NewRouter(httpapi.AuthDeps{
		Queries:     q,
		Cookie:      cookie,
		Brands:      brand.New(q),
		Lists:       list.New(q),
		Automations: autoSvc,
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	authedDo := func(method, path, body string) *http.Response {
		var reqBody *strings.Reader
		if body != "" {
			reqBody = strings.NewReader(body)
		} else {
			reqBody = strings.NewReader("")
		}
		req, _ := http.NewRequest(method, srv.URL+path, reqBody)
		req.AddCookie(session)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		return resp
	}

	// 1) POST /api/automations → 200, status "draft".
	createBody := `{"name":"Welcome","listId":"` + l.ID.String() + `"}`
	resp := authedDo(http.MethodPost, "/api/automations", createBody)
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("create: status=%d, want 200; body: %s", resp.StatusCode, buf.String())
	}
	var created automation.Automation
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.Status != "draft" {
		t.Errorf("status=%q, want draft", created.Status)
	}
	if created.ID == "" {
		t.Fatal("created.ID is empty")
	}

	// 2) GET /api/automations includes it.
	listResp := authedDo(http.MethodGet, "/api/automations", "")
	listBuf := new(bytes.Buffer)
	_, _ = listBuf.ReadFrom(listResp.Body)
	if !bytes.Contains(listBuf.Bytes(), []byte("Welcome")) {
		t.Fatalf("list missing automation: %s", listBuf.String())
	}

	// 3) PUT /api/automations/{id}/steps → 204.
	stepsBody := `{"steps":[{"kind":"send","subject":"Hi","html":"<p>h</p>","waitDays":0}]}`
	stepsResp := authedDo(http.MethodPut, "/api/automations/"+created.ID+"/steps", stepsBody)
	if stepsResp.StatusCode != http.StatusNoContent {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(stepsResp.Body)
		t.Fatalf("steps: status=%d, want 204; body: %s", stepsResp.StatusCode, buf.String())
	}

	// 4) POST /api/automations/{id}/status {status:"live"} → 204.
	statusResp := authedDo(http.MethodPost, "/api/automations/"+created.ID+"/status", `{"status":"live"}`)
	if statusResp.StatusCode != http.StatusNoContent {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(statusResp.Body)
		t.Fatalf("set status: status=%d, want 204; body: %s", statusResp.StatusCode, buf.String())
	}

	// 5) GET /api/automations/{id} shows step + status "live".
	getResp := authedDo(http.MethodGet, "/api/automations/"+created.ID, "")
	var got automation.Automation
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if got.Status != "live" {
		t.Errorf("get status=%q, want live", got.Status)
	}
	if len(got.Steps) != 1 || got.Steps[0].Subject != "Hi" {
		t.Errorf("steps=%+v, want 1 step with subject Hi", got.Steps)
	}
}
