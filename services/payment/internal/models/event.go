// Пакет models содержит различные dto'шки, модельки, константы
package models

import (
	"time"

	"github.com/google/uuid"
)

// Топики для событий:
const (
	TopicAccount         = "account-commands"
	TopicExternalPayment = "external-payment-commands"
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
	EventFreezeComplete   = "freeze.complete"
	EventFreezeFail       = "freeze.fail"
	EventUnfreezeRequest  = "unfreeze.request"
	EventUnfreezeComplete = "unfreeze.complete"
	EventUnfreezeFail     = "unfreeze.fail"

	EventReserveRequest    = "reserve.request"
	EventReserveComplete   = "reserve.complete"
	EventReserveFail       = "reserve.fail"
	EventUnreserveRequest  = "unreserve.request"
	EventUnreserveComplete = "unreserve.complete"
	EventUnreserveFail     = "unreserve.fail"

	EventDepositRequest  = "deposit.request"
	EventDepositComplete = "deposit.complete"
	EventDepositFail     = "deposit.fail"

	EventWithdrawRequest  = "withdraw.request"
	EventWithdrawComplete = "withdraw.complete"
	EventWithdrawFail     = "withdraw.fail"

	EventExternalRequest  = "external.request"
	EventExternalComplete = "external.complete"
	EventExternalFail     = "external.fail"
	EventExternalCommit   = "external.commit"
	EventExternalRollback = "external.rollback"
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

type ExternalRequestPayload struct {
	ISOMessage string `json:"iso_message"`
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
