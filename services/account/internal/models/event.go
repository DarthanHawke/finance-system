// Пакет models содержит различные dto'шки, модельки, константы
package models

import (
	"time"

	"github.com/google/uuid"
)

// Топики для событий:
const (
	TopicAccount = "account-commands"
	TopicPayment = "payment-commands"
)

// Статусы событий:
const (
	// PaymentStatusPending - платеж в обработке
	EventStatusPending string = "pending"

	// PaymentStatusCompleted - платеж завершён
	EventStatusCompleted string = "completed"
)

// Типы событий для Kafka
const (
	EventFreezeRequest    = "freeze.request"
	EventFreezeResponse   = "freeze.response"
	EventUnfreezeRequest  = "unfreeze.request"
	EventUnfreezeResponse = "unfreeze.response"

	EventReserveRequest    = "reserve.request"
	EventReserveResponse   = "reserve.response"
	EventUnreserveRequest  = "unreserve.request"
	EventUnreserveResponse = "unreserve.response"

	EventDepositRequest  = "deposit.request"
	EventDepositResponse = "deposit.response"

	EventWithdrawRequest  = "withdraw.request"
	EventWithdrawResponse = "withdraw.response"
)

type Event struct {
	ID        uuid.UUID `db:"id" json:"id"`
	PaymentID uuid.UUID `db:"payment_id" json:"payment_id"`
	Type      string    `db:"type" json:"type"`
	Status    string    `db:"status" json:"status"`
	Source    string    `db:"source" json:"source"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Payload   []byte    `db:"payload" json:"payload"`
}

type BalanceRequestPayload struct {
	AccountCode string  `json:"code"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
}

type BalanceResponsePayload struct {
	Success bool   `json:"success"`
	Reason  string `json:"reason,omitempty"`
}

type CreateEventRequest struct {
	*Event
}

type GetEventRequest struct {
	Limit int `db:"limit" json:"limit" validate:"required"`
}

type GetEventResponse struct {
	Events []Event `json:"events"`
}

type UpdateEventStatusRequest struct {
	ID     uuid.UUID `db:"id" json:"id"`
	Status string    `json:"status" validate:"required,oneof=completed pending"`
}
