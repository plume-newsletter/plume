package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/email"
	"github.com/plume-newsletter/plume/internal/email/logprovider"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/httpapi"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/signup"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func seedSignupList(t *testing.T, ctx context.Context, q *gen.Queries) gen.List {
	t.Helper()
	owner := uuid.New()
	b, err := brand.New(q).Create(ctx, owner, brand.BrandInput{
		Name: "Acme", FromName: "Acme News", FromEmail: "n@acme.test", ReplyTo: "r@acme.test",
	})
	if err != nil {
		t.Fatalf("seed brand: %v", err)
	}
	l, err := list.New(q).Create(ctx, owner, b.ID, "Main")
	if err != nil {
		t.Fatalf("seed list: %v", err)
	}
	return l
}

func TestSubscribeHandlerFormAlwaysReturns200(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	h := hooks.New()
	ctx := context.Background()
	provider := logprovider.New(io.Discard)

	l := seedSignupList(t, ctx, q)

	router := httpapi.NewRouter(httpapi.AuthDeps{Signup: signup.New(q, h, email.NewStaticResolver(provider), "https://send.example.test")})
	srv := httptest.NewServer(router)
	defer srv.Close()

	form := url.Values{"email": {"reader@test.com"}, "name": {"Reader"}}
	resp, err := http.Post(srv.URL+"/subscribe/"+l.ID.String(), "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("POST /subscribe/{listId}: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := new(bytes.Buffer)
	_, _ = body.ReadFrom(resp.Body)
	if !strings.Contains(body.String(), "Check your email") {
		t.Fatalf("body = %q, want generic confirmation message", body.String())
	}

	sub, err := q.GetSubscriberInListByEmail(ctx, gen.GetSubscriberInListByEmailParams{ListID: l.ID, Email: "reader@test.com"})
	if err != nil {
		t.Fatalf("GetSubscriberInListByEmail: %v", err)
	}
	if sub.Status != "pending" {
		t.Fatalf("status = %q, want pending", sub.Status)
	}
}

func TestSubscribeHandlerJSONBody(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	h := hooks.New()
	ctx := context.Background()
	provider := logprovider.New(io.Discard)

	l := seedSignupList(t, ctx, q)

	router := httpapi.NewRouter(httpapi.AuthDeps{Signup: signup.New(q, h, email.NewStaticResolver(provider), "https://send.example.test")})
	srv := httptest.NewServer(router)
	defer srv.Close()

	payload, _ := json.Marshal(map[string]string{"email": "jsonreader@test.com", "name": "JSON Reader"})
	resp, err := http.Post(srv.URL+"/subscribe/"+l.ID.String(), "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST /subscribe/{listId} json: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	sub, err := q.GetSubscriberInListByEmail(ctx, gen.GetSubscriberInListByEmailParams{ListID: l.ID, Email: "jsonreader@test.com"})
	if err != nil {
		t.Fatalf("GetSubscriberInListByEmail: %v", err)
	}
	if sub.Status != "pending" {
		t.Fatalf("status = %q, want pending", sub.Status)
	}
}

func TestSubscribeHandlerMissingEmailIsBadRequest(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	h := hooks.New()
	ctx := context.Background()
	provider := logprovider.New(io.Discard)

	l := seedSignupList(t, ctx, q)

	router := httpapi.NewRouter(httpapi.AuthDeps{Signup: signup.New(q, h, email.NewStaticResolver(provider), "https://send.example.test")})
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/subscribe/"+l.ID.String(), "application/x-www-form-urlencoded", strings.NewReader(""))
	if err != nil {
		t.Fatalf("POST /subscribe/{listId}: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestSubscribeHandlerUnknownListIs404(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	h := hooks.New()
	provider := logprovider.New(io.Discard)

	router := httpapi.NewRouter(httpapi.AuthDeps{Signup: signup.New(q, h, email.NewStaticResolver(provider), "https://send.example.test")})
	srv := httptest.NewServer(router)
	defer srv.Close()

	form := url.Values{"email": {"ghost@test.com"}}
	resp, err := http.Post(srv.URL+"/subscribe/"+uuid.New().String(), "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("POST /subscribe/{listId}: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestConfirmHandlerActivatesAndAlwaysReturns200(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	h := hooks.New()
	ctx := context.Background()
	provider := logprovider.New(io.Discard)

	l := seedSignupList(t, ctx, q)
	svc := signup.New(q, h, email.NewStaticResolver(provider), "https://send.example.test")
	if err := svc.Subscribe(ctx, l.ID, "confirmable@test.com", "C"); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	sub, err := q.GetSubscriberInListByEmail(ctx, gen.GetSubscriberInListByEmailParams{ListID: l.ID, Email: "confirmable@test.com"})
	if err != nil {
		t.Fatalf("GetSubscriberInListByEmail: %v", err)
	}

	router := httpapi.NewRouter(httpapi.AuthDeps{Signup: svc})
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/confirm/" + sub.ID.String())
	if err != nil {
		t.Fatalf("GET /confirm/{subscriberId}: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	body := new(bytes.Buffer)
	_, _ = body.ReadFrom(resp.Body)
	if !strings.Contains(body.String(), "confirmed") {
		t.Fatalf("body = %q, want confirmation text", body.String())
	}

	updated, err := q.GetSubscriberByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetSubscriberByID: %v", err)
	}
	if updated.Status != "active" {
		t.Fatalf("status = %q, want active", updated.Status)
	}

	// Bad uuid must still return 200 + generic page (leak-safe).
	resp2, err := http.Get(srv.URL + "/confirm/not-a-uuid")
	if err != nil {
		t.Fatalf("GET /confirm/{bad}: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("bad id status = %d, want 200", resp2.StatusCode)
	}
}
