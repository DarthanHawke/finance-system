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
	TopicCurrencyCommands    string = "currency.commands"    // Transaction-service → Currency-service
	TopicCurrencyEvents      string = "currency.events"      // Currency-service → Transaction-service
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
	// EventTransactionCreated - событие: транзакция создана
	EventTransactionCreated string = "transaction.created"
	// EventTransactionCompleted - событие: транзакция успешно завершена
	EventTransactionCompleted string = "transaction.completed"
	// EventTransactionCancelled - событие: транзакция отменена
	EventTransactionCancelled string = "transaction.cancelled"
	// EventTransactionBlocked - событие: транзакция заблокирована, требуется ручное вмешательство
	EventTransactionBlocked string = "transaction.blocked"

	// EventFXRequest - событие: запрос на обмен валюты
	EventFXActivateRequest = "fx.activate.request"
	// EventFXResponse - событие: ответ на запрос об обмене валюты
	EventFXActivateResponse = "fx.activate.response"
	// EventFXRequest - событие: подтверждение обмена валюты
	EventFXCommitRequest = "fx.commit.request"
	// EventFXRequest - событие: ответ на подтверждение обмена валюты
	EventFXCommitResponse = "fx.commit.response"
	// EventFXRequest - событие: запрос на отмену обмена валюты
	EventFXCancelRequest = "fx.cancel.request"
	// EventFXRequest - событие: ответ на отмену обмена валюты
	EventFXCancelResponse = "fx.cancel.response"

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

// Event структура события
type Event struct {
	ID            uuid.UUID  `db:"id" json:"id"`
	TransactionID uuid.UUID  `db:"transaction_id" json:"transaction_id"`
	StepName      string     `db:"step_name" json:"step_name"`
	PartitionKey  string     `db:"partition_key" json:"partition_key"`
	Type          string     `db:"event_type" json:"event_type"`
	Status        string     `db:"event_status" json:"event_status"`
	Source        string     `db:"source" json:"source"`
	TraceID       string     `db:"trace_id" json:"trace_id"`
	SpanID        string     `db:"span_id" json:"span_id"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	ProcessedAt   *time.Time `db:"processed_at" json:"processed_at"`
	Payload       []byte     `db:"payload" json:"payload"`
}

// AccountCommandPayload тело сообщения для сервиса Account
type AccountCommandPayload struct {
	TransactionID string  `json:"transaction_id"`
	StepName      string  `json:"step_name"`
	AccountCode   string  `json:"code"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
}

// SwitchCommandPayload тело сообщения для сервиса Switch
type SwitchCommandPayload struct {
	TransactionID  string            `json:"transaction_id"`
	StepName       string            `json:"step_name"`
	Amount         float64           `json:"amount"`
	SourceCurrency string            `json:"source_currency"`
	TargetCurrency string            `json:"target_currency"`
	Sender         map[string]string `json:"sender"`
	Recipient      map[string]string `json:"recipient"`
	Description    string            `json:"description,omitempty"`
}

// CurrencyCommandPayload тело сообщения для сервиса Currency
type CurrencyCommandPayload struct {
	TransactionID string `json:"transaction_id"`
	StepName      string `json:"step_name"`
	FXDealID      string `json:"fx_deal_id"`
}

// TransactionCreatedPayload тело сообщения с инфой о транзакции для сервиса аналитики
type TransactionCreatedPayload struct {
	ID                  string            `json:"id"`
	ParentTransactionID string            `json:"parent_transaction_id,omitempty"`
	Type                string            `json:"transaction_type" validate:"required"`
	Status              string            `json:"transaction_status" validate:"required"`
	Amount              float64           `json:"amount" validate:"required"`
	SourceCurrency      string            `json:"source_currency" validate:"required"`
	TargetCurrency      string            `json:"target_currency" validate:"required"`
	Sender              map[string]string `json:"sender"`
	Recipient           map[string]string `json:"recipient"`
	Description         string            `json:"description,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
}

// TransactionUpdatePayload тело сообщения для сервиса аналитики для обновления статуса платежа
type TransactionUpdatePayload struct {
	ID       string    `json:"id"`
	Status   string    `json:"transaction_status" validate:"required"`
	Reason   string    `json:"reason"`
	UpdateAt time.Time `json:"update_at"`
}

// ResponsePayload тело сообщения с ответом от других сервисов
type ResponsePayload struct {
	Success bool   `json:"success"`
	Reason  string `json:"reason,omitempty"`
}

// InsertEventRequest обертка для создания события
type InsertEventRequest struct {
	*Event
}

// GetEventRequest структура для запроса на получение событий
type GetEventRequest struct {
	ProcessedAt    time.Time `json:"processed_at"`
	FiveMinutesAgo time.Time `json:"five_minutes_ago"`
	Limit          int       `db:"limit" json:"limit" validate:"required"`
}

// GetEventResponse события для отправки из outbox
type GetEventResponse struct {
	Events []Event `json:"events"`
}

// UpdateAccountRequest структура события
type UpdateAccountRequest struct {
	TransactionID uuid.UUID `db:"transaction_id" json:"transaction_id"`
	Code          string    `db:"code" json:"сode" validate:"required"`
}

// Event структура для запроса на обновление события
type UpdateEventStatusRequest struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Status      string    `db:"event_status" json:"event_status" validate:"required,oneof=completed pending"`
	ProcessedAt time.Time `json:"processed_at"`
}

// TopicMap мапа событий по топикам
var TopicMap = map[string]string{
	EventTransactionCreated:   TopicTransactionEvents,
	EventTransactionCompleted: TopicTransactionEvents,
	EventTransactionCancelled: TopicTransactionEvents,
	EventTransactionBlocked:   TopicTransactionEvents,

	EventFXActivateRequest:  TopicCurrencyCommands,
	EventFXActivateResponse: TopicCurrencyEvents,
	EventFXCommitRequest:    TopicCurrencyCommands,
	EventFXCommitResponse:   TopicCurrencyEvents,
	EventFXCancelRequest:    TopicCurrencyCommands,
	EventFXCancelResponse:   TopicCurrencyEvents,

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
