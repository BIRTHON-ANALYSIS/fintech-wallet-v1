package service

import (
	"context"
	"errors"

	"financial-engine/internal/domain"
	"financial-engine/internal/ports"
)

var ErrInvalidTransfer = errors.New("invalid transfer request")

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

type TransferRequest struct {
	Transaction domain.Transaction
	Source      domain.Wallet
	Destination domain.Wallet
	Amount      domain.Money
	Entries     []domain.LedgerEntry
	Idempotency domain.IdempotencyRecord
}

func (s *TransferService) Transfer(
	ctx context.Context,
	request TransferRequest,
) (domain.Transaction, error) {
	if s.repository == nil {
		return domain.Transaction{}, errors.New(
			"financial repository is not configured",
		)
	}

	if err := domain.ValidateTransfer(
		request.Source,
		request.Destination,
		request.Amount,
	); err != nil {
		return domain.Transaction{}, err
	}

	if request.Transaction.Type != domain.TransactionTransfer {
		return domain.Transaction{}, ErrInvalidTransfer
	}

	if request.Transaction.Amount != request.Amount {
		return domain.Transaction{}, ErrInvalidTransfer
	}

	if request.Transaction.SourceWalletID != request.Source.ID ||
		request.Transaction.DestWalletID != request.Destination.ID {
		return domain.Transaction{}, ErrInvalidTransfer
	}

	if err := request.Transaction.Validate(); err != nil {
		return domain.Transaction{}, err
	}

	if err := request.Idempotency.Validate(); err != nil {
		return domain.Transaction{}, err
	}

	if err := domain.ValidateBalancedEntries(request.Entries); err != nil {
		return domain.Transaction{}, err
	}

	command := ports.TransferCommand{
		Transaction: request.Transaction,
		Source:      request.Source,
		Destination: request.Destination,
		Entries:     request.Entries,
		Idempotency: request.Idempotency,
	}

	return s.repository.TransferTx(ctx, command)
}
