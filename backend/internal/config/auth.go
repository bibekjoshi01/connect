package config

import (
	"encoding/base64"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type Auth struct {
	Enabled        bool
	RPID           string
	Origins        []string
	AndroidOrigins []string
}

// LoadAuth is separate so migrations do not need RP or web-server credentials.
// There is no implicit development RP/origin, wildcard or trust-forwarded switch.
func LoadAuth(getenv func(string) string) (Auth, error) {
	cfg := Auth{RPID: getenv("AUTH_RP_ID")}
	raw, android := getenv("AUTH_RP_ORIGINS"), getenv("AUTH_ANDROID_ORIGINS")
	if cfg.RPID == "" && raw == "" && android == "" {
		return cfg, nil
	}
	invalid := errors.New("AUTH_RP_ID and exact HTTPS AUTH_RP_ORIGINS are required; Android origins must be exact certificate hashes")
	if len(cfg.RPID) > 253 || cfg.RPID == "" || raw == "" || len(raw) > 2048 || len(android) > 2048 || strings.ContainsAny(cfg.RPID, "/:* \\?#@") || net.ParseIP(cfg.RPID) != nil || cfg.RPID != strings.ToLower(cfg.RPID) {
		return Auth{}, invalid
	}
	cfg.Origins = strings.Split(raw, ",")
	if len(cfg.Origins) > 8 {
		return Auth{}, invalid
	}
	seen := map[string]bool{}
	for _, origin := range cfg.Origins {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.Hostname() != strings.ToLower(u.Hostname()) || strings.Contains(u.Host, "*") || seen[origin] {
			return Auth{}, invalid
		}
		if u.Port() != "" {
			port, err := strconv.Atoi(u.Port())
			if err != nil || port < 1 || port > 65535 {
				return Auth{}, invalid
			}
		}
		if u.Hostname() != cfg.RPID && !strings.HasSuffix(u.Hostname(), "."+cfg.RPID) {
			return Auth{}, invalid
		}
		seen[origin] = true
	}
	if android != "" {
		cfg.AndroidOrigins = strings.Split(android, ",")
		if len(cfg.AndroidOrigins) > 8 {
			return Auth{}, invalid
		}
		for _, origin := range cfg.AndroidOrigins {
			encoded, ok := strings.CutPrefix(origin, "android:apk-key-hash:")
			b, err := base64.RawURLEncoding.Strict().DecodeString(encoded)
			if !ok || err != nil || len(b) != 32 || len(encoded) != 43 || seen[origin] {
				return Auth{}, invalid
			}
			seen[origin] = true
		}
	}
	cfg.Enabled = true
	return cfg, nil
}
