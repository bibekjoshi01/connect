package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthContract(t *testing.T) {
	srv := httptest.NewServer(New("unused", Options{}).Handler)
	defer srv.Close()
	client := srv.Client()
	for _, tt := range []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/healthz", http.StatusOK},
		{http.MethodPost, "/healthz", http.StatusMethodNotAllowed},
		{http.MethodGet, "/unknown", http.StatusNotFound},
		{http.MethodGet, "/healthz/extra", http.StatusNotFound},
	} {
		t.Run(tt.method+tt.path, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, srv.URL+tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tt.status {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.status)
			}
			if tt.status == http.StatusOK {
				var body struct {
					Status string `json:"status"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body.Status != "ok" {
					t.Fatalf("unexpected body: %+v", body)
				}
				if resp.Header.Get("Content-Type") != "application/json" {
					t.Fatal("missing JSON content type")
				}
				if resp.Header.Get("Cache-Control") != "no-store" {
					t.Fatal("liveness must not be cached")
				}
			}
		})
	}
}
