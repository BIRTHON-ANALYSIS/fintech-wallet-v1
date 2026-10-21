package domain

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidCurrency = errors.New("invalid currency")
	ErrInvalidAmount   = errors.New("amount must be positive")
)

// Money stores amounts in the currency's minor unit.
// Example: NGN 1500.50 is stored as 150050 kobo.
type Money struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

func NewMoney(amountMinor int64, currency string) (Money, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))

	switch currency {
	case "NGN", "USD", "EUR":
	default:
		return Money{}, ErrInvalidCurrency
	}

	if amountMinor <= 0 {
		return Money{}, ErrInvalidAmount
	}

	return Money{
		AmountMinor: amountMinor,
		Currency:    currency,
	}, nil
}

func (m Money) Validate() error {
	if m.AmountMinor <= 0 {
		return ErrInvalidAmount
	}

	switch strings.ToUpper(m.Currency) {
	case "NGN", "USD", "EUR":
		return nil
	default:
		return ErrInvalidCurrency
	}
}

func (m Money) Add(other Money) (Money, error) {
	if !strings.EqualFold(m.Currency, other.Currency) {
		return Money{}, fmt.Errorf(
			"cannot add %s to %s",
			m.Currency,
			other.Currency,
		)
	}

	if other.AmountMinor > 0 &&
		m.AmountMinor > int64(^uint64(0)>>1)-other.AmountMinor {
		return Money{}, errors.New("money amount overflow")
	}

	return Money{
		AmountMinor: m.AmountMinor + other.AmountMinor,
		Currency:    strings.ToUpper(m.Currency),
	}, nil
}
