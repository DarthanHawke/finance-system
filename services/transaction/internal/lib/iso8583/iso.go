// iso8583 пакет для работы с ISO 8583
package iso8583

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"transaction-service/internal/config"
	"transaction-service/internal/models"

	"github.com/google/uuid"
)

var nilStr = ""

// ISO8583 структура релизующая методы для работы с ISO8583
type ISO8583 struct {
	config *config.ISO8583Config
}

func NewISO8583(cfg *config.ISO8583Config) *ISO8583 {
	return &ISO8583{
		config: cfg,
	}
}

// ParseIncomingMessage парсит входящее ISO сообщение из JSON
func (iso *ISO8583) ParseIncomingMessage(isoData []byte) (*models.ISO8583Message, error) {
	var isoMsg models.ISO8583Message

	if err := json.Unmarshal(isoData, &isoMsg); err != nil {
		return nil, fmt.Errorf("failed to parse ISO message: %w", err)
	}
	return &isoMsg, nil
}

// CreateFinancialRequest создает финансовый запрос (0200)
func (iso *ISO8583) CreateFinancialRequest(
	transaction *models.Transaction,
) (*models.ISO8583Message, error) {
	amount, err := iso.formatAmount(transaction.Amount, transaction.Currency)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %s", err)
	}

	msg := &models.ISO8583Message{
		MTI:              models.MTIFinancialRequest,
		ProcessingCode:   iso.generateProcessingCode(transaction),
		Amount:           amount,
		TransmissionTime: time.Now(),
		STAN:             transaction.Stan,
		LocalTime:        time.Now(),
		LocalDate:        time.Now(),
		AcquirerID:       iso.config.AcquirerID,
		TerminalID:       iso.config.TerminalID,
		MerchantID:       iso.config.MerchantID,
		Currency:         iso.config.CurrencyCode,
		AccountID1:       transaction.SenderAccountCode,
		AccountID2:       transaction.RecipientAccountCode,
	}

	return msg, nil
}

// CreateFinancialResponse создает финансовый ответ (0210)
func (iso *ISO8583) CreateFinancialResponse(
	transaction *models.Transaction,
	success bool,
	reason string,
) (*models.ISO8583Message, error) {
	amount, err := iso.formatAmount(transaction.Amount, transaction.Currency)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %s", err)
	}

	msg := &models.ISO8583Message{
		MTI:              models.MTIFinancialResponse,
		ProcessingCode:   transaction.ProcessingCode,
		Amount:           amount,
		TransmissionTime: time.Now(),
		STAN:             transaction.Stan,
		LocalTime:        time.Now(),
		LocalDate:        time.Now(),
		AcquirerID:       iso.config.AcquirerID,
		TerminalID:       iso.config.TerminalID,
		Currency:         iso.config.CurrencyCode,
	}

	if success {
		msg.ResponseCode = models.RCSuccess
		msg.AuthorizationID = transaction.AuthorizationCode
	} else {
		msg.ResponseCode = iso.getErrorCode(reason)
		msg.AdditionalData = reason
	}

	return msg, nil
}

// CreateReversalMessage создает сообщение отмены (0420)
func (iso *ISO8583) CreateReversalMessage(
	transaction *models.Transaction,
	reason string,
) (*models.ISO8583Message, error) {
	amount, err := iso.formatAmount(transaction.Amount, transaction.Currency)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %s", err)
	}

	msg := &models.ISO8583Message{
		MTI:              models.MTIReversalAdvice,
		ProcessingCode:   models.PCTransactionRefund + models.PCAccountDefault + models.PCAccountDefault, // 200000
		Amount:           amount,
		TransmissionTime: time.Now(),
		STAN:             transaction.Stan,
		LocalTime:        time.Now(),
		LocalDate:        time.Now(),
		AcquirerID:       iso.config.AcquirerID,
		TerminalID:       iso.config.TerminalID,
		Currency:         iso.config.CurrencyCode,
		ResponseCode:     iso.getErrorCode(reason),
		AdditionalData:   reason,
		OriginalData:     fmt.Sprintf("Original: %s", transaction.ID),
	}

	return msg, nil
}

// CreateTransactionFromISO создает транзакцию из ISO сообщения
func (iso *ISO8583) CreateTransactionFromISO(msg *models.ISO8583Message) (*models.CreateTransactionRequest, error) {
	analysis := iso.AnalyzeProcessingCode(msg.ProcessingCode)
	if analysis == nil {
		return nil, fmt.Errorf("invalid processing code: %s", msg.ProcessingCode)
	}

	amount, err := iso.parseAmount(msg.Amount, msg.Currency)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %s, Error: %s", msg.Amount, err)
	}

	currency, err := iso.mapCurrencyCode(msg.Currency)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %s, Error: %s", msg.Amount, err)
	}

	transaction := &models.CreateTransactionRequest{
		Transaction: &models.Transaction{
			ID:                uuid.New(),
			Type:              string(analysis.OperationType),
			Amount:            amount,
			Currency:          currency,
			ProcessingCode:    msg.ProcessingCode,
			Stan:              msg.STAN,
			AuthorizationCode: msg.AuthorizationID,
			Status:            models.TransactionStatusPending,
			CreatedAt:         time.Now(),
		},
	}

	switch analysis.OperationType {
	case models.OpDeposit, models.OpRefund:
		transaction.SenderType = models.AccountExternal
		transaction.RecipientType = models.AccountInternal
		transaction.SenderAccountCode = msg.AccountID1
		transaction.RecipientAccountCode = msg.AccountID2
		transaction.SenderCardNumber = msg.PrimaryAccount
	case models.OpPurchase, models.OpTransfer, models.OpPayment:
		transaction.SenderType = models.AccountInternal
		transaction.RecipientType = models.AccountExternal
		transaction.SenderAccountCode = msg.AccountID1
		transaction.RecipientAccountCode = msg.AccountID2
	}

	return transaction, nil
}

