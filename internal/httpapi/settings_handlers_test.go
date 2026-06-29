package httpapi_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/crypto"
	"github.com/plume-newsletter/plume/internal/httpapi"
	"github.com/plume-newsletter/plume/internal/settings"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestSettingsEndpointsRequireAuthAndNeverExposeSecret(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	cookie := auth.NewCookie([]byte("a-32-byte-or-longer-test-secret!!"))
	_, session := testsupport.SeedAdmin(t, pool, cookie, "a@plume.test", "pw-12345678")

	cipher, err := crypto.New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	q := gen.New(pool)

	router := httpapi.NewRouter(httpapi.AuthDeps{
		Queries: q, Cookie: cookie, Settings: settings.New(q, cipher),
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	// Unauthenticated → 401.
	resp, err := http.Get(srv.URL + "/api/settings")
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no auth: status=%v err=%v", resp, err)
	}

	// Authenticated GET before configuring → 200, sesConfigured:false.
	greq, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/settings", nil)
	greq.AddCookie(session)
	gresp, err := http.DefaultClient.Do(greq)
	if err != nil || gresp.StatusCode != http.StatusOK {
		t.Fatalf("get before configure: status=%v err=%v", gresp, err)
	}
	body := new(bytes.Buffer)
	_, _ = body.ReadFrom(gresp.Body)
	if !bytes.Contains(body.Bytes(), []byte(`"sesConfigured":false`)) {
		t.Fatalf("expected sesConfigured:false, got %s", body.String())
	}

	// PUT credentials → 204.
	preq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/settings/ses",
		bytes.NewBufferString(`{"accessKeyId":"AKIATEST","secretAccessKey":"super-secret","region":"us-east-1"}`))
	preq.AddCookie(session)
	preq.Header.Set("Content-Type", "application/json")
	presp, err := http.DefaultClient.Do(preq)
	if err != nil || presp.StatusCode != http.StatusNoContent {
		t.Fatalf("put ses: status=%v err=%v", presp, err)
	}

	// Authenticated GET after configuring → 200, sesConfigured:true, region set, no secret leaked.
	greq2, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/settings", nil)
	greq2.AddCookie(session)
	gresp2, err := http.DefaultClient.Do(greq2)
	if err != nil || gresp2.StatusCode != http.StatusOK {
		t.Fatalf("get after configure: status=%v err=%v", gresp2, err)
	}
	body2 := new(bytes.Buffer)
	_, _ = body2.ReadFrom(gresp2.Body)
	if !bytes.Contains(body2.Bytes(), []byte(`"sesConfigured":true`)) {
		t.Fatalf("expected sesConfigured:true, got %s", body2.String())
	}
	if !bytes.Contains(body2.Bytes(), []byte(`"sesRegion":"us-east-1"`)) {
		t.Fatalf("expected sesRegion us-east-1, got %s", body2.String())
	}
	if bytes.Contains(body2.Bytes(), []byte("super-secret")) {
		t.Fatalf("response leaked the secret: %s", body2.String())
	}
}
