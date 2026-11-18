package accounts

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUsername(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{{"Alice_7", "alice_7"}, {"ab", ""}, {" alice", ""}, {"a-b", ""}, {"1alice", ""}, {"álIce", ""}, {strings.Repeat("a", 33), ""}} {
		got, err := Username(tc.raw)
		if got != tc.want || (err != nil) != (tc.want == "") {
			t.Errorf("unexpected username validation for %q", tc.raw)
		}
	}
}
func TestRandomValueAndDigest(t *testing.T) {
	a, b := RandomValue(), RandomValue()
	if a == b {
		t.Fatal("repeated random value")
	}
	bytes, err := DecodeValue(a)
	if err != nil || len(bytes) != 32 {
		t.Fatal("invalid generated value")
	}
	for _, bad := range []string{"", a + "=", strings.Repeat("_", 43), strings.Repeat("x", 44)} {
		if _, err := DecodeValue(bad); err == nil {
			t.Fatal("accepted noncanonical value")
		}
	}
	digest, err := Digest(a)
	if err != nil || len(digest) != 32 || string(digest) == string(bytes) {
		t.Fatal("invalid digest")
	}
}
func TestStrictJSON(t *testing.T) {
	for _, body := range []string{
		`{"username":"alice","username":"bob"}`, `{"Username":"alice"}`, `{"username":"alice","extra":true}`,
		`{"username":null}`, `{"username":"alice"} {}`, `[]`, `null`, `{"username":123}`, strings.Repeat(" ", 32769),
		`{"username":{"nested":1,"nested":2}}`,
	} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		var value struct {
			Username string `json:"username"`
		}
		if decode(httptest.NewRecorder(), r, &value, "username") == nil {
			t.Fatal("accepted invalid JSON envelope")
		}
	}
}
func TestHTTPSAndBearerBoundaries(t *testing.T) {
	service, err := New(nil, "connect.test", []string{"https://connect.test"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	service.RegisterRoutes(mux)
	for _, tc := range []struct {
		path, auth string
		tls        bool
		want       int
	}{
		{"/v1/me", "", false, 400}, {"/v1/me", "", true, 401}, {"/v1/me", "Bearer invalid", true, 401},
		{"/v1/me?token=never-accepted", "", true, 400},
	} {
		r := httptest.NewRequest("GET", tc.path, nil)
		if tc.tls {
			r.TLS = &tls.ConnectionState{}
		}
		r.Header.Set("Authorization", tc.auth)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != tc.want || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("incorrect authentication boundary")
		}
	}
}
func TestSourceLimiterAndEviction(t *testing.T) {
	l := limiter{}
	now := time.Now()
	for i := 0; i < 5; i++ {
		if !l.allow("127.0.0.1", 0, now) {
			t.Fatal("early limit")
		}
	}
	if l.allow("127.0.0.1", 0, now) {
		t.Fatal("registration limit ignored")
	}
	if !l.allow("127.0.0.1", 1, now) {
		t.Fatal("unrelated bucket exhausted")
	}
	if !l.allow("127.0.0.1", 0, now.Add(time.Minute)) {
		t.Fatal("window not reset")
	}
	l.sources = make(map[string]*source)
	for i := 0; i < 10000; i++ {
		l.sources[string(rune(i))] = &source{seen: now}
	}
	if l.allow("new source", 1, now) {
		t.Fatal("source capacity unbounded")
	}
	if !l.allow("new source", 1, now.Add(11*time.Minute)) {
		t.Fatal("idle entries not evicted")
	}
}
func TestVerificationConcurrencyBound(t *testing.T) {
	a := httpAPI{verify: make(chan struct{}, 8)}
	for i := 0; i < 8; i++ {
		a.verify <- struct{}{}
	}
	r := httptest.NewRequest("POST", "https://connect.test/v1/authentication/finish", strings.NewReader(`{}`))
	r.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != 429 {
		t.Fatal("excess verification work was admitted")
	}
	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] != "rate_limited" {
		t.Fatal("unexpected concurrency response")
	}
}

func TestDependencyErrorsAreRedacted(t *testing.T) {
	w := httptest.NewRecorder()
	failure(w, errors.New("postgres://private:password@internal/database"))
	if w.Code != 503 || strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), "internal") {
		t.Fatal("dependency error disclosed")
	}
}
func TestForwardedSourceIsNotTrusted(t *testing.T) {
	service, _ := New(nil, "connect.test", []string{"https://connect.test"}, nil)
	api := &httpAPI{service: service, verify: make(chan struct{}, 8)}
	for i := 0; i < 5; i++ {
		if !api.limiter.allow("127.0.0.1", 0, time.Now()) {
			t.Fatal("unexpected limiter admission")
		}
	}
	r := httptest.NewRequest("POST", "https://connect.test/v1/accounts/registration/options", strings.NewReader(`{"username":"alice"}`))
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "192.0.2.1")
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	api.ServeHTTP(w, r)
	if w.Code != 429 {
		t.Fatal("forwarded header bypassed source limit")
	}
}
