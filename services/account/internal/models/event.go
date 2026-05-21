// Пакет models содержит различные dto'шки, модельки, константы
package models

import (
	"time"

	"github.com/google/uuid"
)

// Топики для событий:
const (
	TopicAccount     = "account-commands"
	TopicTransaction = "transaction-commands"
)

// Статусы событий:
const (
	// TransactionStatusPending - платеж в обработке
	EventStatusPending string = "pending"

	// TransactionStatusCompleted - платеж завершён
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
	EventRefundRequest    = "refund.request"
	EventRefundResponse   = "refund.response"

	EventBlockRequest  = "block.request"
	EventBlockResponse = "block.response"
)

type Event struct {
	ID            uuid.UUID `db:"id" json:"id"`
	TransactionID uuid.UUID `db:"transaction_id" json:"transaction_id"`
	PartitionKey  string    `db:"partition_key" json:"partition_key"`
	Type          string    `db:"type" json:"type"`
	Status        string    `db:"status" json:"status"`
	Source        string    `db:"source" json:"source"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	ProcessedAt   time.Time `db:"processed_at" json:"processed_at"`
	Payload       []byte    `db:"payload" json:"payload"`
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

var TopicMap = map[string]string{
	EventFreezeRequest:    TopicTransaction,
	EventFreezeResponse:   TopicTransaction,
	EventUnfreezeRequest:  TopicTransaction,
	EventUnfreezeResponse: TopicTransaction,

	EventReserveRequest:    TopicTransaction,
	EventReserveResponse:   TopicTransaction,
	EventUnreserveRequest:  TopicTransaction,
	EventUnreserveResponse: TopicTransaction,

	EventDepositRequest:  TopicTransaction,
	EventDepositResponse: TopicTransaction,

	EventWithdrawRequest:  TopicTransaction,
	EventWithdrawResponse: TopicTransaction,
	EventRefundRequest:    TopicTransaction,
	EventRefundResponse:   TopicTransaction,

	EventBlockRequest:  TopicTransaction,
	EventBlockResponse: TopicTransaction,
}
