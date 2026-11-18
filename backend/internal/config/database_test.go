package config

import (
	"strings"
	"testing"
)

func TestDatabaseConfiguration(t *testing.T) {
	for _, tt := range []struct {
		name      string
		overrides map[string]string
		invalid   bool
	}{
		{"local development", nil, false},
		{"remote verified TLS", map[string]string{"DATABASE_URL": "postgres://connect:secret@db.example.com/connect?sslmode=verify-full", "APP_ENV": "production"}, false},
		{"missing URL", map[string]string{"DATABASE_URL": ""}, true},
		{"redacted malformed URL", map[string]string{"DATABASE_URL": "postgres://user:highly-sensitive%wrong@127.0.0.1/connect"}, true},
		{"remote plaintext", map[string]string{"DATABASE_URL": "postgres://connect:secret@db.example.com/connect?sslmode=disable"}, true},
		{"production plaintext", map[string]string{"APP_ENV": "production"}, true},
		{"unverified TLS", map[string]string{"DATABASE_URL": "postgres://connect:secret@db.example.com/connect?sslmode=require"}, true},
		{"host override", map[string]string{"DATABASE_URL": "postgres://connect:secret@127.0.0.1/connect?sslmode=disable&host=remote.example.com"}, true},
		{"duplicate mode", map[string]string{"DATABASE_URL": "postgres://connect:secret@127.0.0.1/connect?sslmode=disable&sslmode=verify-full"}, true},
		{"unknown environment", map[string]string{"APP_ENV": "demo"}, true},
		{"zero pool", map[string]string{"DB_MAX_OPEN_CONNS": "0"}, true},
		{"oversized pool", map[string]string{"DB_MAX_OPEN_CONNS": "10000"}, true},
		{"idle exceeds open", map[string]string{"DB_MAX_OPEN_CONNS": "2"}, true},
		{"negative idle", map[string]string{"DB_MAX_IDLE_CONNS": "-1"}, true},
		{"zero deadline", map[string]string{"DB_READY_TIMEOUT": "0s"}, true},
		{"excessive deadline", map[string]string{"DB_READY_TIMEOUT": "60s"}, true},
		{"invalid deadline", map[string]string{"DB_STARTUP_TIMEOUT": "forever"}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			values := map[string]string{"DATABASE_URL": "postgres://connect:local@127.0.0.1/connect?sslmode=disable"}
			for key, value := range tt.overrides {
				values[key] = value
			}
			cfg, err := Load(func(key string) string { return values[key] })
			if (err != nil) != tt.invalid {
				t.Fatalf("invalid = %v, error = %v", tt.invalid, err)
			}
			if err != nil && (strings.Contains(err.Error(), "highly-sensitive") || strings.Contains(err.Error(), "secret")) {
				t.Fatal("configuration error leaked sensitive input")
			}
			if !tt.invalid && (cfg.DBMaxOpen != 8 || cfg.DBMaxIdle != 4 || cfg.DBReadyTimeout <= 0) {
				t.Fatal("unexpected defaults")
			}
		})
	}
}
