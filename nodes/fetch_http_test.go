package nodes_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	gen "axiom-official/generic-echo/gen"
	"axiom-official/generic-echo/nodes"
)

// TestFetchHTTP_GETReturnsStatusAndBody drives FetchHTTP against a local
// httptest.Server (hermetic — no real network) and asserts BOTH generalized
// output fields are populated correctly: the bug class this kills is either
// field being dropped or the wrong one being read (e.g. status_code always
// zero, or body silently truncated to "").
func TestFetchHTTP_GETReturnsStatusAndBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusTeapot)
		fmt.Fprint(w, "hello from the test server")
	}))
	defer srv.Close()

	ctx := context.Background()
	ax := newTestContext(t)
	input := &gen.HTTPRequest{Url: srv.URL}

	got, err := nodes.FetchHTTP(ctx, ax, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.StatusCode != http.StatusTeapot {
		t.Errorf("StatusCode = %d, want %d", got.StatusCode, http.StatusTeapot)
	}
	if got.Body != "hello from the test server" {
		t.Errorf("Body = %q, want %q", got.Body, "hello from the test server")
	}
}

// TestFetchHTTP_PostsRequestBody proves the request-side generalized field
// (Body) actually reaches the outbound call, not just the response side.
func TestFetchHTTP_PostsRequestBody(t *testing.T) {
	var gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		buf := make([]byte, 64)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()
	ax := newTestContext(t)
	input := &gen.HTTPRequest{Url: srv.URL, Method: http.MethodPost, Body: "request payload"}

	if _, err := nodes.FetchHTTP(ctx, ax, input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("server saw method %q, want POST", gotMethod)
	}
	if gotBody != "request payload" {
		t.Errorf("server saw body %q, want %q", gotBody, "request payload")
	}
}

// TestFetchHTTP_DefaultsToGET proves an empty Method field defaults to GET
// rather than producing a broken/empty-method request.
func TestFetchHTTP_DefaultsToGET(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()
	ax := newTestContext(t)
	input := &gen.HTTPRequest{Url: srv.URL}

	if _, err := nodes.FetchHTTP(ctx, ax, input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("server saw method %q, want GET (the empty-Method default)", gotMethod)
	}
}
