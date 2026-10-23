package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"financial-engine/internal/domain"
	"financial-engine/internal/ports"
)

var ErrInvalidTransfer = errors.New(
	"invalid transfer request",
)

type TransferService struct {
	repository ports.Repository
}

func NewTransferService(
	repository ports.Repository,
) *TransferService {
	return &TransferService{
		repository: repository,
	}
}

// TransferRequest describes what the caller wants to do.
// It deliberately does not accept caller-supplied balances or ledger entries.
type TransferRequest struct {
	SourceWalletID      string
	DestinationWalletID string
	Amount              domain.Money
	IdempotencyKey      string
	Description         string
}

func (s *TransferService) Transfer(
	ctx context.Context,
	request TransferRequest,
) (domain.Transaction, error) {
	if s == nil || s.repository == nil {
		return domain.Transaction{}, errors.New(
			"financial repository is not configured",
		)
	}

	if err := ctx.Err(); err != nil {
		return domain.Transaction{}, err
	}

	sourceID := strings.TrimSpace(request.SourceWalletID)
	destinationID := strings.TrimSpace(
		request.DestinationWalletID,
	)
	key := strings.TrimSpace(request.IdempotencyKey)
	description := strings.TrimSpace(request.Description)

	if sourceID == "" ||
		destinationID == "" ||
		key == "" ||
		sourceID == destinationID {
		return domain.Transaction{}, ErrInvalidTransfer
	}

	if err := request.Amount.Validate(); err != nil {
		return domain.Transaction{}, err
	}

	now := time.Now().UTC()

	transactionID, err := generateID("txn")
	if err != nil {
		return domain.Transaction{}, err
	}

	reference, err := generateID("trf")
	if err != nil {
		return domain.Transaction{}, err
	}

	// Hash the business-relevant request fields.
	// Reusing a key for a different request must be rejected
	// by the repository rather than treated as the same transfer.
	canonicalRequest := struct {
		SourceWalletID      string `json:"source_wallet_id"`
		DestinationWalletID string `json:"destination_wallet_id"`
		Currency            string `json:"currency"`
		AmountMinor         int64  `json:"amount_minor"`
		Description         string `json:"description"`
	}{
		SourceWalletID:      sourceID,
		DestinationWalletID: destinationID,
		Currency:            strings.ToUpper(request.Amount.Currency),
		AmountMinor:         request.Amount.AmountMinor,
		Description:         description,
	}

	requestBytes, err := json.Marshal(canonicalRequest)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf(
			"encode transfer request: %w",
			err,
		)
	}

	requestHash := sha256.Sum256(requestBytes)

	idempotency := domain.IdempotencyRecord{
		Key:         key,
		RequestHash: hex.EncodeToString(requestHash[:]),
		Status:      domain.IdempotencyProcessing,
		CreatedAt:   now,
		ExpiresAt:   now.Add(24 * time.Hour),
	}

	if err := idempotency.Validate(); err != nil {
		return domain.Transaction{}, err
	}

	transaction := domain.Transaction{
		ID:             transactionID,
		Reference:      reference,
		IdempotencyKey: key,
		Type:           domain.TransactionTransfer,
		Status:         domain.StatusPending,
		Amount:         request.Amount,
		SourceWalletID: sourceID,
		DestWalletID:   destinationID,
		Description:    description,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := transaction.Validate(); err != nil {
		return domain.Transaction{}, err
	}

	// Build and validate the balanced ledger entries.
	entries, err := BuildTransferEntries(
		transaction.ID,
		sourceID,
		destinationID,
		request.Amount,
	)
	if err != nil {
		return domain.Transaction{}, err
	}

	// Pass the transfer intent and entries to the repository.
	// The repository must re-check the real wallet state and
	// execute all database changes in one atomic transaction.
	command := ports.TransferCommand{
		TransactionID:  transaction.ID,
		Reference:      transaction.Reference,
		SourceWalletID: sourceID,
		DestWalletID:   destinationID,
		Amount:         request.Amount,
		Entries:        entries,
		Idempotency:    idempotency,
	}

	return s.repository.TransferTx(ctx, command)
}

func generateID(prefix string) (string, error) {
	var bytes [16]byte

	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf(
			"generate %s ID: %w",
			prefix,
			err,
		)
	}

	return prefix + "_" + hex.EncodeToString(bytes[:]), nil
}
