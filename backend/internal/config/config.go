package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"
)

// Config must never be logged: DatabaseURL may contain credentials.
type Config struct {
	ListenAddr         string
	Environment        string
	DatabaseURL        string
	DBMaxOpen          int
	DBMaxIdle          int
	DBConnectTimeout   time.Duration
	DBStartupTimeout   time.Duration
	DBReadyTimeout     time.Duration
	DBMigrationTimeout time.Duration
}

func Load(getenv func(string) string) (Config, error) {
	addr := getenv("API_LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return Config{}, fmt.Errorf("API_LISTEN_ADDR must be host:port: %w", err)
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return Config{}, fmt.Errorf("API_LISTEN_ADDR port must be between 1 and 65535")
	}
	cfg := Config{ListenAddr: addr, Environment: getenv("APP_ENV"), DatabaseURL: getenv("DATABASE_URL")}
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}
	if cfg.Environment != "development" && cfg.Environment != "test" && cfg.Environment != "production" {
		return Config{}, fmt.Errorf("APP_ENV must be development, test, or production")
	}
	if err := validateDatabaseURL(cfg.DatabaseURL, cfg.Environment); err != nil {
		return Config{}, err
	}
	for _, setting := range []struct {
		name               string
		target             *int
		fallback, min, max int
	}{
		{"DB_MAX_OPEN_CONNS", &cfg.DBMaxOpen, 8, 1, 64},
		{"DB_MAX_IDLE_CONNS", &cfg.DBMaxIdle, 4, 0, 64},
	} {
		*setting.target = setting.fallback
		if raw := getenv(setting.name); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < setting.min || value > setting.max {
				return Config{}, fmt.Errorf("%s is outside the supported range", setting.name)
			}
			*setting.target = value
		}
	}
	if cfg.DBMaxIdle > cfg.DBMaxOpen {
		return Config{}, fmt.Errorf("DB_MAX_IDLE_CONNS must not exceed DB_MAX_OPEN_CONNS")
	}
	for _, setting := range []struct {
		name               string
		target             *time.Duration
		fallback, min, max time.Duration
	}{
		{"DB_CONNECT_TIMEOUT", &cfg.DBConnectTimeout, 2 * time.Second, 100 * time.Millisecond, 30 * time.Second},
		{"DB_STARTUP_TIMEOUT", &cfg.DBStartupTimeout, 10 * time.Second, 100 * time.Millisecond, 2 * time.Minute},
		{"DB_READY_TIMEOUT", &cfg.DBReadyTimeout, time.Second, 50 * time.Millisecond, 5 * time.Second},
		{"DB_MIGRATION_TIMEOUT", &cfg.DBMigrationTimeout, 30 * time.Second, time.Second, 5 * time.Minute},
	} {
		*setting.target = setting.fallback
		if raw := getenv(setting.name); raw != "" {
			value, err := time.ParseDuration(raw)
			if err != nil || value < setting.min || value > setting.max {
				return Config{}, fmt.Errorf("%s is outside the supported duration range", setting.name)
			}
			*setting.target = value
		}
	}
	return cfg, nil
}

func validateDatabaseURL(raw, environment string) error {
	// Never wrap URL/parser errors: they can contain passwords or untrusted input.
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 8192 || u == nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Hostname() == "" || u.Path == "" || u.Path == "/" || u.User == nil || u.User.Username() == "" || u.Fragment != "" {
		return fmt.Errorf("DATABASE_URL must be a PostgreSQL URL with user, host and database")
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return fmt.Errorf("DATABASE_URL has invalid options")
	}
	for name, values := range query {
		if len(values) != 1 {
			return fmt.Errorf("DATABASE_URL options must not be duplicated")
		}
		switch name {
		case "sslmode", "sslrootcert", "sslcert", "sslkey":
		default:
			return fmt.Errorf("DATABASE_URL contains an unsupported option")
		}
	}
	host := u.Hostname()
	loopback := host == "localhost"
	if ip := net.ParseIP(host); ip != nil {
		loopback = ip.IsLoopback()
	}
	mode := query.Get("sslmode")
	if mode == "verify-full" {
		return nil
	}
	if mode == "disable" && loopback && environment != "production" {
		return nil
	}
	return fmt.Errorf("DATABASE_URL requires sslmode=verify-full; plaintext is allowed only for non-production loopback PostgreSQL")
}
