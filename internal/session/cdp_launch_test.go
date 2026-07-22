package session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCDPPortFromURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want int
		err  bool
	}{
		{"http://127.0.0.1:9222", 9222, false},
		{"http://localhost:9333", 9333, false},
		{"http://127.0.0.1", 9222, false},
		{"", 0, true},
		{"not-a-url", 0, true},
		{"http://127.0.0.1:0", 0, true},
	}
	for _, tc := range cases {
		got, err := cdpPortFromURL(tc.in)
		if tc.err {
			if err == nil {
				t.Fatalf("%q: expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %d want %d", tc.in, got, tc.want)
		}
	}
}

func TestWaitCDPReady(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/version" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Browser":"test"}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := waitCDPReady(ctx, srv.URL, 2*time.Second); err != nil {
		t.Fatalf("waitCDPReady: %v", err)
	}
}

func TestWaitCDPReadyTimeout(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := waitCDPReady(ctx, srv.URL, 500*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
