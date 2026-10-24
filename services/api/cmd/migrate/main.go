package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"connect.local/api/internal/config"
	"connect.local/api/internal/database"
	"connect.local/api/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger, os.Args[1:]); err != nil {
		logger.Error("migration command failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, args []string) error {
	if len(args) != 1 || (args[0] != "up" && args[0] != "down" && args[0] != "status") {
		return errors.New("usage: migrate up|down|status")
	}
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	db, err := database.Open(cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	root, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(root, cfg.DBMigrationTimeout)
	defer cancel()
	if err := database.WaitAvailable(ctx, db, cfg.DBConnectTimeout); err != nil {
		return err
	}
	provider, err := migrations.New(db)
	if err != nil {
		return errors.New("migration runner initialization failed")
	}
	switch args[0] {
	case "up":
		_, err = provider.Up(ctx)
	case "down":
		_, err = provider.Down(ctx)
	case "status":
	}
	if err != nil {
		return errors.New("migration operation failed; check database availability, migration lock and schema state")
	}
	version, err := provider.GetDBVersion(ctx)
	if err != nil {
		return errors.New("migration version unavailable")
	}
	logger.Info("migration command completed", "operation", args[0], "version", version)
	return nil
}
