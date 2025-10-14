package iso8583

import (
	"fmt"
	"payment-service/internal/models"
	"strings"
	"time"

	"github.com/moov-io/iso8583"
)

// ISO8583Generator генерирует ISO 8583 сообщения
type ISO8583Generator struct {
	config *Config
}

// NewGenerator создает новый генератор
func NewGenerator(config *Config) *ISO8583Generator {
	if config == nil {
		config = DefaultConfig()
	}
	return &ISO8583Generator{
		config: config,
	}
}

// GenerateMessage генерирует ISO 8583 сообщение на основе платежа
func (g *ISO8583Generator) GenerateMessage(payment *models.Payment) ([]byte, error) {
	// Создаем новое сообщение
	message := iso8583.NewMessage(iso8583.Spec87)

	// Устанавливаем MTI (Message Type Indicator)
	mti, err := g.getMTI(payment)
	if err != nil {
		return nil, fmt.Errorf("failed to get MTI: %w", err)
	}
	message.MTI(mti)

	// Заполняем основные поля
	if err := g.populateFields(message, payment); err != nil {
		return nil, fmt.Errorf("failed to populate fields: %w", err)
	}

	// Генерируем бинарное представление
	rawMessage, err := message.Pack()
	if err != nil {
		return nil, fmt.Errorf("failed to pack message: %w", err)
	}

	return rawMessage, nil
}

// getMTI возвращает тип сообщения на основе типа платежа
func (g *ISO8583Generator) getMTI(payment *models.Payment) (string, error) {
	switch payment.PaymentType {
	case "purchase", "payment":
		return "0200", nil // Financial transaction request
	case "refund":
		return "0200", nil // Refund request
	case "reversal":
		return "0420", nil // Reversal request
	case "balance_inquiry":
		return "0100", nil // Authorization request
	default:
		return "0200", nil // По умолчанию финансовый запрос
	}
}

// populateFields заполняет поля ISO 8583
func (g *ISO8583Generator) populateFields(message *iso8583.Message, payment *models.Payment) error {
	// Поле 2: Primary Account Number (PAN)
	if payment.SenderCardNumber != "" {
		if err := message.Field(2, payment.SenderCardNumber); err != nil {
			return err
		}
	}

	// Поле 3: Processing Code
	processingCode := g.getProcessingCode(payment)
	if err := message.Field(3, processingCode); err != nil {
		return err
	}

	// Поле 4: Amount, Transaction
	amount := g.formatAmount(payment.Amount)
	if err := message.Field(4, amount); err != nil {
		return err
	}

	// Поле 7: Transmission Date & Time
	transmissionTime := g.formatTransmissionTime(payment.CreatedAt)
	if err := message.Field(7, transmissionTime); err != nil {
		return err
	}

	// Поле 11: System Trace Audit Number (STAN)
	stan := g.getSTAN(payment)
	if err := message.Field(11, stan); err != nil {
		return err
	}

	// Поле 12: Local Transaction Time
	localTime := g.formatLocalTime(payment.CreatedAt)
	if err := message.Field(12, localTime); err != nil {
		return err
	}

	// Поле 13: Local Transaction Date
	localDate := g.formatLocalDate(payment.CreatedAt)
	if err := message.Field(13, localDate); err != nil {
		return err
	}

	// Поле 18: Merchant Type
	if err := message.Field(18, "5999"); err != nil { // Miscellaneous stores
		return err
	}

	// Поле 22: POS Entry Mode
	if err := message.Field(22, g.config.POSEntryMode); err != nil {
		return err
	}

	// Поле 25: POS Condition Code
	if err := message.Field(25, "00"); err != nil { // Normal presentment
		return err
	}

	// Поле 32: Acquiring Institution Identification Code
	if err := message.Field(32, g.config.AcquiringInstitutionID); err != nil {
		return err
	}

	// Поле 37: Retrieval Reference Number
	rrn := g.generateRRN(payment)
	if err := message.Field(37, rrn); err != nil {
		return err
	}

	// Поле 39: Response Code (для ответов, в запросе обычно не заполняется)
	if payment.Status != "" {
		responseCode := g.getResponseCode(payment.Status)
		if responseCode != "" {
			if err := message.Field(39, responseCode); err != nil {
				return err
			}
		}
	}

	// Поле 41: Card Acceptor Terminal Identification
	if err := message.Field(41, g.config.TerminalID); err != nil {
		return err
	}

	// Поле 42: Card Acceptor Identification Code
	if err := message.Field(42, g.config.MerchantID); err != nil {
		return err
	}

	// Поле 43: Card Acceptor Name/Location
	merchantData := g.formatMerchantData()
	if err := message.Field(43, merchantData); err != nil {
		return err
	}

	// Поле 49: Currency Code, Transaction
	if err := message.Field(49, g.config.CurrencyCode); err != nil {
		return err
	}

	// Поле 52: PIN Data (в реальной системе здесь был бы зашифрованный PIN)
	// В эмуляции оставляем пустым

	// Поле 54: Additional Amounts
	additionalAmounts := g.formatAdditionalAmounts(payment)
	if additionalAmounts != "" {
		if err := message.Field(54, additionalAmounts); err != nil {
			return err
		}
	}

	// Поле 102: Account Identification 1 (отправитель)
	if payment.SenderAccountCode != "" {
		if err := message.Field(102, payment.SenderAccountCode); err != nil {
			return err
		}
	}

	// Поле 103: Account Identification 2 (получатель)
	if payment.RecipientAccountCode != "" {
		if err := message.Field(103, payment.RecipientAccountCode); err != nil {
			return err
		}
	}

	// Поле 111: Amount, Original (для возвратов и отмен)
	if payment.PaymentType == "refund" || payment.PaymentType == "reversal" {
		originalAmount := g.formatAmount(payment.Amount)
		if err := message.Field(111, originalAmount); err != nil {
			return err
		}
	}

	// Поле 120: Private Use - дополнительные данные
	privateData := g.formatPrivateData(payment)
	if privateData != "" {
		if err := message.Field(120, privateData); err != nil {
			return err
		}
	}

	return nil
}

