package repository

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"time"

	"financial-engine/internal/domain"
	"financial-engine/internal/ports"
)

var (
	ErrInvalidTransferCommand = errors.New(
		"invalid transfer command",
	)
	ErrDestinationBalanceOverflow = errors.New(
		"destination wallet balance overflow",
	)
)

// MemoryRepository is a temporary in-memory implementation
// of the financial repository.
//
// It is useful for development, but it is not persistent storage.
type MemoryRepository struct {
	mu sync.RWMutex

	wallets       map[string]domain.Wallet
	transactions  map[string]domain.Transaction
	ledgerEntries map[string][]domain.LedgerEntry
	idempotency   map[string]domain.IdempotencyRecord
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		wallets:       make(map[string]domain.Wallet),
		transactions:  make(map[string]domain.Transaction),
		ledgerEntries: make(map[string][]domain.LedgerEntry),
		idempotency:   make(map[string]domain.IdempotencyRecord),
	}
}

// AddWallet adds a wallet to the in-memory repository.
// This is intended for development setup and demonstrations.
func (r *MemoryRepository) AddWallet(
	wallet domain.Wallet,
) error {
	if r == nil {
		return errors.New("repository is nil")
	}

	if err := wallet.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.wallets[wallet.ID]; exists {
		return errors.New("wallet already exists")
	}

	r.wallets[wallet.ID] = wallet
	return nil
}

