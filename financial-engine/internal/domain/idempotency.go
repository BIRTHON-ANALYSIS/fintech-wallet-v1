package domain

import (
	"errors"
	"strings"
	"time"
)

type IdempotencyRecord struct {
	Key           string    `json:"-"`
	RequestHash   string    `json:"-"`
	TransactionID string    `json:"transaction_id,omitempty"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

const (
	IdempotencyProcessing = "processing"
	IdempotencyCompleted  = "completed"
	IdempotencyFailed     = "failed"
)

func (r IdempotencyRecord) Validate() error {
	if strings.TrimSpace(r.Key) == "" {
		return errors.New("idempotency key is required")
	}

	if strings.TrimSpace(r.RequestHash) == "" {
		return errors.New("request hash is required")
	}

	switch r.Status {
	case IdempotencyProcessing, IdempotencyCompleted, IdempotencyFailed:
	default:
		return errors.New("invalid idempotency status")
	}

	if !r.ExpiresAt.IsZero() && r.ExpiresAt.Before(r.CreatedAt) {
		return errors.New("idempotency expiry cannot precede creation")
	}

	return nil
}
