package httpserver

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"connect.local/api/internal/observability"
)

type Options struct {
	Ready         func(context.Context) error
	ReadyTimeout  time.Duration
	DatabaseStats func() sql.DBStats
	Metrics       *observability.Metrics
}

// New separates liveness from bounded, fail-closed dependency readiness.
func New(addr string, options Options) *http.Server {
	metrics := options.Metrics
	if metrics == nil {
		metrics = &observability.Metrics{}
	}
	if options.ReadyTimeout <= 0 {
		options.ReadyTimeout = time.Second
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte("{\"status\":\"ok\"}\n"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		metrics.ReadinessChecks.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		ctx, cancel := context.WithTimeout(r.Context(), options.ReadyTimeout)
		defer cancel()
		if options.Ready == nil || options.Ready(ctx) != nil {
			metrics.ReadinessFailures.Add(1)
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("{\"status\":\"unavailable\"}\n"))
			return
		}
		_, _ = w.Write([]byte("{\"status\":\"ready\"}\n"))
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		var stats sql.DBStats
		if options.DatabaseStats != nil {
			stats = options.DatabaseStats()
		}
		metrics.Write(w, stats)
	})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		metrics.Requests.Add(1)
		defer func() { metrics.ObserveRequest(time.Since(start)) }()
		mux.ServeHTTP(w, r)
	})
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
}
