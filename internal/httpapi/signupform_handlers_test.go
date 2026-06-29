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
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/httpapi"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/signupform"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestSignupFormEndpoints(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	cookie := auth.NewCookie([]byte("a-32-byte-or-longer-test-secret!!"))
	adminID, session := testsupport.SeedAdmin(t, pool, cookie, "sf@plume.test", "pw-12345678")

	q := gen.New(pool)
	ctx := context.Background()

	// Seed brand + list to attach forms to.
	b, err := brand.New(q).Create(ctx, adminID, brand.BrandInput{Name: "Acme", FromName: "A", FromEmail: "a@a.test", ReplyTo: ""})
	if err != nil {
		t.Fatalf("seed brand: %v", err)
	}
	l, err := list.New(q).Create(ctx, adminID, b.ID, "Main")
	if err != nil {
		t.Fatalf("seed list: %v", err)
	}

	router := httpapi.NewRouter(httpapi.AuthDeps{
		Queries:     q,
		Cookie:      cookie,
		Brands:      brand.New(q),
		Lists:       list.New(q),
		SignupForms: signupform.New(q),
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	authedPost := func(path, body string) *http.Response {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+path, strings.NewReader(body))
		req.AddCookie(session)
		req.Header.Set("Content-Type", "application/json")
		resp, reqErr := http.DefaultClient.Do(req)
		if reqErr != nil {
			t.Fatalf("authedPost %s: %v", path, reqErr)
		}
		return resp
	}
	authedGet := func(path string) *http.Response {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		req.AddCookie(session)
		resp, reqErr := http.DefaultClient.Do(req)
		if reqErr != nil {
			t.Fatalf("authedGet %s: %v", path, reqErr)
		}
		return resp
	}

	// 1) Authed create returns a form id; list returns it.
	createBody := `{"listId":"` + l.ID.String() + `","name":"Hero","heading":"Join us","description":"Weekly","buttonText":""}`
	resp := authedPost("/api/signup-forms", createBody)
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("create status = %d, want 200; body: %s", resp.StatusCode, buf.String())
	}
	var created signupform.Form
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created form: %v", err)
	}
	if created.ID == "" {
		t.Fatal("create returned empty id")
	}

	lresp := authedGet("/api/signup-forms")
	lbuf := new(bytes.Buffer)
	_, _ = lbuf.ReadFrom(lresp.Body)
	if !bytes.Contains(lbuf.Bytes(), []byte("Hero")) {
		t.Fatalf("list missing form: %s", lbuf.String())
	}

	// 2) Public GET /f/{id} -> 200, Content-Type text/html, body contains the heading
	//    and action="/subscribe/<listId>".
	pubResp, pubErr := http.Get(srv.URL + "/f/" + created.ID) //nolint:noctx
	if pubErr != nil {
		t.Fatalf("public get: %v", pubErr)
	}
	if pubResp.StatusCode != http.StatusOK {
		t.Fatalf("public get status = %d, want 200", pubResp.StatusCode)
	}
	ct := pubResp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("content-type = %q, want text/html", ct)
	}
	pubBody := new(bytes.Buffer)
	_, _ = pubBody.ReadFrom(pubResp.Body)
	if !bytes.Contains(pubBody.Bytes(), []byte("Join us")) {
		t.Fatalf("public page missing heading: %s", pubBody.String())
	}
	if !bytes.Contains(pubBody.Bytes(), []byte(`action="/subscribe/`+l.ID.String())) {
		t.Fatalf("public page missing subscribe action: %s", pubBody.String())
	}

	// 3) Unknown /f/<zero-uuid> -> 404.
	notFoundResp, _ := http.Get(srv.URL + "/f/00000000-0000-0000-0000-000000000000") //nolint:noctx
	if notFoundResp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown form status = %d, want 404", notFoundResp.StatusCode)
	}

	// 4) XSS: create a form whose heading is "<script>alert(1)</script>",
	//    GET /f/{id} and assert body contains &lt;script&gt; and NOT raw <script>alert(1).
	xssBody := `{"listId":"` + l.ID.String() + `","name":"XSS Test","heading":"<script>alert(1)</script>","description":"d","buttonText":"Go"}`
	xssResp := authedPost("/api/signup-forms", xssBody)
	if xssResp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(xssResp.Body)
		t.Fatalf("xss create status = %d, want 200; body: %s", xssResp.StatusCode, buf.String())
	}
	var xssForm signupform.Form
	if err := json.NewDecoder(xssResp.Body).Decode(&xssForm); err != nil {
		t.Fatalf("decode xss form: %v", err)
	}
	xssLandingResp, _ := http.Get(srv.URL + "/f/" + xssForm.ID) //nolint:noctx
	xssLandingBody := new(bytes.Buffer)
	_, _ = xssLandingBody.ReadFrom(xssLandingResp.Body)
	if !bytes.Contains(xssLandingBody.Bytes(), []byte("&lt;script&gt;")) {
		t.Fatalf("XSS: expected &lt;script&gt; in page body; got: %s", xssLandingBody.String())
	}
	if bytes.Contains(xssLandingBody.Bytes(), []byte("<script>alert(1)")) {
		t.Fatalf("XSS: raw <script>alert(1) found in page body — XSS not escaped!")
	}
}
