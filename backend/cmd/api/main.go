package main

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connect.local/api/internal/accounts"
	accountspg "connect.local/api/internal/accounts/postgres"
	"connect.local/api/internal/config"
	"connect.local/api/internal/database"
	"connect.local/api/internal/httpserver"
	"connect.local/api/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	authConfig, err := config.LoadAuth(os.Getenv)
	if err != nil {
		return err
	}
	certFile, keyFile := os.Getenv("API_TLS_CERT_FILE"), os.Getenv("API_TLS_KEY_FILE")
	var tlsConfig *tls.Config
	if certFile != "" || keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return errors.New("API TLS certificate/key could not be loaded")
		}
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{cert}}
	}
	if authConfig.Enabled && tlsConfig == nil {
		return errors.New("account routes require API_TLS_CERT_FILE and API_TLS_KEY_FILE")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := database.Open(cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	startup, cancel := context.WithTimeout(ctx, cfg.DBStartupTimeout)
	err = database.WaitAvailable(startup, db, cfg.DBConnectTimeout)
	cancel()
	if err != nil {
		return err
	}
	var registerRoutes func(*http.ServeMux)
	var accountService *accounts.Service
	if authConfig.Enabled {
		accountService, err = accounts.New(accountspg.New(db), authConfig.RPID, authConfig.Origins, authConfig.AndroidOrigins)
		if err != nil {
			return err
		}
		registerRoutes = accountService.RegisterRoutes
	}
	srv := httpserver.New(cfg.ListenAddr, httpserver.Options{
		Ready:          func(ctx context.Context) error { return database.Ready(ctx, db, migrations.LatestVersion) },
		ReadyTimeout:   cfg.DBReadyTimeout,
		DatabaseStats:  db.Stats,
		RegisterRoutes: registerRoutes,
	})
	// net/http TLS errors include peer addresses and may carry untrusted text.
	// Emit only a static transport event, without forwarding the raw message.
	srv.ErrorLog = log.New(transportLog{logger}, "", 0)
	done := make(chan error, 1)
	srv.TLSConfig = tlsConfig
	cleanupCtx, cleanupCancel := context.WithCancel(ctx)
	defer cleanupCancel()
	cleanupDone := make(chan struct{})
	if accountService != nil {
		go func() { defer close(cleanupDone); accountService.RunCleanup(cleanupCtx) }()
	} else {
		close(cleanupDone)
	}
	defer func() { cleanupCancel(); <-cleanupDone }()
	go func() {
		if tlsConfig != nil {
			done <- srv.ListenAndServeTLS("", "")
		} else {
			done <- srv.ListenAndServe()
		}
	}()
	logger.Info("api starting", "address", cfg.ListenAddr)
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			_ = srv.Close()
			return err
		}
		err := <-done
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		logger.Info("api stopped")
		return nil
	}
}

type transportLog struct{ logger *slog.Logger }

func (l transportLog) Write(message []byte) (int, error) {
	l.logger.Warn("http transport error")
	return len(message), nil
}
