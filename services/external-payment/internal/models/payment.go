package models

import (
	"time"

	"github.com/google/uuid"
)

const Source = "external-payment-service"

// Топики
const (
	TopicExternalTransaction = "external-transaction-commands"
	TopicTransaction         = "transaction-commands"
)

// Типы событий, которые слушает сервис
const (
	EventExternalRequest  = "external.request"
	EventExternalCommit   = "external.commit"
	EventExternalRollback = "external.rollback"
)

// Типы событий, которые отправляет сервис
const (
	EventExternalResponse   = "external.response"
	EventTransactionRequest = "transaction.request"
)

type Event struct {
	ID            uuid.UUID `db:"id" json:"id"`
	TransactionID uuid.UUID `db:"transaction_id" json:"transaction_id"`
	PartitionKey  string    `db:"partition_key" json:"partition_key"`
	Type          string    `db:"type" json:"type"`
	Status        string    `db:"status" json:"status"`
	Source        string    `db:"source" json:"source"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	Payload       []byte    `db:"payload" json:"payload"`
}

// BalanceRequestPayload — для инициирования входящего платежа
type BalanceRequestPayload struct {
	AccountCode string  `json:"code"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
}

// BalanceResponsePayload — ответ на операции
type BalanceResponsePayload struct {
	Success bool   `json:"success"`
	Reason  string `json:"reason,omitempty"`
}

// CreateExternalPaymentRequest запрос на создание внешнего платежа
type CreateExternalPaymentRequest struct {
	SenderAccountCode    string  `json:"sender_account_code" form:"sender_account_code"`
	RecipientAccountCode string  `json:"recipient_account_code" form:"recipient_account_code"`
	Amount               float64 `json:"amount" form:"amount"`
	Currency             string  `json:"currency" form:"currency"`
	Description          string  `json:"description,omitempty" form:"description"`
}

// CreateExternalPaymentResponse ответ на создание внешнего платежа
type CreateExternalPaymentResponse struct {
	TransactionID uuid.UUID `json:"transaction_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status        string    `json:"status" example:"pending"`
	Message       string    `json:"message" example:"External payment initiated"`
}
