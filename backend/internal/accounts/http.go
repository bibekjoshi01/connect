package accounts

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type source struct {
	counts [3]int
	window time.Time
	seen   time.Time
}
type limiter struct {
	mu      sync.Mutex
	sources map[string]*source
	swept   time.Time
}

func (l *limiter) allow(peer string, category int, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.sources == nil {
		l.sources = make(map[string]*source)
	}
	if now.Sub(l.swept) >= time.Minute {
		for key, value := range l.sources {
			if now.Sub(value.seen) >= 10*time.Minute {
				delete(l.sources, key)
			}
		}
		l.swept = now
	}
	entry := l.sources[peer]
	if entry == nil {
		if len(l.sources) >= 10000 {
			return false
		}
		entry = &source{window: now}
		l.sources[peer] = entry
	}
	entry.seen = now
	if now.Sub(entry.window) >= time.Minute {
		entry.window = now
		entry.counts = [3]int{}
	}
	limit := [3]int{5, 20, 20}[category]
	if entry.counts[category] >= limit {
		return false
	}
	entry.counts[category]++
	return true
}

type httpAPI struct {
	service *Service
	limiter limiter
	verify  chan struct{}
}

func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	api := &httpAPI{service: s, verify: make(chan struct{}, 8)}
	mux.Handle("/v1/", api)
}
func reply(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	if value != nil {
		_ = json.NewEncoder(w).Encode(value)
	}
}
func failure(w http.ResponseWriter, err error) {
	code := http.StatusServiceUnavailable
	public := ErrUnavailable
	switch {
	case errors.Is(err, ErrInvalid):
		code = http.StatusBadRequest
		public = ErrInvalid
	case errors.Is(err, ErrUnauthorized):
		code = http.StatusUnauthorized
		public = ErrUnauthorized
		w.Header().Set("WWW-Authenticate", "Bearer")
	case errors.Is(err, ErrConflict):
		code = http.StatusConflict
		public = ErrConflict
	case errors.Is(err, ErrLimited):
		code = http.StatusTooManyRequests
		public = ErrLimited
		w.Header().Set("Retry-After", "60")
	}
	reply(w, code, map[string]string{"error": public.Error()})
}
func bearer(r *http.Request) (string, error) {
	values := r.Header.Values("Authorization")
	if len(values) != 1 {
		return "", ErrUnauthorized
	}
	parts := strings.Split(values[0], " ")
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", ErrUnauthorized
	}
	if _, err := DecodeValue(parts[1]); err != nil {
		return "", ErrUnauthorized
	}
	return parts[1], nil
}
func (a *httpAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.TLS == nil {
		reply(w, 400, map[string]string{"error": "https_required"})
		return
	}
	if r.URL.RawQuery != "" {
		failure(w, ErrInvalid)
		return
	}
	if r.URL.Path == "/v1/me" && r.Method == http.MethodGet {
		token, err := bearer(r)
		if err != nil {
			failure(w, err)
			return
		}
		profile, err := a.service.Profile(r.Context(), token)
		if err != nil {
			failure(w, err)
			return
		}
		reply(w, 200, profile)
		return
	}
	if r.URL.Path == "/v1/sessions/current" && r.Method == http.MethodDelete {
		token, err := bearer(r)
		if err == nil {
			err = a.service.Logout(r.Context(), token)
		}
		if err != nil {
			failure(w, err)
			return
		}
		reply(w, 204, nil)
		return
	}
	category := -1
	switch r.URL.Path {
	case "/v1/accounts/registration/options":
		category = 0
	case "/v1/authentication/options":
		category = 1
	case "/v1/accounts/registration/finish", "/v1/authentication/finish":
		category = 2
	case "/v1/me", "/v1/sessions/current":
		method := http.MethodGet
		if r.URL.Path == "/v1/sessions/current" {
			method = http.MethodDelete
		}
		w.Header().Set("Allow", method)
		reply(w, http.StatusMethodNotAllowed, map[string]string{"error": "invalid_request"})
		return
	default:
		reply(w, http.StatusNotFound, map[string]string{"error": "invalid_request"})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		reply(w, http.StatusMethodNotAllowed, map[string]string{"error": "invalid_request"})
		return
	}
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		failure(w, ErrInvalid)
		return
	}
	ip := net.ParseIP(peer)
	if ip == nil {
		failure(w, ErrInvalid)
		return
	}
	// Forwarded and X-Forwarded-For are deliberately ignored.
	if !a.limiter.allow(ip.String(), category, time.Now()) {
		failure(w, ErrLimited)
		return
	}
	if category == 2 {
		select {
		case a.verify <- struct{}{}:
			defer func() { <-a.verify }()
		default:
			failure(w, ErrLimited)
			return
		}
	}
	switch r.URL.Path {
	case "/v1/accounts/registration/options":
		var body struct {
			Username string `json:"username"`
		}
		if decode(w, r, &body, "username") != nil {
			failure(w, ErrInvalid)
			return
		}
		result, err := a.service.BeginRegistration(r.Context(), body.Username)
		if err != nil {
			failure(w, err)
			return
		}
		reply(w, 200, result)
	case "/v1/authentication/options":
		var body struct {
			Handle string `json:"auth_handle"`
		}
		if decode(w, r, &body, "auth_handle") != nil {
			failure(w, ErrInvalid)
			return
		}
		result, err := a.service.BeginAuthentication(r.Context(), body.Handle)
		if err != nil {
			failure(w, err)
			return
		}
		reply(w, 200, result)
	default:
		var body struct {
			ID         string          `json:"ceremony_id"`
			Credential json.RawMessage `json:"credential"`
		}
		if decode(w, r, &body, "ceremony_id", "credential") != nil || len(body.Credential) == 0 || body.Credential[0] != '{' {
			failure(w, ErrInvalid)
			return
		}
		var result Session
		if r.URL.Path == "/v1/accounts/registration/finish" {
			result, err = a.service.FinishRegistration(r.Context(), body.ID, body.Credential)
		} else {
			result, err = a.service.FinishAuthentication(r.Context(), body.ID, body.Credential)
		}
		if err != nil {
			failure(w, err)
			return
		}
		status := 200
		if r.URL.Path == "/v1/accounts/registration/finish" {
			status = 201
		}
		reply(w, status, result)
	}
}
func decode(w http.ResponseWriter, r *http.Request, target any, fields ...string) error {
	media, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return ErrInvalid
	}
	for key, value := range params {
		if key != "charset" || !strings.EqualFold(value, "utf-8") {
			return ErrInvalid
		}
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 32<<10))
	if err != nil {
		return ErrInvalid
	}
	tokens := json.NewDecoder(bytes.NewReader(raw))
	if uniqueJSON(tokens, 0) != nil {
		return ErrInvalid
	}
	if _, err = tokens.Token(); err != io.EOF {
		return ErrInvalid
	}
	var envelope map[string]json.RawMessage
	if json.Unmarshal(raw, &envelope) != nil || len(envelope) != len(fields) {
		return ErrInvalid
	}
	for _, field := range fields {
		value, ok := envelope[field]
		if !ok || bytes.Equal(value, []byte("null")) {
			return ErrInvalid
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		return ErrInvalid
	}
	return nil
}

// JSON duplicate keys are rejected recursively, including credential responses.
// Nesting is bounded independently of the body-size bound.
func uniqueJSON(d *json.Decoder, depth int) error {
	if depth > 32 {
		return ErrInvalid
	}
	token, err := d.Token()
	if err != nil {
		return ErrInvalid
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return ErrInvalid
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return ErrInvalid
			}
			seen[name] = true
			if uniqueJSON(d, depth+1) != nil {
				return ErrInvalid
			}
		}
	case '[':
		for d.More() {
			if uniqueJSON(d, depth+1) != nil {
				return ErrInvalid
			}
		}
	default:
		return ErrInvalid
	}
	_, err = d.Token()
	if err != nil {
		return ErrInvalid
	}
	return nil
}
