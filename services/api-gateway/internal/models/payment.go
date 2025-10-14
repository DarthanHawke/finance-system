package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	PaymentStatusPending   string = "pending"
	PaymentStatusCompleted string = "completed"
	PaymentStatusCancelled string = "cancelled"
	PaymentStatusRefunded  string = "refunded"
)

const (
	CurrencyUSD string = "USD"
	CurrencyEUR string = "EUR"
	CurrencyRUB string = "RUB"
)

// структура платежа
type Payment struct {
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

// PaymentRequest - DTO для платежа
type PaymentRequest struct {
	PaymentID uuid.UUID `json:"paymentID" validate:"required"`
}

// TransferRequest - DTO для создания платежа
type TransferRequest struct {
	Sender      string  `json:"sender,omitempty" validate:"max=255"`
	Receiver    string  `json:"receiver,omitempty" validate:"max=255"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Description string  `json:"description,omitempty" validate:"max=255"`
}

// UpdateStatusRequest - DTO для обновления статуса платежа
type UpdateStatusRequest struct {
	PaymentID uuid.UUID `json:"paymentID" validate:"required"`
	Status    string    `json:"status" validate:"required,oneof=completed refunded"`
}

type DepositRequest struct {
	AccountID   string  `json:"account_id"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description,omitempty"`
}

type DepositResponse struct {
	PaymentID uuid.UUID `json:"payment_id"`
}

type ConvertCurrencyRequest struct {
	FromAccountID string  `json:"from_account_id"`
	ToAccountID   string  `json:"to_account_id"`
	Amount        float64 `json:"amount"`
	Description   string  `json:"description,omitempty"`
}

type ConvertCurrencyResponse struct {
	OperationID uuid.UUID `json:"operation_id"`
}

type OperationHistoryRequest struct {
	AccountID string `json:"account_id"`
	Limit     int    `json:"limit"`
	Offset    int    `json:"offset"`
}

type UpdateCurrencyRateRequest struct {
	FromCurrency string  `json:"from_currency"`
	ToCurrency   string  `json:"to_currency"`
	Rate         float64 `json:"rate"`
}
