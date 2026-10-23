package service

import (
	"errors"
	"strings"
	"time"

	"financial-engine/internal/domain"
)

var ErrInvalidLedgerCommand = errors.New(
	"invalid ledger command",
)

// BuildTransferEntries creates the debit and credit entries
// for a wallet-to-wallet transfer.
//
// It validates the entries but does not save them or change
// wallet balances.
func BuildTransferEntries(
	transactionID string,
	sourceWalletID string,
	destinationWalletID string,
	amount domain.Money,
) ([]domain.LedgerEntry, error) {
	transactionID = strings.TrimSpace(transactionID)
	sourceWalletID = strings.TrimSpace(sourceWalletID)
	destinationWalletID = strings.TrimSpace(destinationWalletID)

	if transactionID == "" ||
		sourceWalletID == "" ||
		destinationWalletID == "" ||
		sourceWalletID == destinationWalletID {
		return nil, ErrInvalidLedgerCommand
	}

	if err := amount.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	currency := strings.ToUpper(amount.Currency)

	entries := []domain.LedgerEntry{
		{
			ID:            "led_" + transactionID + "_debit",
			TransactionID: transactionID,
			AccountID:     sourceWalletID,
			Side:          domain.Debit,
			AmountMinor:   amount.AmountMinor,
			Currency:      currency,
			CreatedAt:     now,
		},
		{
			ID:            "led_" + transactionID + "_credit",
			TransactionID: transactionID,
			AccountID:     destinationWalletID,
			Side:          domain.Credit,
			AmountMinor:   amount.AmountMinor,
			Currency:      currency,
			CreatedAt:     now,
		},
	}

	if err := domain.ValidateBalancedEntries(entries); err != nil {
		return nil, err
	}

	return entries, nil
}
