// Пакет models содержит различные dto'шки, модельки, константы
package models

import (
	"time"

	"github.com/google/uuid"
)

// Топики для событий:
const (
	TopicTransactionCommands string = "transaction.commands" // API-Gateway-service / Payment-Switch-service → Transaction-service
	TopicTransactionEvents   string = "transaction.events"   // Transaction → Analytics-service
	TopicAccountCommands     string = "account.commands"     // Transaction-service → Account-service
	TopicAccountEvents       string = "account.events"       // Account-service → Transaction-service
	TopicSwitchCommands      string = "switch.commands"      // Transaction-service → Payment-Switch-service
	TopicSwitchEvents        string = "switch.events"        // Payment-Switch-service → Transaction-service
)

// Статусы событий:
const (
	// EventStatusPending - событие в обработке
	EventStatusPending string = "pending"

	// EventStatusCompleted - событие завершено
	EventStatusCompleted string = "completed"
)

// Типы событий для Kafka
const (
	// EventTransactionCreate - событие: создание транзакции
	EventTransactionCreate string = "transaction.create"
	// EventTransactionCreated — событие: транзакция создана
	EventTransactionCreated string = "transaction.created"
	// EventTransactionCompleted — событие: транзакция успешно завершена
	EventTransactionCompleted string = "transaction.completed"
	// EventTransactionCancelled — событие: транзакция отменена
	EventTransactionCancelled string = "transaction.cancelled"
	// EventTransactionBlocked — событие: транзакция заблокирована, требуется ручное вмешательство
	EventTransactionBlocked string = "transaction.blocked"

	// EventFreezeRequest - событие: запрос на заморозку сердтв для списания
	EventFreezeRequest string = "freeze.request"
	// EventFreezeResponse - событие: ответ на запрос о заморозки средств для списания
	EventFreezeResponse string = "freeze.response"
	// EventUnfreezeRequest - событие: запрос на отмену заморзки средств
	EventUnfreezeRequest string = "unfreeze.request"
	// EventUnfreezeResponse - событие: ответ на зампрос об отмене заморозки средств
	EventUnfreezeResponse string = "unfreeze.response"

	// EventCaptureRequest - событие: запрос на списание из замороженных средств
	EventCaptureRequest string = "capture.request"
	// EventCaptureResponse - событие: ответ на запрос о списании средств из замороженных
	EventCaptureResponse string = "capture.response"

	// EventCreditRequest - событие: запрос на прямое пополнение средств
	EventCreditRequest string = "credit.request"
	// EventCreditResponse - событие: ответ на запрос о прямом пополнении средств
	EventCreditResponse string = "credit.response"
	// EventDebitRequest - событие: запрос на прямое списание средств
	EventDebitRequest string = "debit.request"
	// EventDebitResponse - событие: ответ на запрос о прямом списании средств
	EventDebitResponse string = "debit.response"

	// EventPaymentRequest - событие: запрос на отправку платежа во внешнюю систему
	EventPaymentRequest string = "payment.request"
	// EventPaymentResponse - событие: ответ внешней системы на запрос платежа
	EventPaymentResponse string = "payment.response"
	// EventCommitRequest - событие: запрос на подтверждение завершения операции во внешней системе
	EventCommitRequest string = "commit.request"
	// EventCommitResponse - событие: ответ внешней системы на подтверждение завершения
	EventCommitResponse string = "commit.response"
	// EventRollbackRequest - событие: запрос на отмену операции во внешней системе
	EventRollbackRequest string = "rollback.request"
	// EventRollbackResponse - событие: ответ внешней системы на отмену операции
	EventRollbackResponse string = "rollback.response"

	// EventBlockRequest - событие: запрос на блокировку счета
	EventBlockRequest string = "block.request"
	// EventBlockResponse - событие: ответ о запросе блокировки счета
	EventBlockResponse string = "block.response"
)

type Event struct {
	ID            uuid.UUID  `db:"id" json:"id"`
	TransactionID uuid.UUID  `db:"transaction_id" json:"transaction_id"`
	StepName      string     `db:"step_name" json:"step_name"`
	PartitionKey  string     `db:"partition_key" json:"partition_key"`
	Type          string     `db:"type" json:"type"`
	Status        string     `db:"status" json:"status"`
	Source        string     `db:"source" json:"source"`
	TraceID       string     `db:"trace_id" json:"trace_id"`
	SpanID        string     `db:"span_id" json:"span_id"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	ProcessedAt   *time.Time `db:"processed_at" json:"processed_at"`
	Payload       []byte     `db:"payload" json:"payload"`
}

type AccountCommandPayload struct {
	TransactionID string  `json:"transaction_id"`
	StepName      string  `json:"step_name"`
	AccountCode   string  `json:"code"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
}

type SwitchCommandPayload struct {
	TransactionID string            `json:"transaction_id"`
	StepName      string            `json:"step_name"`
	Amount        float64           `json:"amount"`
	Currency      string            `json:"currency"`
	Sender        map[string]string `json:"sender"`
	Recipient     map[string]string `json:"recipient"`
	Description   string            `json:"description,omitempty"`
}

type TransactionCreatedPayload struct {
	Type           string            `json:"type"`
	Amount         float64           `json:"amount"`
	Currency       string            `json:"currency"`
	IdempotencyKey string            `json:"idempotency_key"`
	Sender         map[string]string `json:"sender"`
	Recipient      map[string]string `json:"recipient"`
	Description    string            `json:"description,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

type TransactionCompletedPayload struct {
	CompletedAt time.Time `json:"completed_at"`
}

type TransactionCancelledPayload struct {
	Reason      string    `json:"reason"`
	CancelledAt time.Time `json:"cancelled_at"`
}

type ResponsePayload struct {
	Success bool   `json:"success"`
	Reason  string `json:"reason,omitempty"`
}

type UpdateAccountPayload struct {
	Code string `db:"code" json:"сode" validate:"required"`
}

type BuildEventRequest struct {
	Transaction  *Transaction
	StepName     string
	PartitionKey string
	EventType    string
	TraceID      string `db:"trace_id" json:"trace_id"`
	SpanID       string `db:"span_id" json:"span_id"`
	Payload      any
}

type InsertEventRequest struct {
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
	EventTransactionCreated:   TopicTransactionEvents,
	EventTransactionCompleted: TopicTransactionEvents,
	EventTransactionCancelled: TopicTransactionEvents,
	EventTransactionBlocked:   TopicTransactionEvents,

	EventFreezeRequest:    TopicAccountCommands,
	EventFreezeResponse:   TopicAccountEvents,
	EventUnfreezeRequest:  TopicAccountCommands,
	EventUnfreezeResponse: TopicAccountEvents,

	EventCaptureRequest:  TopicAccountCommands,
	EventCaptureResponse: TopicAccountEvents,

	EventCreditRequest:  TopicAccountCommands,
	EventCreditResponse: TopicAccountEvents,
	EventDebitRequest:   TopicAccountCommands,
	EventDebitResponse:  TopicAccountEvents,

	EventPaymentRequest:   TopicSwitchCommands,
	EventPaymentResponse:  TopicSwitchEvents,
	EventCommitRequest:    TopicSwitchCommands,
	EventCommitResponse:   TopicSwitchEvents,
	EventRollbackRequest:  TopicSwitchCommands,
	EventRollbackResponse: TopicSwitchEvents,

	EventBlockRequest:  TopicAccountCommands,
	EventBlockResponse: TopicAccountEvents,
}
