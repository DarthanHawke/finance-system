// Пакет models содержит различные dto'шки, модельки, константы
package models

import (
	"time"

	"github.com/google/uuid"
)

const Source = "transaction-service"

// Статусы платежей:
const (
	// TransactionStatusProcessing - платеж в обработке
	TransactionStatusProcessing string = "PROCESSING"

	// TransactionStatusAwaitingExternal - ожидает ответа от внешней системы
	TransactionStatusAwaitingExternal = "AWAITING_EXTERNAL"

	// TransactionStatusCompleted - платеж завершён
	TransactionStatusCompleted string = "COMPLETED"

	// TransactionStatusCancelled - платеж отменён
	TransactionStatusCancelled string = "CANCELLED"

	// TransactionStatusBlocked - платеж не может быть выполнен, требуется ручное вмешательство
	TransactionStatusBlocked string = "BLOCKED"
)

// Типы транзакций
const (
	// TransactionTypeOnUsTransfer - перевод между счетами внутри системы
	TransactionTypeOnUsTransfer = "ON_US_TRANSFER"

	// TransactionTypeInboundRemittance - входящий перевод из внешней системы
	TransactionTypeInboundRemittance = "INBOUND_REMITTANCE"

	// TransactionTypeOutboundRemittance - исходящий перевод во внешнюю систему
	TransactionTypeOutboundRemittance = "OUTBOUND_REMITTANCE"

	// TransactionTypeDirectDebit - списание по требованию получателя внутри системы
	TransactionTypeDirectDebit = "DIRECT_DEBIT"

	// TransactionTypeInboundDirectDebit - входящий перевод по мандату из внешней системы
	TransactionTypeInboundDirectDebit = "INBOUND_DIRECT_DEBIT"

	// TransactionTypeOutboundDirectDebit - исходящий перевод по мандату во внешнюю систему
	TransactionTypeOutboundDirectDebit = "OUTBOUND_DIRECT_DEBIT"

	// TransactionTypeTopUpRequest - запрос на входящий перевод из внешнего источника
	TransactionTypeTopUpRequest = "TOPUP_REQUEST"

	// TransactionTypeRefund - возврат
	TransactionTypeRefund = "REFUND"

	// TransactionTypeChargeback - принудительный возврат, инициированный внешней системой
	TransactionTypeChargeback = "CHARGEBACK"

	// TransactionTypeAdjustment - административная операция
	TransactionTypeAdjustment = "ADJUSTMENT"
)

// Типы инициаторов
const (
	// InitiatorCustomer - клиент
	InitiatorCustomer = "CUSTOMER"

	// InitiatorExternal - внешняя система
	InitiatorExternal = "EXTERNAL"

	// InitiatorSystem - система
	InitiatorSystem = "SYSTEM"
)

// Роли участников
const (
	RoleSender    = "SENDER"
	RoleRecipient = "RECIPIENT"
)

// Типы участников
const (
	TypeInternal = "INTERNAL"
	TypeExternal = "EXTERNAL"
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

// Transaction - дтошка платежа
type Transaction struct {
	ID                  uuid.UUID          `db:"id" json:"id"`
	IdempotencyKey      *string            `db:"idempotency_key" json:"idempotency_key,omitempty"`
	ParentTransactionID *uuid.UUID         `db:"parent_transaction_id" json:"parent_transaction_id,omitempty"`
	Type                string             `db:"type" json:"type"`
	Status              string             `db:"status" json:"status"`
	Amount              float64            `db:"amount" json:"amount"`
	Currency            string             `db:"currency" json:"currency"`
	Initiator           string             `db:"initiator" json:"initiator"`
	Description         string             `db:"description" json:"description,omitempty"`
	CreatedAt           time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time          `db:"updated_at" json:"updated_at"`
	Parties             []TransactionParty `json:"parties,omitempty"`
}

type TransactionParty struct {
	Role        string            `db:"role" json:"role"`
	Type        string            `db:"party_type" json:"party_type"`
	Identifiers map[string]string `db:"identifiers" json:"identifiers"`
}

type CreateTransactionRequest struct {
	*Transaction
}

type GetTransactionRequest struct {
	ID uuid.UUID `db:"internal__id" json:"internal__id" validate:"required"`
}

type GetTransactionResponse struct {
	*Transaction
}

type GetTransactionsRequest struct {
	AccountCode string `json:"code" validate:"required"`
	Limit       int    `db:"limit" json:"limit" validate:"required"`
	Offset      int    `db:"offset" json:"offset" validate:"required"`
}

type GetTransactionsResponse struct {
	Transactions []Transaction `json:"transactions"`
}

type UpdateTransactionStatusRequest struct {
	ID                   uuid.UUID `json:"id" validate:"required"`
	SenderAccountCode    string    `db:"sender_account_code" json:"sender_account_code"`
	RecipientAccountCode string    `db:"recipient_account_code" json:"recipient_account_code"`
	Status               string    `json:"status" validate:"required,oneof=completed cancelled blocked pending"`
}
