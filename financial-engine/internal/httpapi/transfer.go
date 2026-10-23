package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"financial-engine/internal/domain"
	"financial-engine/internal/ports"
	"financial-engine/internal/service"
)

type TransferHandler struct {
	transfers *service.TransferService
}

func NewTransferHandler(transfers *service.TransferService) *TransferHandler {
	return &TransferHandler{transfers: transfers}
}

type transferRequest struct {
	SourceWalletID      string `json:"source_wallet_id"`
	DestinationWalletID string `json:"destination_wallet_id"`
	AmountMinor         int64  `json:"amount_minor"`
	Currency            string `json:"currency"`
	IdempotencyKey      string `json:"idempotency_key"`
	Description         string `json:"description"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *TransferHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Error: "method not allowed",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input transferRequest
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid request body",
		})
		return
	}

	// Reject multiple JSON values in one request body.
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "request must contain a single JSON object",
		})
		return
	}

	input.SourceWalletID = strings.TrimSpace(input.SourceWalletID)
	input.DestinationWalletID = strings.TrimSpace(input.DestinationWalletID)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	input.Description = strings.TrimSpace(input.Description)

	if input.SourceWalletID == "" ||
		input.DestinationWalletID == "" ||
		input.IdempotencyKey == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "source_wallet_id, destination_wallet_id and idempotency_key are required",
		})
		return
	}

	if input.AmountMinor <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "amount_minor must be greater than zero",
		})
		return
	}

	money := domain.Money{
		AmountMinor: input.AmountMinor,
		Currency:    input.Currency,
	}

	if err := money.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid amount or currency",
		})
		return
	}

	result, err := h.transfers.Transfer(r.Context(), service.TransferRequest{
		SourceWalletID:      input.SourceWalletID,
		DestinationWalletID: input.DestinationWalletID,
		Amount:              money,
		IdempotencyKey:      input.IdempotencyKey,
		Description:         input.Description,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInsufficientFunds),
			errors.Is(err, ports.ErrInsufficientFunds):
			writeJSON(w, http.StatusUnprocessableEntity, errorResponse{
				Error: "insufficient funds",
			})

		case errors.Is(err, domain.ErrWalletNotActive),
			errors.Is(err, ports.ErrWalletNotActive):
			writeJSON(w, http.StatusUnprocessableEntity, errorResponse{
				Error: "source or destination wallet is not active",
			})

		case errors.Is(err, domain.ErrCurrencyMismatch),
			errors.Is(err, ports.ErrCurrencyMismatch):
			writeJSON(w, http.StatusUnprocessableEntity, errorResponse{
				Error: "wallet currency mismatch",
			})

		case errors.Is(err, domain.ErrIdempotencyConflict),
			errors.Is(err, ports.ErrIdempotencyConflict):
			writeJSON(w, http.StatusConflict, errorResponse{
				Error: "idempotency key was already used for a different request",
			})

		case errors.Is(err, domain.ErrDuplicateRequest):
			writeJSON(w, http.StatusConflict, errorResponse{
				Error: "request is already being processed",
			})

		default:
			slog.Error("transfer failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Error: "transfer could not be completed",
			})
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}
