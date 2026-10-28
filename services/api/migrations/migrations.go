package migrations

import (
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

const LatestVersion int64 = 1

//go:embed *.sql
var files embed.FS

// New uses a PostgreSQL session advisory lock and Goose's default transactions.
// Migrations are explicit CLI operations, never an API startup side effect.
func New(db *sql.DB) (*goose.Provider, error) {
	locker, err := lock.NewPostgresSessionLocker(lock.WithLockTimeout(1, 5), lock.WithUnlockTimeout(1, 2))
	if err != nil {
		return nil, err
	}
	return goose.NewProvider(goose.DialectPostgres, db, files, goose.WithSessionLocker(locker), goose.WithDisableGlobalRegistry(true), goose.WithTableName("public.goose_db_version"))
}
