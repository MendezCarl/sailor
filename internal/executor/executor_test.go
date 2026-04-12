package executor

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MendezCarl/sailor.git/internal/request"
)

func makeReq(method, url string) *request.Request {
	return &request.Request{Method: method, URL: url}
}

func TestSend_FollowRedirects_Default(t *testing.T) {
	// Server returns 302 → /dest, /dest returns 200.
	mux := http.NewServeMux()
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dest", http.StatusFound)
	})
	mux.HandleFunc("/dest", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := Send(makeReq("GET", srv.URL+"/redirect"), 0, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
}

func TestSend_NoFollowRedirects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dest", http.StatusFound)
	}))
	defer srv.Close()

	falseVal := false
	resp, err := Send(makeReq("GET", srv.URL+"/redirect"), 0, Options{FollowRedirects: &falseVal})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusFound {
		t.Errorf("status: got %d, want 302", resp.StatusCode)
	}
}

func TestSend_InsecureTLS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Without --insecure: should fail TLS verification.
	_, err := Send(makeReq("GET", srv.URL), 0, Options{Insecure: false})
	if err == nil {
		t.Error("expected TLS error without --insecure, got none")
	}

	// With --insecure: should succeed.
	resp, err := Send(makeReq("GET", srv.URL), 0, Options{Insecure: true})
	if err != nil {
		t.Fatalf("unexpected error with --insecure: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
}

func TestSend_TimeoutAndInsecureCompose(t *testing.T) {
	// Both timeout and Insecure can be set simultaneously without one canceling the other.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := Send(makeReq("GET", srv.URL), 5*time.Second, Options{Insecure: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
}
