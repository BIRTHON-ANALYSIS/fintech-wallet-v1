package migrations

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed 001_init.sql
var migrationFiles embed.FS

const initialMigration = "001_init.sql"

func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("migration database pool is nil")
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Create a migration history table. This is safe to run repeatedly.
	_, err = tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create migration history: %w", err)
	}

	var alreadyApplied bool
	err = tx.QueryRow(
		ctx,
		`SELECT EXISTS (
			SELECT 1 FROM schema_migrations WHERE version = $1
		)`,
		initialMigration,
	).Scan(&alreadyApplied)

	if err != nil {
		return fmt.Errorf("check migration history: %w", err)
	}

	if alreadyApplied {
		return tx.Commit(ctx)
	}

	sqlBytes, err := migrationFiles.ReadFile(initialMigration)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", initialMigration, err)
	}

	sql := strings.TrimSpace(string(sqlBytes))
	if sql == "" {
		return fmt.Errorf("migration %s is empty", initialMigration)
	}

	// Execute the schema and migration record in one transaction.
	// The migration SQL must be trusted application code.
	_, err = tx.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("execute migration %s: %w", initialMigration, err)
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1)`,
		initialMigration,
	)
	if err != nil {
		return fmt.Errorf("record migration %s: %w", initialMigration, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", initialMigration, err)
	}

	return nil
}
