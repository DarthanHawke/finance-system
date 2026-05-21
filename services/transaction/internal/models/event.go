// Пакет models содержит различные dto'шки, модельки, константы
package models

import (
	"time"

	"github.com/google/uuid"
)

// Топики для событий:
const (
	TopicAccount             = "account-commands"
	TopicExternalTransaction = "external-transaction-commands"
	TopicTransaction         = "transaction-commands"
)

// Статусы событий:
const (
	// EventStatusPending - событие в обработке
	EventStatusPending string = "pending"

	// EventStatusCompleted - событие завершено
	EventStatusCompleted string = "completed"

	// EventStatusFailed - событие не может быть отправлено(первышен max Retry)
	EventStatusFailed string = "failed"
)

// Типы событий для Kafka
const (
	EventTransactionRequest = "transaction.request"

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

	EventExternalRequest  = "external.request"
	EventExternalResponse = "external.response"
	EventExternalCommit   = "external.commit"
	EventExternalRollback = "external.rollback"
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

type UpdateAccountPayload struct {
	Code string `db:"code" json:"сode" validate:"required"`
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

type UpdateAccountRequest struct {
	TransactionID uuid.UUID `db:"transaction_id" json:"transaction_id"`
	Code          string    `db:"code" json:"сode" validate:"required"`
}

type UpdateEventStatusRequest struct {
	ID     uuid.UUID `db:"id" json:"id"`
	Status string    `db:"status" json:"status" validate:"required,oneof=completed pending"`
}

var TopicMap = map[string]string{
	EventFreezeRequest:    TopicAccount,
	EventFreezeResponse:   TopicAccount,
	EventUnfreezeRequest:  TopicAccount,
	EventUnfreezeResponse: TopicAccount,

	EventReserveRequest:    TopicAccount,
	EventReserveResponse:   TopicAccount,
	EventUnreserveRequest:  TopicAccount,
	EventUnreserveResponse: TopicAccount,

	EventDepositRequest:  TopicAccount,
	EventDepositResponse: TopicAccount,

	EventWithdrawRequest:  TopicAccount,
	EventWithdrawResponse: TopicAccount,
	EventRefundRequest:    TopicAccount,
	EventRefundResponse:   TopicAccount,

	EventBlockRequest:  TopicAccount,
	EventBlockResponse: TopicAccount,

	EventExternalRequest:  TopicExternalTransaction,
	EventExternalResponse: TopicExternalTransaction,
	EventExternalCommit:   TopicExternalTransaction,
	EventExternalRollback: TopicExternalTransaction,
}