// GetWallet returns a snapshot of a wallet.
func (r *MemoryRepository) GetWallet(
	walletID string,
) (domain.Wallet, error) {
	if r == nil {
		return domain.Wallet{}, errors.New("repository is nil")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	wallet, exists := r.wallets[walletID]
	if !exists {
		return domain.Wallet{}, ports.ErrNotFound
	}

	return wallet, nil
}

// GetTransaction returns a stored transaction.
func (r *MemoryRepository) GetTransaction(
	transactionID string,
) (domain.Transaction, error) {
	if r == nil {
		return domain.Transaction{}, errors.New("repository is nil")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	transaction, exists := r.transactions[transactionID]
	if !exists {
		return domain.Transaction{}, ports.ErrNotFound
	}

	return transaction, nil
}

// GetLedgerEntries returns a copy of the stored ledger entries.
func (r *MemoryRepository) GetLedgerEntries(
	transactionID string,
) ([]domain.LedgerEntry, error) {
	if r == nil {
		return nil, errors.New("repository is nil")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	entries, exists := r.ledgerEntries[transactionID]
	if !exists {
		return nil, ports.ErrNotFound
	}

	return append([]domain.LedgerEntry(nil), entries...), nil
}

// TransferTx performs the complete transfer while holding a lock.
//
// In production, the equivalent work must be performed inside
// a database transaction with appropriate wallet-row locking.
func (r *MemoryRepository) TransferTx(
	ctx context.Context,
	command ports.TransferCommand,
) (domain.Transaction, error) {
	if r == nil {
		return domain.Transaction{}, errors.New("repository is nil")
	}

	if err := ctx.Err(); err != nil {
		return domain.Transaction{}, err
	}

	if strings.TrimSpace(command.TransactionID) == "" ||
		strings.TrimSpace(command.Reference) == "" ||
		strings.TrimSpace(command.SourceWalletID) == "" ||
		strings.TrimSpace(command.DestWalletID) == "" ||
		command.SourceWalletID == command.DestWalletID {
		return domain.Transaction{}, ErrInvalidTransferCommand
	}

	if err := command.Amount.Validate(); err != nil {
		return domain.Transaction{}, err
	}

	if err := command.Idempotency.Validate(); err != nil {
		return domain.Transaction{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check idempotency before processing a new transfer.
	if existing, found := r.idempotency[command.Idempotency.Key]; found {
		if existing.RequestHash != command.Idempotency.RequestHash {
			return domain.Transaction{}, ports.ErrIdempotencyConflict
		}

		// A completed matching request returns its original result.
		if existing.Status == domain.IdempotencyCompleted &&
			existing.TransactionID != "" {
			transaction, exists := r.transactions[existing.TransactionID]
			if exists {
				return transaction, nil
			}
		}

		return domain.Transaction{}, domain.ErrDuplicateRequest
	}

	source, sourceExists := r.wallets[command.SourceWalletID]
	if !sourceExists {
		return domain.Transaction{}, ports.ErrNotFound
	}

	destination, destinationExists := r.wallets[command.DestWalletID]
	if !destinationExists {
		return domain.Transaction{}, ports.ErrNotFound
	}

	// Validate the actual stored wallets, not caller-supplied balances.
	if err := domain.ValidateTransfer(
		source,
		destination,
		command.Amount,
	); err != nil {
		switch {
		case errors.Is(err, domain.ErrInsufficientFunds):
			return domain.Transaction{}, ports.ErrInsufficientFunds
		case errors.Is(err, domain.ErrWalletNotActive):
			return domain.Transaction{}, ports.ErrWalletNotActive
		case errors.Is(err, domain.ErrCurrencyMismatch):
			return domain.Transaction{}, ports.ErrCurrencyMismatch
		default:
			return domain.Transaction{}, err
		}
	}

	// Verify that the supplied entries form the expected balanced transfer.
	entries := append(
		[]domain.LedgerEntry(nil),
		command.Entries...,
	)

	if len(entries) != 2 {
		return domain.Transaction{}, ErrInvalidTransferCommand
	}

	if err := domain.ValidateBalancedEntries(entries); err != nil {
		return domain.Transaction{}, err
	}

	seenDebit := false
	seenCredit := false

	for _, entry := range entries {
		if entry.TransactionID != command.TransactionID ||
			entry.AmountMinor != command.Amount.AmountMinor ||
			!strings.EqualFold(entry.Currency, command.Amount.Currency) {
			return domain.Transaction{}, ErrInvalidTransferCommand
		}

		switch {
		case entry.Side == domain.Debit &&
			entry.AccountID == source.ID:
			if seenDebit {
				return domain.Transaction{}, ErrInvalidTransferCommand
			}
			seenDebit = true

		case entry.Side == domain.Credit &&
			entry.AccountID == destination.ID:
			if seenCredit {
				return domain.Transaction{}, ErrInvalidTransferCommand
			}
			seenCredit = true

		default:
			return domain.Transaction{}, ErrInvalidTransferCommand
		}
	}

	if !seenDebit || !seenCredit {
		return domain.Transaction{}, ErrInvalidTransferCommand
	}

	// Guard against integer overflow when crediting the destination.
	if destination.Balance.AmountMinor >
		math.MaxInt64-command.Amount.AmountMinor {
		return domain.Transaction{}, ErrDestinationBalanceOverflow
	}

	// Prepare the new state before changing the stored wallets.
	now := time.Now().UTC()

	source.Balance.AmountMinor -= command.Amount.AmountMinor
	source.UpdatedAt = now

	destination.Balance.AmountMinor += command.Amount.AmountMinor
	destination.UpdatedAt = now

	transaction := domain.Transaction{
		ID:             command.TransactionID,
		Reference:      command.Reference,
		IdempotencyKey: command.Idempotency.Key,
		Type:           domain.TransactionTransfer,
		Status:         domain.StatusSucceeded,
		Amount:         command.Amount,
		SourceWalletID: source.ID,
		DestWalletID:   destination.ID,
		Description:    strings.TrimSpace(command.Description),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := transaction.Validate(); err != nil {
		return domain.Transaction{}, err
	}

	// Store all resulting state while holding the same lock.
	// No other operation using this repository can observe a partial transfer.
	r.wallets[source.ID] = source
	r.wallets[destination.ID] = destination
	r.transactions[transaction.ID] = transaction
	r.ledgerEntries[transaction.ID] = entries

	record := command.Idempotency
	record.Status = domain.IdempotencyCompleted
	record.TransactionID = transaction.ID
	r.idempotency[record.Key] = record

	return transaction, nil
}
