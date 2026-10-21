package domain

import (
	"errors"
	"strings"
	"time"
)

type TransactionType string

const (
	TransactionDeposit     TransactionType = "deposit"
	TransactionWithdrawal  TransactionType = "withdrawal"
	TransactionTransfer    TransactionType = "transfer"
	TransactionPayment     TransactionType = "payment"
	TransactionRefund      TransactionType = "refund"
	TransactionExchange    TransactionType = "exchange"
	TransactionBillPayment TransactionType = "bill_payment"
)

type TransactionStatus string

const (
	StatusPending    TransactionStatus = "pending"
	StatusProcessing TransactionStatus = "processing"
	StatusSucceeded  TransactionStatus = "succeeded"
	StatusFailed     TransactionStatus = "failed"
	StatusReversed   TransactionStatus = "reversed"
)

type Transaction struct {
	ID             string            `json:"id"`
	Reference      string            `json:"reference"`
	IdempotencyKey string            `json:"-"`
	Type           TransactionType   `json:"type"`
	Status         TransactionStatus `json:"status"`
	Amount         Money             `json:"amount"`
	SourceWalletID string            `json:"source_wallet_id,omitempty"`
	DestWalletID   string            `json:"destination_wallet_id,omitempty"`
	Description    string            `json:"description,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

func (t Transaction) Validate() error {
	if strings.TrimSpace(t.ID) == "" {
		return errors.New("transaction ID is required")
	}

	if strings.TrimSpace(t.Reference) == "" {
		return errors.New("transaction reference is required")
	}

	if err := t.Amount.Validate(); err != nil {
		return err
	}

	switch t.Type {
	case TransactionDeposit,
		TransactionWithdrawal,
		TransactionTransfer,
		TransactionPayment,
		TransactionRefund,
		TransactionExchange,
		TransactionBillPayment:
	default:
		return errors.New("invalid transaction type")
	}

	switch t.Status {
	case StatusPending, StatusProcessing, StatusSucceeded,
		StatusFailed, StatusReversed:
	default:
		return errors.New("invalid transaction status")
	}

	return nil
}
