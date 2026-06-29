package httpapi_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/httpapi"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestBrandEndpointsRequireAuthAndPersist(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	cookie := auth.NewCookie([]byte("a-32-byte-or-longer-test-secret!!"))
	_, session := testsupport.SeedAdmin(t, pool, cookie, "a@plume.test", "pw-12345678")

	router := httpapi.NewRouter(httpapi.AuthDeps{
		Queries: gen.New(pool), Cookie: cookie, Brands: brand.New(gen.New(pool)),
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	// Unauthenticated → 401.
	resp, _ := http.Get(srv.URL + "/api/brands")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no auth: status = %d, want 401", resp.StatusCode)
	}

	// Authenticated create → 201.
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/brands",
		bytes.NewBufferString(`{"name":"Acme","fromEmail":"n@acme.test"}`))
	req.AddCookie(session)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: status=%v err=%v", resp.StatusCode, err)
	}

	// List shows the created brand.
	lreq, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/brands", nil)
	lreq.AddCookie(session)
	lresp, _ := http.DefaultClient.Do(lreq)
	body := new(bytes.Buffer)
	_, _ = body.ReadFrom(lresp.Body)
	if !bytes.Contains(body.Bytes(), []byte("Acme")) {
		t.Fatalf("list missing brand: %s", body.String())
	}
}
