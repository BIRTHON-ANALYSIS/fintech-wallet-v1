package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"financial-engine/internal/domain"
	"financial-engine/internal/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) TransferTx(
	ctx context.Context,
	cmd ports.TransferCommand,
) (domain.Transaction, error) {
	var zero domain.Transaction

	if r == nil || r.db == nil {
		return zero, errors.New("PostgreSQL repository is not initialized")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if strings.TrimSpace(cmd.TransactionID) == "" ||
		strings.TrimSpace(cmd.Reference) == "" ||
		strings.TrimSpace(cmd.SourceWalletID) == "" ||
		strings.TrimSpace(cmd.DestWalletID) == "" {
		return zero, errors.New("transfer identifiers are required")
	}
	if cmd.SourceWalletID == cmd.DestWalletID {
		return zero, errors.New("source and destination wallets must differ")
	}
	if err := cmd.Amount.Validate(); err != nil {
		return zero, fmt.Errorf("invalid transfer amount: %w", err)
	}
	if err := cmd.Idempotency.Validate(); err != nil {
		return zero, fmt.Errorf("invalid idempotency record: %w", err)
	}
	if len(cmd.Entries) != 2 {
		return zero, errors.New("a transfer must contain exactly two ledger entries")
	}
	if err := domain.ValidateBalancedEntries(cmd.Entries); err != nil {
		return zero, fmt.Errorf("invalid ledger entries: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		return zero, fmt.Errorf("begin transfer transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Reserve the idempotency key. Concurrent requests using the same key
	// cannot both execute the transfer.
	tag, err := tx.Exec(ctx, `
		INSERT INTO idempotency_records
			(idempotency_key, request_hash, transaction_id, status, created_at, expires_at)
		VALUES ($1, $2, $3, 'processing', NOW(), $4)
		ON CONFLICT (idempotency_key) DO NOTHING
	`,
		cmd.Idempotency.Key,
		cmd.Idempotency.RequestHash,
		cmd.TransactionID,
		cmd.Idempotency.ExpiresAt,
	)
	if err != nil {
		return zero, fmt.Errorf("reserve idempotency key: %w", err)
	}

	if tag.RowsAffected() == 0 {
		var existingHash string
		var existingTransactionID *string
		var existingStatus string

		err = tx.QueryRow(ctx, `
			SELECT request_hash, transaction_id, status
			FROM idempotency_records
			WHERE idempotency_key = $1
			FOR UPDATE
		`, cmd.Idempotency.Key).Scan(
			&existingHash,
			&existingTransactionID,
			&existingStatus,
		)
		if err != nil {
			return zero, fmt.Errorf("read existing idempotency record: %w", err)
		}

		if existingHash != cmd.Idempotency.RequestHash {
			return zero, ports.ErrIdempotencyConflict
		}

		if existingStatus == "completed" &&
			existingTransactionID != nil {
			return r.loadTransaction(ctx, tx, *existingTransactionID)
		}

		return zero, domain.ErrDuplicateRequest
	}

	// Lock wallets in deterministic order to reduce deadlock risk.
	firstID, secondID := cmd.SourceWalletID, cmd.DestWalletID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	rows, err := tx.Query(ctx, `
		SELECT id, user_id, currency, balance_minor, status,
		       created_at, updated_at
		FROM wallets
		WHERE id = ANY($1::text[])
		ORDER BY id
		FOR UPDATE
	`, []string{firstID, secondID})
	if err != nil {
		return zero, fmt.Errorf("lock transfer wallets: %w", err)
	}

	type walletRow struct {
		id        string
		userID    string
		currency  string
		balance   int64
		status    string
		createdAt time.Time
		updatedAt time.Time
	}

	locked := make(map[string]walletRow, 2)

	for rows.Next() {
		var w walletRow
		if err := rows.Scan(
			&w.id,
			&w.userID,
			&w.currency,
			&w.balance,
			&w.status,
			&w.createdAt,
			&w.updatedAt,
		); err != nil {
			rows.Close()
			return zero, fmt.Errorf("scan locked wallet: %w", err)
		}
		locked[w.id] = w
	}

	err = rows.Err()
	rows.Close()
	if err != nil {
		return zero, fmt.Errorf("read locked wallets: %w", err)
	}

	source, sourceOK := locked[cmd.SourceWalletID]
	destination, destinationOK := locked[cmd.DestWalletID]

	if !sourceOK || !destinationOK {
		return zero, ports.ErrNotFound
	}
	if source.status != "active" || destination.status != "active" {
		return zero, ports.ErrWalletNotActive
	}
	if source.currency != cmd.Amount.Currency ||
		destination.currency != cmd.Amount.Currency {
		return zero, ports.ErrCurrencyMismatch
	}
	if source.balance < cmd.Amount.AmountMinor {
		return zero, ports.ErrInsufficientFunds
	}
	if destination.balance > math.MaxInt64-cmd.Amount.AmountMinor {
		return zero, errors.New("destination wallet balance overflow")
	}

	// Persist the transaction before the ledger entries. All changes remain
	// uncommitted until the final COMMIT.
	_, err = tx.Exec(ctx, `
		INSERT INTO financial_transactions
			(id, reference, idempotency_key, type, status, amount_minor,
			 currency, source_wallet_id, destination_wallet_id, description,
			 created_at, updated_at)
		VALUES ($1, $2, $3, 'transfer', 'processing', $4, $5, $6, $7, $8, NOW(), NOW())
	`,
		cmd.TransactionID,
		cmd.Reference,
		cmd.Idempotency.Key,
		cmd.Amount.AmountMinor,
		cmd.Amount.Currency,
		cmd.SourceWalletID,
		cmd.DestWalletID,
		cmd.Description,
	)
	if err != nil {
		return zero, fmt.Errorf("create financial transaction: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE wallets
		SET balance_minor = balance_minor - $1, updated_at = NOW()
		WHERE id = $2
	`, cmd.Amount.AmountMinor, cmd.SourceWalletID)
	if err != nil {
		return zero, fmt.Errorf("debit source wallet: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE wallets
		SET balance_minor = balance_minor + $1, updated_at = NOW()
		WHERE id = $2
	`, cmd.Amount.AmountMinor, cmd.DestWalletID)
	if err != nil {
		return zero, fmt.Errorf("credit destination wallet: %w", err)
	}

	for _, entry := range cmd.Entries {
		if entry.TransactionID != cmd.TransactionID ||
			entry.AmountMinor != cmd.Amount.AmountMinor ||
			entry.Currency != cmd.Amount.Currency {
			return zero, errors.New("ledger entry does not match transfer")
		}

		if entry.AccountID != cmd.SourceWalletID &&
			entry.AccountID != cmd.DestWalletID {
			return zero, errors.New("ledger entry references an unrelated wallet")
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO ledger_entries
				(id, transaction_id, account_id, side, amount_minor,
				 currency, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())
		`,
			entry.ID,
			entry.TransactionID,
			entry.AccountID,
			string(entry.Side),
			entry.AmountMinor,
			entry.Currency,
		)
		if err != nil {
			return zero, fmt.Errorf("insert ledger entry: %w", err)
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE financial_transactions
		SET status = 'succeeded', updated_at = NOW()
		WHERE id = $1
	`, cmd.TransactionID)
	if err != nil {
		return zero, fmt.Errorf("complete financial transaction: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE idempotency_records
		SET status = 'completed', transaction_id = $2
		WHERE idempotency_key = $1
	`, cmd.Idempotency.Key, cmd.TransactionID)
	if err != nil {
		return zero, fmt.Errorf("complete idempotency record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return zero, fmt.Errorf("commit transfer: %w", err)
	}

	// Return the persisted transaction using the same domain mapping as
	// idempotent retries.
	return r.loadTransaction(ctx, r.db, cmd.TransactionID)
}

type transactionQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (r *PostgresRepository) loadTransaction(
	ctx context.Context,
	q transactionQuerier,
	id string,
) (domain.Transaction, error) {
	var result domain.Transaction
	var txType string
	var status string
	var amountMinor int64
	var currency string

	err := q.QueryRow(ctx, `
		SELECT id, reference, idempotency_key, type, status, amount_minor,
		       currency, source_wallet_id, destination_wallet_id, description,
		       created_at, updated_at
		FROM financial_transactions
		WHERE id = $1
	`, id).Scan(
		&result.ID,
		&result.Reference,
		&result.IdempotencyKey,
		&txType,
		&status,
		&amountMinor,
		&currency,
		&result.SourceWalletID,
		&result.DestWalletID,
		&result.Description,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return result, fmt.Errorf("load financial transaction: %w", err)
	}

	// Convert database strings into the domain's named string types.
	// If your domain uses different type names, these two assignments
	// are the only mapping lines that need adjustment.
	result.Type = domain.TransactionType(txType)
	result.Status = domain.TransactionStatus(status)
	result.Amount = domain.Money{
		AmountMinor: amountMinor,
		Currency:    currency,
	}

	return result, nil
}
