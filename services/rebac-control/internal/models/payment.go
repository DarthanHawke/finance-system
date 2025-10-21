package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	TransactionStatusPending   string = "pending"
	TransactionStatusCompleted string = "completed"
	TransactionStatusCancelled string = "cancelled"
	TransactionStatusRefunded  string = "refunded"
)

const (
	CurrencyUSD string = "USD"
	CurrencyEUR string = "EUR"
	CurrencyRUB string = "RUB"
)

// структура платежа
type Transaction struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Sender      string    `json:"sender" db:"sender"`
	Receiver    string    `json:"receiver" db:"receiver"`
	Amount      float64   `json:"amount" db:"amount"`
	Currency    string    `json:"currency" db:"currency"`
	Status      string    `json:"status" db:"status"`
	Description string    `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

// TransactionCreateRequest - DTO для создания платежа
type TransactionCreateRequest struct {
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Currency    string  `json:"currency" validate:"required,oneof=USD EUR RUB"`
	Description string  `json:"description,omitempty" validate:"max=255"`
}
