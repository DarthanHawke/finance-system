// Пакет models содержит различные dto'шки, модельки, константы
package models

import (
	"time"

	"github.com/google/uuid"
)

const Source = "payment-service"

// Статусы платежей:
const (
	// PaymentStatusPending - платеж в обработке
	PaymentStatusPending string = "pending"

	// PaymentStatusCompleted - платеж завершён
	PaymentStatusCompleted string = "completed"

	// PaymentStatusCancelled - платеж отменён
	PaymentStatusCancelled string = "cancelled"
)

// Тип платежей
const (
	// AccountInternal - внутренний счет
	AccountInternal string = "internal"

	// AccountExternal - внешний счет
	AccountExternal string = "external"
)

// Валюты:
const (
	// CurrencyUSD - доллары
	CurrencyUSD string = "USD"

	// CurrencyEUR - евро
	CurrencyEUR string = "EUR"

	// CurrencyRUB - рубли
	CurrencyRUB string = "RUB"
)

// Payment - дтошка платежа
type Payment struct {
	ID                   uuid.UUID `db:"id" json:"id"`
	PaymentType          string    `db:"payment_type" json:"payment_type"`
	Direction            string    `db:"direction" json:"direction"`
	Amount               float64   `db:"amount" json:"amount"`
	Currency             string    `db:"currency" json:"currency"`
	SenderType           string    `db:"sender_type" json:"sender_type"`
	SenderAccountCode    string    `db:"sender_account_code" json:"sender_account_code"`
	SenderPhone          string    `db:"sender_phone" json:"sender_phone"`
	SenderCardNumber     string    `db:"sender_card_number" json:"sender_card_number"`
	RecipientType        string    `db:"recipient_type" json:"recipient_type"`
	RecipientAccountCode string    `db:"recipient_account_code" json:"recipient_account_code"`
	RecipientPhone       string    `db:"recipient_phone" json:"recipient_phone"`
	RecipientCardNumber  string    `db:"recipient_card_number" json:"recipient_card_number"`
	ProcessingCode       string    `db:"processing_code" json:"processing_code"`
	Stan                 string    `db:"stan" json:"stan"`
	AuthorizationCode    string    `db:"authorization_code" json:"authorization_code"`
	Description          string    `db:"description" json:"description,omitempty"`
	Status               string    `db:"status" json:"status"`
	CreatedAt            time.Time `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time `db:"updated_at" json:"updated_at"`
}

type CreatePaymentRequest struct {
	*Payment
	*Event
}

type GetPaymentRequest struct {
	ID uuid.UUID `db:"internal__id" json:"internal__id" validate:"required"`
}

type GetPaymentResponse struct {
	*Payment
}

type GetPaymentsRequest struct {
	AccountCode string `json:"code" validate:"required"`
	Limit       int    `db:"limit" json:"limit" validate:"required"`
	Offset      int    `db:"limit" json:"offset" validate:"required"`
}

type GetPaymentsResponse struct {
	Payments []Payment `json:"payments"`
}

type UpdatePaymentStatusRequest struct {
	ID                   uuid.UUID `json:"id" validate:"required"`
	SenderAccountCode    string    `db:"sender_account_code" json:"sender_account_code"`
	RecipientAccountCode string    `db:"recipient_account_code" json:"recipient_account_code"`
	Status               string    `json:"status" validate:"required,oneof=completed cancelled failed refunded"`
}
