package models

import (
	"time"

	"github.com/google/uuid"
)

const Source = "api-gateway-service"

// Топики для событий:
const (
	TopicTransaction = "transaction-commands"
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
const (
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
