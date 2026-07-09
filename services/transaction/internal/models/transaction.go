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

	// TransactionTypeOnUsTransfer - перевод между счетами внутри системы с валютным обменом
	TransactionTypeOnUsFXTransfer string = "ON_US_FX_TRANSFER"

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
	TransactionTypeAdjustmentCredit string = "ADJUSTMENT_CREDIT"

	// TransactionTypeAdjustment - административная операция
	TransactionTypeAdjustmentDebit string = "ADJUSTMENT_DEBIT"
)

// Типы саг
const (
	SagaOnUsTransfer       string = "ON_US_TRANSFER"
	SagaOnUsFXTransfer     string = "ON_US_FX_TRANSFER"
	SagaOutboundRemittance string = "OUTBOUND_REMITTANCE"
	SagaInboundRemittance  string = "INBOUND_REMITTANCE"
	SagaTopUpRequest       string = "TOPUP_REQUEST"
	SagaRefundInternal     string = "REFUND_INTERNAL"
	SagaRefundOutbound     string = "REFUND_OUTBOUND"
	SagaRefundInbound      string = "REFUND_INBOUND"
	SagaRefundOnUsFX       string = "REFUND_ON_US_FX"
	SagaChargebackInternal string = "CHARGEBACK_INTERNAL"
	SagaChargebackOutbound string = "CHARGEBACK_OUTBOUND"
	SagaChargebackInbound  string = "CHARGEBACK_INBOUND"
	SagaChargebackOnUsFX   string = "CHARGEBACK_ON_US_FX"
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

// Причины отмены транзакции
const (
	ReasonTransactionCannotBeComplited string = "blocked, transaction cannot be completed"
	ReasonSenderFreezeRejected         string = "sender freeze rejected"
	ReasonCompensated                  string = "compensated"
	ReasonTopUpRejected                string = "topup_request_rejected"
	ReasonCompleted                    string = ""
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

// Transaction структура транзакции
type Transaction struct {
	ID                  uuid.UUID          `db:"id" json:"id"`
	IdempotencyKey      string             `db:"idempotency_key" json:"idempotency_key"`
	ParentTransactionID *uuid.UUID         `db:"parent_transaction_id" json:"parent_transaction_id,omitempty"`
	FXDealID            *uuid.UUID         `db:"fx_deal_id" json:"fx_deal_id,omitempty"`
	Type                string             `db:"transaction_type" json:"transaction_type" validate:"required"`
	SagaType            string             `db:"saga_type" json:"saga_type"`
	SagaState           string             `db:"saga_state" json:"saga_state"`
	Status              string             `db:"transaction_status" json:"transaction_status"`
	Amount              float64            `db:"amount" json:"amount" validate:"required"`
	SourceCurrency      string             `db:"source_currency" json:"source_currency" validate:"required"`
	TargetCurrency      string             `db:"target_currency" json:"target_currency" validate:"required"`
	Description         string             `db:"transaction_description" json:"transaction_description,omitempty"`
	CreatedAt           time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time          `db:"updated_at" json:"updated_at"`
	Parties             []TransactionParty `json:"parties"`
}

// TransactionParty структура реквизитов
type TransactionParty struct {
	TransactionID uuid.UUID         `db:"transaction_id" json:"transaction_id" validate:"required"`
	PartyRole     string            `db:"party_role" json:"party_role" validate:"required"`
	PartyType     string            `db:"party_type" json:"party_type" validate:"required"`
	Identifiers   map[string]string `db:"identifiers" json:"identifiers" validate:"required"`
}

// CleanMap сохраняет только не пустые значения map
func CleanMap(m map[string]string) map[string]string {
	result := make(map[string]string, len(m))
	for key, value := range m {
		if value != "" {
			result[key] = value
		}
	}
	return result
}

// InsertTransactionRequest обертка для вставки транзакции в бд
type InsertTransactionRequest struct {
	*Transaction
}

// InsertPartiesRequest обертка для вставки реквизитов в бд
type InsertPartiesRequest struct {
	Parties []TransactionParty `json:"parties"`
}

// UpdateSagaStateRequest структура для запроса на обновление состояния саги
type UpdateSagaStateRequest struct {
	TransactionID uuid.UUID `db:"id" json:"id" validate:"required"`
	ExpectedFrom  string    `db:"-" json:"-,omitempty"`
	SagaState     string    `db:"saga_state" json:"saga_state" validate:"required"`
	Status        *string   `db:"transaction_status" json:"transaction_status,omitempty"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

// GetTransactionRequest структура для запроса на получение транзакции
type GetTransactionRequest struct {
	ID uuid.UUID `db:"internal__id" json:"internal__id" validate:"required"`
}

// GetTransactionResponse обертка для транзакции из бд при вызове Get
type GetTransactionResponse struct {
	*Transaction
}

// UpdateTransactionStatusRequest структура для запроса на обновление статуса транзкации
type UpdateTransactionStatusRequest struct {
	ID     uuid.UUID `db:"id" json:"id"`
	Status string    `json:"transaction_status" validate:"required,oneof=completed cancelled blocked pending"`
}

// TransactionPayload payload с информацией о платеже для создания транзакции
type TransactionPayload struct {
	Type           string  `json:"transaction_type" validate:"required"`
	Amount         float64 `json:"amount" validate:"required"`
	SourceCurrency string  `json:"source_currency" validate:"required"`
	TargetCurrency string  `json:"target_currency" validate:"required"`
	IdempotencyKey string  `json:"idempotency_key" validate:"required"`
	FXDealID       string  `json:"fx_deal_id,omitempty"`

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
	SenderMandateID    string `json:"sender_mandate_id,omitempty"`

	// Получатель
	RecipientAccountCode  string `json:"recipient_account_code,omitempty"`
	RecipientPhone        string `json:"recipient_phone,omitempty"`
	RecipientCardNumber   string `json:"recipient_card_number,omitempty"`
	RecipientIBAN         string `json:"recipient_iban,omitempty"`
	RecipientSWIFTBIC     string `json:"recipient_swift_bic,omitempty"`
	RecipientBankCode     string `json:"recipient_bank_code,omitempty"`
	RecipientBankCodeType string `json:"recipient_bank_code_type,omitempty"`
	RecipientName         string `json:"recipient_name,omitempty"`
	RecipientMandateID    string `json:"recipient_mandate_id,omitempty"`
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
		"MANDATE_ID":     p.SenderMandateID,
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
		"MANDATE_ID":     p.RecipientMandateID,
	}
}
