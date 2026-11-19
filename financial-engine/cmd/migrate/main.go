package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	"financial-engine/internal/database"
	"financial-engine/internal/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.OpenPostgres(ctx)
	if err != nil {
		log.Fatalf("PostgreSQL connection failed: %v", err)
	}
	defer pool.Close()

	slog.Info("PostgreSQL connection established")

	if err := migrations.Apply(ctx, pool); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}

	slog.Info("database migration completed successfully")
}
