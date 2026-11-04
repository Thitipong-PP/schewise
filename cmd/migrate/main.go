// Command migrate manages only Schewise's application database.
package main

import (
	"context"
	"os"
	"time"

	"github.com/Thitipong-PP/schewise/internal/config"
	"github.com/Thitipong-PP/schewise/internal/database"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()
	if len(os.Args) != 2 || (os.Args[1] != "up" && os.Args[1] != "status") {
		logger.Error("usage: go run ./cmd/migrate <up|status>")
		os.Exit(1)
	}
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid database configuration")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed")
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()
	provider, err := database.Provider(db)
	if err != nil {
		logger.Error("migration initialization failed")
		os.Exit(1)
	}
	if os.Args[1] == "up" {
		_, err = provider.Up(ctx)
	} else {
		var version int64
		version, err = provider.GetDBVersion(ctx)
		if err == nil {
			logger.Info("migration.status", zap.Int64("version", version))
		}
	}
	if err != nil {
		logger.Error("migration operation failed")
		os.Exit(1)
	}
	logger.Info("migration.completed")
}
