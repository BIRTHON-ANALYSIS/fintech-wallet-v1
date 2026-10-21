package ports

import (
	"context"
	"errors"

	"financial-engine/internal/domain"
)

var ErrNotFound = errors.New("record not found")

// Repository defines the persistence operations needed by
// the financial engine.
//
// Implementations must guarantee that TransferTx executes
// all its changes atomically.
type Repository interface {
	TransferTx(
		ctx context.Context,
		command TransferCommand,
	) (domain.Transaction, error)
}

// TransferCommand contains the information required to
// execute a wallet-to-wallet transfer.
type TransferCommand struct {
	Transaction domain.Transaction
	Source      domain.Wallet
	Destination domain.Wallet
	Entries     []domain.LedgerEntry
	Idempotency domain.IdempotencyRecord
}