// mapCurrencyCode возвращает буквенный код валюты
func (iso *ISO8583) mapCurrencyCode(currencyCode string) (string, error) {
	for code, info := range models.CurrencyMap {
		if info.Code == currencyCode {
			return code, nil
		}
	}
	return nilStr, fmt.Errorf("cant map currency, %s", currencyCode)
}

// parseAmount парсит сумму из ISO8583 с учетом валюты
func (iso *ISO8583) parseAmount(amountStr string, currency string) (float64, error) {
	if amountStr == "" {
		return 0, nil
	}

	info, exists := models.CurrencyMap[currency]
	if !exists {
		info = models.CurrencyInfo{Decimals: 2}
	}

	// Убираем ведущие нули
	amountStr = strings.TrimLeft(amountStr, "0")
	if amountStr == "" {
		return 0, nil
	}

	// Парсим как целое число (минимальные единицы)
	amountInMinor, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse amount: %w", err)
	}

	// Конвертируем в основные единицы
	divisor := math.Pow(10, float64(info.Decimals))
	amount := float64(amountInMinor) / divisor

	return amount, nil
}

// AnalyzeProcessingCode анализирует Processing Code
func (iso *ISO8583) AnalyzeProcessingCode(processingCode string) *models.AnalysisResult {
	if len(processingCode) != 6 {
		return nil
	}

	result := &models.AnalysisResult{
		RawCode:         processingCode,
		TransactionType: processingCode[:2],
		FromAccountType: processingCode[2:4],
		ToAccountType:   processingCode[4:6],
	}

	// Определяем тип операции
	switch result.TransactionType {
	case models.PCTransactionPurchase:
		result.OperationType = models.OpPurchase
	case models.PCTransactionWithdrawal:
		result.OperationType = models.OpWithdrawal
	case models.PCTransactionTransfer:
		result.OperationType = models.OpTransfer
	case models.PCTransactionPayment:
		result.OperationType = models.OpPayment
	case models.PCTransactionRefund:
		result.OperationType = models.OpRefund
	case models.PCTransactionDeposit:
		result.OperationType = models.OpDeposit
	default:
		result.OperationType = models.OpUnknown
	}

	return result
}

// formatAmount форматирует сумму для ISO8583 с учетом валюты
func (iso *ISO8583) formatAmount(amount float64, currency string) (string, error) {
	info, exists := models.CurrencyMap[currency]
	if !exists {
		return nilStr, fmt.Errorf("failed to format amount: %f", amount)
	}

	// Преобразуем в минимальные единицы
	multiplier := math.Pow(10, float64(info.Decimals))

	// Используем точное преобразование через int64 чтобы избежать ошибок float
	amountInMinor := int64(math.Round(amount * multiplier))

	// Форматируем как 12-значное число с ведущими нулями
	formatted := fmt.Sprintf("%012d", amountInMinor)

	return formatted, nil
}

// generateProcessingCode генерирует Processing Code на основе транзакции и направления
func (iso *ISO8583) generateProcessingCode(transaction *models.Transaction) string {
	// Определяем тип операции (первые 2 цифры)
	var transactionType string

	switch transaction.Type {
	case "purchase":
		transactionType = models.PCTransactionPurchase
	case "withdrawal":
		transactionType = models.PCTransactionWithdrawal
	case "transfer":
		transactionType = models.PCTransactionTransfer
	case "payment":
		transactionType = models.PCTransactionPayment
	case "refund":
		transactionType = models.PCTransactionRefund
	case "deposit":
		transactionType = models.PCTransactionDeposit
	default:
		transactionType = models.PCTransactionTransfer
	}

	// Определяем тип счета отправителя (3-4 цифры)
	var fromAccountType string
	switch {
	case transaction.SenderAccountCode != "":
		fromAccountType = models.PCAccountElectronic
	case transaction.SenderCardNumber != "":
		fromAccountType = models.PCAccountCreditCard
	case transaction.SenderPhone != "":
		fromAccountType = models.PCAccountElectronic
	default:
		fromAccountType = models.PCAccountUniversal
	}

	// Определяем тип счета получателя (5-6 цифры)
	var toAccountType string
	switch {
	case transaction.RecipientAccountCode != "":
		toAccountType = models.PCAccountElectronic
	case transaction.RecipientCardNumber != "":
		toAccountType = models.PCAccountCreditCard
	case transaction.RecipientPhone != "":
		toAccountType = models.PCAccountElectronic
	default:
		toAccountType = models.PCAccountUniversal
	}
	// Формируем Processing Code: AABBCC
	processingCode := transactionType + fromAccountType + toAccountType

	return processingCode
}

func (iso *ISO8583) getErrorCode(reason string) string {
	switch {
	case strings.Contains(reason, "insufficient"):
		return models.RCInsufficientFunds
	case strings.Contains(reason, "timeout"):
		return models.RCTimeout
	case strings.Contains(reason, "invalid_card"):
		return models.RCInvalidCard
	case strings.Contains(reason, "pin"):
		return models.RCInvalidPIN
	default:
		return models.RCSystemError
	}
}
