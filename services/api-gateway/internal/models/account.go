package models

import (
	"time"

	"github.com/google/uuid"
)

type BalanceOperation struct {
	ID            uuid.UUID `json:"id"`
	OperationID   uuid.UUID `json:"operation_id"`
	AccountID     string    `json:"account_id"`
	Currency      string    `json:"currency"`
	Amount        float64   `json:"amount"`
	NewBalance    float64   `json:"new_balance"`
	OperationType string    `json:"operation_type"`
	Status        string    `json:"status"`
	TransactionID uuid.UUID `json:"transaction_id,omitempty"`
	Description   string    `json:"description,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	ProcessedAt   time.Time `json:"processed_at,omitempty"`
}

type CurrencyAccount struct {
	ID            string    `db:"id" json:"id"`
	Name          string    `db:"account_name" json:"account_name"`
	UserID        uuid.UUID `db:"user_id" json:"user_id"`
	Currency      string    `db:"currency" json:"currency"`
	Balance       float64   `db:"balance" json:"balance"`
	BlockedAmount float64   `db:"blocked_amount" json:"blocked_amount"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

type CreateAccountRequest struct {
	Currency string `json:"currency"`
	Name     string `json:"account_name"`
}

type CreateAccountResponse struct {
	AccountID string `json:"account_id"`
}

type GetAccountRequest struct {
	AccountID string `json:"account_id"`
}

type GetUserAccountsRequest struct {
	UserID uuid.UUID `json:"user_id"`
}

type GetBalanceRequest struct {
	AccountID string `json:"account_id"`
}

type GetBalanceResponse struct {
	Balance float64 `json:"balance"`
}
