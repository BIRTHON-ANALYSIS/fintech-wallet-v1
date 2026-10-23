package ports

import (
	"context"
	"errors"

	"financial-engine/internal/domain"
)

var (
	ErrNotFound            = errors.New("record not found")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different request")
	ErrWalletNotActive     = errors.New("wallet is not active")
	ErrCurrencyMismatch    = errors.New("currency mismatch")
)

// Repository defines the persistence operations required by the service.
type Repository interface {
	// TransferTx must validate wallet state and save all transfer
	// changes atomically.
	TransferTx(
		ctx context.Context,
		command TransferCommand,
	) (domain.Transaction, error)
}

// TransferCommand carries a transfer request to the repository.
type TransferCommand struct {
	TransactionID  string
	Reference      string
	SourceWalletID string
	DestWalletID   string
	Amount         domain.Money
	Description    string
	Entries        []domain.LedgerEntry
	Idempotency    domain.IdempotencyRecord
}
