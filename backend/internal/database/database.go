// Package database owns PostgreSQL infrastructure; it contains no account logic.
package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"connect.local/api/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func Open(cfg config.Config) (*sql.DB, error) {
	connection, err := pgx.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, errors.New("invalid PostgreSQL connection configuration")
	}
	connection.ConnectTimeout = cfg.DBConnectTimeout
	db := stdlib.OpenDB(*connection)
	db.SetMaxOpenConns(cfg.DBMaxOpen)
	db.SetMaxIdleConns(cfg.DBMaxIdle)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)
	return db, nil
}

// WaitAvailable retries startup connectivity within the caller's total deadline.
// Connection/driver errors are deliberately not returned to logs or HTTP clients.
func WaitAvailable(ctx context.Context, db *sql.DB, attemptTimeout time.Duration) error {
	for {
		attempt, cancel := context.WithTimeout(ctx, attemptTimeout)
		err := db.PingContext(attempt)
		cancel()
		if err == nil {
			return nil
		}
		timer := time.NewTimer(250 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.New("PostgreSQL unavailable before startup deadline")
		case <-timer.C:
		}
	}
}

// Ready is read-only. It fails closed on connection errors or missing migrations.
func Ready(ctx context.Context, db *sql.DB, expectedVersion int64) error {
	var version int64
	var namespaceExists bool
	err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version_id), 0), to_regnamespace('connect') IS NOT NULL FROM public.goose_db_version WHERE is_applied`).Scan(&version, &namespaceExists)
	if err != nil {
		return errors.New("PostgreSQL readiness unavailable")
	}
	if version != expectedVersion || !namespaceExists {
		return errors.New("PostgreSQL schema is not current")
	}
	return nil
}
