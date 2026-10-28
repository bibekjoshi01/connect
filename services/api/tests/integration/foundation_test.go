//go:build integration

package integration

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"connect.local/api/internal/config"
	"connect.local/api/internal/database"
	"connect.local/api/internal/httpserver"
	"connect.local/api/migrations"
)

func TestPostgresFoundation(t *testing.T) {
	raw := os.Getenv("TEST_DATABASE_URL")
	u, err := url.Parse(raw)
	if err != nil || u == nil || u.Path != "/connect_stage1_test" {
		t.Fatal("Use make api-integration with the dedicated connect_stage1_test database")
	}
	cfg, err := config.Load(func(key string) string {
		if key == "DATABASE_URL" {
			return raw
		}
		if key == "APP_ENV" {
			return "test"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := database.WaitAvailable(ctx, db, cfg.DBConnectTimeout); err != nil {
		t.Fatal(err)
	}
	if db.Stats().MaxOpenConnections != cfg.DBMaxOpen {
		t.Fatal("pool was not bounded")
	}
	if err := database.Ready(ctx, db, migrations.LatestVersion); err == nil {
		t.Fatal("unmigrated database reported ready")
	}
	provider, err := migrations.New(db)
	if err != nil {
		t.Fatal(err)
	}
	up, err := provider.Up(ctx)
	if err != nil || len(up) != 1 {
		t.Fatalf("fresh migration: count=%d error=%v", len(up), err)
	}
	up, err = provider.Up(ctx)
	if err != nil || len(up) != 0 {
		t.Fatalf("migration was not idempotent: count=%d error=%v", len(up), err)
	}
	if err := database.Ready(ctx, db, migrations.LatestVersion); err != nil {
		t.Fatal(err)
	}

	t.Run("readiness deadline and recovery", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(ctx, "LOCK TABLE public.goose_db_version IN ACCESS EXCLUSIVE MODE"); err != nil {
			t.Fatal(err)
		}
		blocked, stop := context.WithTimeout(ctx, 75*time.Millisecond)
		defer stop()
		start := time.Now()
		if err := database.Ready(blocked, db, migrations.LatestVersion); err == nil {
			t.Fatal("blocked readiness succeeded")
		}
		if time.Since(start) > time.Second {
			t.Fatal("readiness deadline was not enforced")
		}
		if err := tx.Rollback(); err != nil {
			t.Fatal(err)
		}
		if err := database.Ready(ctx, db, migrations.LatestVersion); err != nil {
			t.Fatal("readiness did not recover")
		}
	})

	t.Run("migration lock contention", func(t *testing.T) {
		conn, err := db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock(4097083626)"); err != nil {
			t.Fatal(err)
		}
		defer conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock(4097083626)")
		locked, stop := context.WithTimeout(ctx, 100*time.Millisecond)
		defer stop()
		if _, err := provider.Down(locked); err == nil {
			t.Fatal("migration ignored advisory lock")
		}
		if err := database.Ready(ctx, db, migrations.LatestVersion); err != nil {
			t.Fatal("lock contention changed schema")
		}
	})

	t.Run("rollback protects application objects", func(t *testing.T) {
		if _, err := db.ExecContext(ctx, "CREATE TABLE connect.qa_guard (id integer)"); err != nil {
			t.Fatal(err)
		}
		if _, err := provider.Down(ctx); err == nil {
			t.Fatal("rollback deleted nonempty application namespace")
		}
		version, err := provider.GetDBVersion(ctx)
		if err != nil || version != migrations.LatestVersion {
			t.Fatal("failed rollback changed migration version")
		}
		if _, err := db.ExecContext(ctx, "DROP TABLE connect.qa_guard"); err != nil {
			t.Fatal(err)
		}
	})

	if _, err := provider.Down(ctx); err != nil {
		t.Fatal(err)
	}
	if err := database.Ready(ctx, db, migrations.LatestVersion); err == nil {
		t.Fatal("rolled-back database reported ready")
	}
	var workers sync.WaitGroup
	errors := make(chan error, 2)
	for i := 0; i < 2; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			concurrent, err := migrations.New(db)
			if err == nil {
				_, err = concurrent.Up(ctx)
			}
			errors <- err
		}()
	}
	workers.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal("concurrent migration failed", err)
		}
	}
	if err := database.Ready(ctx, db, migrations.LatestVersion); err != nil {
		t.Fatal("reapplied schema was not ready")
	}

	srv := httpserver.New("unused", httpserver.Options{Ready: func(ctx context.Context) error { return database.Ready(ctx, db, migrations.LatestVersion) }, DatabaseStats: db.Stats})
	response := httptest.NewRecorder()
	srv.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusOK {
		t.Fatal("migrated service was not ready")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	srv.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatal("closed database reported ready")
	}
	live := httptest.NewRecorder()
	srv.Handler.ServeHTTP(live, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if live.Code != http.StatusOK {
		t.Fatal("dependency outage broke liveness")
	}
}

func TestStartupDeadline(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	cfg, err := config.Load(func(key string) string {
		if key == "DATABASE_URL" {
			return "postgres://local:unused@" + listener.Addr().String() + "/connect_stage1_test?sslmode=disable"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := database.WaitAvailable(ctx, db, 75*time.Millisecond); err == nil {
		t.Fatal("non-PostgreSQL listener passed startup")
	}
	if time.Since(start) > time.Second {
		t.Fatal("startup retry exceeded deadline")
	}
}
