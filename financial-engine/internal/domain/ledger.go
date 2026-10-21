package domain

import (
	"errors"
	"strings"
	"time"
)

type EntrySide string

const (
	Debit  EntrySide = "debit"
	Credit EntrySide = "credit"
)

type LedgerEntry struct {
	ID            string    `json:"id"`
	TransactionID string    `json:"transaction_id"`
	AccountID     string    `json:"account_id"`
	Side          EntrySide `json:"side"`
	AmountMinor   int64     `json:"amount_minor"`
	Currency      string    `json:"currency"`
	CreatedAt     time.Time `json:"created_at"`
}

func (e LedgerEntry) Validate() error {
	if strings.TrimSpace(e.ID) == "" {
		return errors.New("ledger entry ID is required")
	}

	if strings.TrimSpace(e.TransactionID) == "" {
		return errors.New("transaction ID is required")
	}

	if strings.TrimSpace(e.AccountID) == "" {
		return errors.New("account ID is required")
	}

	if e.AmountMinor <= 0 {
		return errors.New("ledger entry amount must be positive")
	}

	switch e.Side {
	case Debit, Credit:
	default:
		return errors.New("invalid ledger entry side")
	}

	switch strings.ToUpper(e.Currency) {
	case "NGN", "USD", "EUR":
	default:
		return ErrInvalidCurrency
	}

	return nil
}

// ValidateBalancedEntries ensures debits equal credits
// for a single-currency transaction.
func ValidateBalancedEntries(entries []LedgerEntry) error {
	if len(entries) < 2 {
		return errors.New("a ledger transaction requires at least two entries")
	}

	var debitTotal int64
	var creditTotal int64
	currency := entries[0].Currency
	transactionID := entries[0].TransactionID

	for _, entry := range entries {
		if err := entry.Validate(); err != nil {
			return err
		}

		if !strings.EqualFold(entry.Currency, currency) {
			return errors.New("ledger entries must use the same currency")
		}

		if entry.TransactionID != transactionID {
			return errors.New("ledger entries must belong to the same transaction")
		}

		switch entry.Side {
		case Debit:
			if debitTotal > int64(^uint64(0)>>1)-entry.AmountMinor {
				return errors.New("debit total overflow")
			}
			debitTotal += entry.AmountMinor

		case Credit:
			if creditTotal > int64(^uint64(0)>>1)-entry.AmountMinor {
				return errors.New("credit total overflow")
			}
			creditTotal += entry.AmountMinor
		}
	}

	if debitTotal != creditTotal {
		return errors.New("ledger transaction is unbalanced")
	}

	return nil
}
