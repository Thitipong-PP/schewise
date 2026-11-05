package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Thitipong-PP/schewise/internal/auth"
	"github.com/Thitipong-PP/schewise/internal/config"
	"github.com/Thitipong-PP/schewise/internal/database"
	"github.com/Thitipong-PP/schewise/internal/httpapi"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err = run(ctx, logger)
	stop()
	if err != nil {
		logger.Error("API stopped", zap.Error(err))
		_ = logger.Sync()
		os.Exit(1)
	}
	_ = logger.Sync()
}

func run(ctx context.Context, logger *zap.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	startup, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	db, err := database.Open(startup, cfg.DatabaseURL)
	if err != nil {
		return errors.New("application database connection failed; check database configuration and availability")
	}
	defer func() { _ = db.Close() }()
	if err := database.Migrate(startup, db); err != nil {
		return errors.New("application database migration failed")
	}
	logger.Info("database.migrated")
	service, err := auth.New(db, logger, cfg.SessionTTL, auth.InviteConfig{PublicBaseURL: cfg.PublicBaseURL, TTL: cfg.InviteTTL})
	if err != nil {
		return errors.New("authentication initialization failed")
	}
	if _, err := service.Bootstrap(startup, cfg.AdminEmail, cfg.AdminPassword); err != nil {
		return errors.New("administrator bootstrap failed; verify installation credentials and database availability")
	}
	app := httpapi.NewApp(httpapi.Config{Auth: service, RBAC: rbac.NewRepository(db, logger, nil), Logger: logger, Origin: cfg.Origin})
	cleanupCtx, cleanupCancel := context.WithCancel(ctx)
	defer cleanupCancel()
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-cleanupCtx.Done():
				return
			case <-ticker.C:
				cleanup, done := context.WithTimeout(cleanupCtx, 10*time.Second)
				if err := service.Cleanup(cleanup); err != nil {
					logger.Error("session.cleanup_failed")
				} else {
					logger.Info("session.cleanup_completed")
				}
				done()
			}
		}
	}()
	errCh := make(chan error, 1)
	go func() {
		logger.Info("API listening", zap.String("address", cfg.Address))
		errCh <- app.Listen(cfg.Address, fiber.ListenConfig{DisableStartupMessage: true})
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("API shutting down")
	}
	shutdown, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()
	if err := app.ShutdownWithContext(shutdown); err != nil {
		return errors.New("API shutdown failed")
	}
	return <-errCh
}
