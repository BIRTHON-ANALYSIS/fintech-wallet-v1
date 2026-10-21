package domain

import (
	"errors"
	"strings"
	"time"
)

type WalletStatus string

const (
	WalletActive WalletStatus = "active"
	WalletFrozen WalletStatus = "frozen"
	WalletClosed WalletStatus = "closed"
)

type Wallet struct {
	ID        string       `json:"id"`
	UserID    string       `json:"user_id"`
	Currency  string       `json:"currency"`
	Balance   Money        `json:"balance"`
	Status    WalletStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func (w Wallet) Validate() error {
	if strings.TrimSpace(w.ID) == "" {
		return errors.New("wallet ID is required")
	}

	if strings.TrimSpace(w.UserID) == "" {
		return errors.New("user ID is required")
	}

	if w.Balance.AmountMinor < 0 {
		return errors.New("wallet balance cannot be negative")
	}

	if !strings.EqualFold(w.Currency, w.Balance.Currency) {
		return errors.New("wallet and balance currencies must match")
	}

	switch w.Status {
	case WalletActive, WalletFrozen, WalletClosed:
		return nil
	default:
		return errors.New("invalid wallet status")
	}
}
