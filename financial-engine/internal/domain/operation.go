package domain

import (
	"errors"
	"strings"
)

var (
	ErrWalletNotActive     = errors.New("wallet is not active")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrCurrencyMismatch    = errors.New("currency mismatch")
	ErrDuplicateRequest    = errors.New("duplicate request")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different request")
)

func ValidateTransfer(
	source Wallet,
	destination Wallet,
	amount Money,
) error {
	if err := source.Validate(); err != nil {
		return err
	}

	if err := destination.Validate(); err != nil {
		return err
	}

	if err := amount.Validate(); err != nil {
		return err
	}

	if source.ID == destination.ID {
		return errors.New("source and destination wallets must differ")
	}

	if source.Status != WalletActive || destination.Status != WalletActive {
		return ErrWalletNotActive
	}

	if !strings.EqualFold(source.Currency, destination.Currency) ||
		!strings.EqualFold(source.Currency, amount.Currency) {
		return ErrCurrencyMismatch
	}

	if source.Balance.AmountMinor < amount.AmountMinor {
		return ErrInsufficientFunds
	}

	return nil
}
