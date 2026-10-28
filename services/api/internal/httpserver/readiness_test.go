package httpserver

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestReadiness(t *testing.T) {
	for _, tt := range []struct {
		name  string
		ready func(context.Context) error
		want  int
	}{
		{"missing dependency", nil, http.StatusServiceUnavailable},
		{"available", func(context.Context) error { return nil }, http.StatusOK},
		{"failed", func(context.Context) error { return errors.New("password=must-not-leak") }, http.StatusServiceUnavailable},
		{"deadline", func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }, http.StatusServiceUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := New("unused", Options{Ready: tt.ready, ReadyTimeout: 10 * time.Millisecond})
			response := httptest.NewRecorder()
			start := time.Now()
			srv.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if response.Code != tt.want {
				t.Fatalf("status %d, want %d", response.Code, tt.want)
			}
			if strings.Contains(response.Body.String(), "must-not-leak") {
				t.Fatal("readiness leaked driver error")
			}
			if time.Since(start) > time.Second {
				t.Fatal("readiness exceeded its deadline")
			}
			live := httptest.NewRecorder()
			srv.Handler.ServeHTTP(live, httptest.NewRequest(http.MethodGet, "/healthz", nil))
			if live.Code != http.StatusOK {
				t.Fatal("dependency failure changed process liveness")
			}
		})
	}
}

func TestMetricsAreBoundedAndConcurrent(t *testing.T) {
	srv := New("unused", Options{DatabaseStats: func() sql.DBStats { return sql.DBStats{OpenConnections: 7, InUse: 2, Idle: 5} }})
	var workers sync.WaitGroup
	for i := 0; i < 32; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			response := httptest.NewRecorder()
			srv.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/untrusted-user-secret", nil))
		}()
	}
	workers.Wait()
	response := httptest.NewRecorder()
	srv.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body, _ := io.ReadAll(response.Result().Body)
	for _, expected := range []string{"connect_http_requests_total 33\n", "connect_db_open_connections 7\n", "connect_db_in_use_connections 2\n"} {
		if !strings.Contains(string(body), expected) {
			t.Fatalf("missing metric %q", expected)
		}
	}
	if strings.Contains(string(body), "untrusted-user-secret") {
		t.Fatal("metrics included an unbounded request path")
	}
}
