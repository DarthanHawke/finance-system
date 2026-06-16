// Пакет models содержит различные dto'шки, модельки, константы
package models

import (
	"time"

	"github.com/google/uuid"
)

const Source string = "transaction-service"

// Статусы платежей:
const (
	// TransactionStatusProcessing - платеж в обработке
	TransactionStatusProcessing string = "PROCESSING"

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
	TransactionTypeOnUsTransfer string = "ON_US_TRANSFER"

	// TransactionTypeInboundRemittance - входящий перевод из внешней системы
	TransactionTypeInboundRemittance string = "INBOUND_REMITTANCE"

	// TransactionTypeOutboundRemittance - исходящий перевод во внешнюю систему
	TransactionTypeOutboundRemittance string = "OUTBOUND_REMITTANCE"

	// TransactionTypeDirectDebit - списание по требованию получателя внутри системы
	TransactionTypeDirectDebit string = "DIRECT_DEBIT"

	// TransactionTypeInboundDirectDebit - входящий перевод по мандату из внешней системы
	TransactionTypeInboundDirectDebit string = "INBOUND_DIRECT_DEBIT"

	// TransactionTypeOutboundDirectDebit - исходящий перевод по мандату во внешнюю систему
	TransactionTypeOutboundDirectDebit string = "OUTBOUND_DIRECT_DEBIT"

	// TransactionTypeTopUpRequest - запрос на входящий перевод из внешнего источника
	TransactionTypeTopUpRequest string = "TOPUP_REQUEST"

	// TransactionTypeRefund - возврат
	TransactionTypeRefund string = "REFUND"

	// TransactionTypeChargeback - принудительный возврат, инициированный внешней системой
	TransactionTypeChargeback string = "CHARGEBACK"

	// TransactionTypeAdjustment - административная операция
	TransactionTypeAdjustment string = "ADJUSTMENT"
)

// Типы саг
const (
	SagaOnUsTransfer       string = "ON_US_TRANSFER"
	SagaOutboundRemittance string = "OUTBOUND_REMITTANCE"
	SagaInboundRemittance  string = "INBOUND_REMITTANCE"
	SagaTopUpRequest       string = "TOPUP_REQUEST"
	SagaRefundInternal     string = "REFUND_INTERNAL"
	SagaRefundOutbound     string = "REFUND_OUTBOUND"
	SagaRefundInbound      string = "REFUND_INBOUND"
	SagaChargebackInternal string = "CHARGEBACK_INTERNAL"
	SagaChargebackOutbound string = "CHARGEBACK_OUTBOUND"
	SagaChargebackInbound  string = "CHARGEBACK_INBOUND"
	SagaAdjustmentCredit   string = "ADJUSTMENT_CREDIT"
	SagaAdjustmentDebit    string = "ADJUSTMENT_DEBIT"
)

// Роли участников
const (
	PartyRoleSender    string = "SENDER"
	PartyRoleRecipient string = "RECIPIENT"
)

// Типы участников
const (
	PartyTypeInternal string = "INTERNAL"
	PartyTypeExternal string = "EXTERNAL"
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
	IdempotencyKey      string             `db:"idempotency_key" json:"idempotency_key"`
	ParentTransactionID uuid.UUID          `db:"parent_transaction_id" json:"parent_transaction_id,omitempty"`
	Type                string             `db:"type" json:"type" validate:"required"`
	SagaType            string             `db:"saga_type" json:"saga_type"`
	SagaState           string             `db:"saga_state" json:"saga_state"`
	Status              string             `db:"status" json:"status"`
	Amount              float64            `db:"amount" json:"amount" validate:"required"`
	Currency            string             `db:"currency" json:"currency" validate:"required"`
	Description         string             `db:"description" json:"description,omitempty"`
	CreatedAt           time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time          `db:"updated_at" json:"updated_at"`
	Parties             []TransactionParty `json:"parties"`
}

type TransactionParty struct {
	PartyRole   string            `db:"role" json:"role"`
	PartyType   string            `db:"party_type" json:"party_type"`
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

type TransactionPayload struct {
	Type           string  `json:"type" validate:"required"`
	Amount         float64 `json:"amount" validate:"required"`
	Currency       string  `json:"currency" validate:"required"`
	IdempotencyKey string  `json:"idempotency_key" validate:"required"`

	ParentTransactionID string `json:"parent_transaction_id,omitempty"`
	ExternalReference   string `json:"external_reference,omitempty"`
	Description         string `json:"description,omitempty"`

	// Отправитель
	SenderAccountCode  string `json:"sender_account_code,omitempty"`
	SenderPhone        string `json:"sender_phone,omitempty"`
	SenderCardNumber   string `json:"sender_card_number,omitempty"`
	SenderIBAN         string `json:"sender_iban,omitempty"`
	SenderSWIFTBIC     string `json:"sender_swift_bic,omitempty"`
	SenderBankCode     string `json:"sender_bank_code,omitempty"`
	SenderBankCodeType string `json:"sender_bank_code_type,omitempty"`
	SenderName         string `json:"sender_name,omitempty"`

	// Получатель
	RecipientAccountCode  string `json:"recipient_account_code,omitempty"`
	RecipientPhone        string `json:"recipient_phone,omitempty"`
	RecipientCardNumber   string `json:"recipient_card_number,omitempty"`
	RecipientIBAN         string `json:"recipient_iban,omitempty"`
	RecipientSWIFTBIC     string `json:"recipient_swift_bic,omitempty"`
	RecipientBankCode     string `json:"recipient_bank_code,omitempty"`
	RecipientBankCodeType string `json:"recipient_bank_code_type,omitempty"`
	RecipientName         string `json:"recipient_name,omitempty"`
}

// SenderType возвращает party type для sender исходя из transaction type
func (p *TransactionPayload) SenderType() string {
	switch p.Type {
	case TransactionTypeInboundRemittance, TransactionTypeInboundDirectDebit, TransactionTypeChargeback:
		return PartyTypeExternal
	default:
		return PartyTypeInternal
	}
}

// RecipientType возвращает party type для recipient исходя из transaction type
func (p *TransactionPayload) RecipientType() string {
	switch p.Type {
	case TransactionTypeOutboundRemittance, TransactionTypeOutboundDirectDebit, TransactionTypeTopUpRequest:
		return PartyTypeExternal
	default:
		return PartyTypeInternal
	}
}

// SenderFields собирает поля отправителя в map
func (p *TransactionPayload) SenderFields() map[string]string {
	return map[string]string{
		"ACCOUNT_CODE":   p.SenderAccountCode,
		"PHONE":          p.SenderPhone,
		"CARD_NUMBER":    p.SenderCardNumber,
		"IBAN":           p.SenderIBAN,
		"SWIFT_BIC":      p.SenderSWIFTBIC,
		"BANK_CODE":      p.SenderBankCode,
		"BANK_CODE_TYPE": p.SenderBankCodeType,
		"NAME":           p.SenderName,
	}
}

// RecipientFields собирает поля получателя в map
func (p *TransactionPayload) RecipientFields() map[string]string {
	return map[string]string{
		"ACCOUNT_CODE":   p.RecipientAccountCode,
		"PHONE":          p.RecipientPhone,
		"CARD_NUMBER":    p.RecipientCardNumber,
		"IBAN":           p.RecipientIBAN,
		"SWIFT_BIC":      p.RecipientSWIFTBIC,
		"BANK_CODE":      p.RecipientBankCode,
		"BANK_CODE_TYPE": p.RecipientBankCodeType,
		"NAME":           p.RecipientName,
	}
}