// Вспомогательные методы

func (g *ISO8583Generator) getProcessingCode(payment *models.Payment) string {
	if payment.ProcessingCode != "" {
		return payment.ProcessingCode
	}

	switch payment.PaymentType {
	case "purchase", "payment":
		return "000000" // Purchase
	case "refund":
		return "200000" // Refund
	case "withdrawal":
		return "010000" // Cash withdrawal
	case "balance_inquiry":
		return "310000" // Balance inquiry
	default:
		return "000000"
	}
}

func (g *ISO8583Generator) formatAmount(amount float64) string {
	// Преобразуем в центы/копейки (минимальные единицы)
	cents := int(amount * 100)
	return fmt.Sprintf("%012d", cents)
}

func (g *ISO8583Generator) formatTransmissionTime(t time.Time) string {
	return t.Format("0102150405") // MMDDhhmmss
}

func (g *ISO8583Generator) formatLocalTime(t time.Time) string {
	return t.Format("150405") // hhmmss
}

func (g *ISO8583Generator) formatLocalDate(t time.Time) string {
	return t.Format("0102") // MMDD
}

func (g *ISO8583Generator) getSTAN(payment *models.Payment) string {
	if payment.Stan != "" {
		return fmt.Sprintf("%06s", payment.Stan)
	}
	return ""
}

func (g *ISO8583Generator) generateRRN(payment *models.Payment) string {
	// Retrieval Reference Number - уникальный идентификатор транзакции
	timestamp := payment.CreatedAt.Format("060102")
	uniquePart := fmt.Sprintf("%06d", payment.ID.Time())
	return timestamp + uniquePart
}

func (g *ISO8583Generator) getResponseCode(status string) string {
	switch status {
	case "completed", "success":
		return "00" // Approved
	case "pending":
		return "09" // Contact acquirer
	case "failed", "declined":
		return "05" // Do not honor
	case "insufficient_funds":
		return "51" // Insufficient funds
	default:
		return "06" // Error
	}
}

func (g *ISO8583Generator) formatMerchantData() string {
	// Формат: Merchant Name + Город + Страна (фиксированная длина)
	name := fmt.Sprintf("%-25s", g.config.MerchantName)[:25]
	city := fmt.Sprintf("%-13s", g.config.MerchantLocation)[:13]
	country := fmt.Sprintf("%-2s", "RU")[:2] // код страны
	return name + city + country
}

func (g *ISO8583Generator) formatAdditionalAmounts(payment *models.Payment) string {
	// Формат: ТипСчетаКодВалютыЗнакСуммыСумма
	// Для реальной системы здесь могут быть дополнительные суммы
	return ""
}

func (g *ISO8583Generator) formatPrivateData(payment *models.Payment) string {
	// Приватные данные в формате TAG=Value|TAG=Value
	var data []string

	if payment.Description != "" {
		data = append(data, fmt.Sprintf("DESC=%s", payment.Description))
	}

	if payment.SenderPhone != "" {
		data = append(data, fmt.Sprintf("SNDPH=%s", payment.SenderPhone))
	}

	if payment.RecipientPhone != "" {
		data = append(data, fmt.Sprintf("RCVPH=%s", payment.RecipientPhone))
	}

	if payment.AuthorizationCode != "" {
		data = append(data, fmt.Sprintf("AUTH=%s", payment.AuthorizationCode))
	}

	return strings.Join(data, "|")
}
