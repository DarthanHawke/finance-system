package models

import (
	"time"

	"github.com/google/uuid"
)

type BalanceOperation struct {
	ID            uuid.UUID `json:"id" db:"id"`
	OperationID   uuid.UUID `json:"operation_id" db:"operation_id"`
	AccountCode   string    `json:"account_code" db:"account_code"`
	UserID        uuid.UUID `json:"user_id,omitempty" db:"user_id" `
	Currency      string    `json:"currency" db:"currency"`
	Amount        float64   `json:"amount" db:"amount"`
	NewBalance    float64   `json:"new_balance" db:"new_balance"`
	OperationType string    `json:"operation_type" db:"operation_type"`
	Status        string    `json:"status" db:"status"`
	PaymentID     uuid.UUID `json:"payment_id,omitempty" db:"payment_id"`
	Description   string    `json:"description,omitempty" db:"description"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	ProcessedAt   time.Time `json:"processed_at,omitempty" db:"processed_at"`
}

type CurrencyAccount struct {
	AccountCode   string    `db:"account_code" json:"account_code"`
	Name          string    `db:"account_name" json:"account_name"`
	UserID        uuid.UUID `db:"user_id" json:"user_id"`
	Currency      string    `db:"currency" json:"currency"`
	Balance       float64   `db:"balance" json:"balance"`
	BlockedAmount float64   `db:"blocked_amount" json:"blocked_amount,omitempty"`
	CreatedAt     time.Time `db:"created_at" json:"created_at,omitempty"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at,omitempty"`
}
