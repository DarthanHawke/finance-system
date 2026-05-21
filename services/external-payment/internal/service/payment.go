package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"

	"external-payment-service/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ExternalPaymentService struct {
	producer Producer
	logger   *zap.Logger
	rng      *rand.Rand
}

type Producer interface {
	Produce(ctx context.Context, topic, key string, event any) error
}

func NewExternalPaymentService(producer Producer, logger *zap.Logger) *ExternalPaymentService {
	return &ExternalPaymentService{
		producer: producer,
		logger:   logger.With(zap.String("component", "external_payment_service")),
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// generateSTAN генерирует STAN
func generateSTAN() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

// generateRRN генерирует RRN
func generateRRN() string {
	return fmt.Sprintf("%012d", rand.Int63n(1000000000000))
}

// generateAuthID генерирует код авторизации
func generateAuthID() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func NewEvent(transactionID uuid.UUID, eventType string, payload []byte) *models.Event {
	return &models.Event{
		ID:            uuid.New(),
		TransactionID: transactionID,
		Type:          eventType,
		Status:        "pending",
		Source:        models.Source,
		CreatedAt:     time.Now(),
		Payload:       payload,
	}
}

// CreateExternalPayment создать внешний платёж
func (s *ExternalPaymentService) CreateExternalPayment(
	ctx context.Context,
	req *models.CreateExternalPaymentRequest,
) (*models.CreateExternalPaymentResponse, error) {
	const op = "service.CreateExternalPayment"

	logger := s.logger.With(zap.String("op", op))

	transactionID := uuid.New()

	logger.Info("creating external payment",
		zap.String("sender", req.SenderAccountCode),
		zap.String("recipient", req.RecipientAccountCode),
		zap.Float64("amount", req.Amount),
	)

	// Маппим валюту в числовой код ISO8583
	currencyCode, err := mapCurrencyToISO(req.Currency)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Формируем ISO8583 запрос
	isoMsg := &models.ISO8583Message{
		MTI:            models.MTIFinancialRequest,
		PrimaryAccount: req.SenderAccountCode,
		ProcessingCode: "216060",
		Amount:         formatISOAmount(req.Amount, req.Currency),
		STAN:           generateSTAN(),
		RRN:            generateRRN(),
		AccountID1:     req.SenderAccountCode,
		AccountID2:     req.RecipientAccountCode,
		Currency:       currencyCode,
	}

	payload, _ := json.Marshal(&models.ExternalRequestPayload{ISOMessage: isoMsg})

	event := NewEvent(transactionID, models.EventTransactionRequest, payload)

	key := fmt.Sprintf("%s:%s", event.Type, req.SenderAccountCode)
	if err := s.producer.Produce(ctx, models.TopicTransaction, key, event); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	logger.Info("external payment event sent to transaction service")

	return &models.CreateExternalPaymentResponse{
		TransactionID: transactionID,
		Status:        "pending",
		Message:       "External payment initiated",
	}, nil
}

// mapCurrencyToISO маппит буквенный код валюты в числовой ISO8583
func mapCurrencyToISO(currency string) (string, error) {
	switch currency {
	case "RUB":
		return "643", nil
	case "USD":
		return "840", nil
	case "EUR":
		return "978", nil
	default:
		return "", fmt.Errorf("unsupported currency: %s", currency)
	}
}

// formatISOAmount форматирует сумму в 12-значный формат ISO8583 в минимальных единицах
func formatISOAmount(amount float64, currency string) string {
	decimals := 2
	amountInMinor := int64(amount * math.Pow(10, float64(decimals)))
	return fmt.Sprintf("%012d", amountInMinor)
}

// HandleExternalRequest обрабатывает запрос на внешний платёж
func (s *ExternalPaymentService) HandleExternalRequest(ctx context.Context, event *models.Event) error {
	const op = "service.HandleExternalRequest"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("processing external payment request")

	var reqPayload models.ExternalRequestPayload
	if err := json.Unmarshal(event.Payload, &reqPayload); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Эмулируем обработку внешнего платежа
	success := s.randomSuccess()

	isoMsg := reqPayload.ISOMessage
	responseCode := models.RCSuccess
	reason := ""

	if !success {
		// Случайный код ошибки
		responseCode = s.randomErrorCode()
		reason = s.getReasonForCode(responseCode)
		logger.Warn("external payment failed",
			zap.String("response_code", responseCode),
			zap.String("reason", reason),
		)
	}

	// Формируем ISO-ответ
	response := &models.ISO8583Message{
		MTI:             models.MTIFinancialResponse,
		PrimaryAccount:  isoMsg.PrimaryAccount,
		ProcessingCode:  isoMsg.ProcessingCode,
		Amount:          isoMsg.Amount,
		STAN:            isoMsg.STAN,
		RRN:             isoMsg.RRN,
		AuthorizationID: generateAuthID(),
		ResponseCode:    responseCode,
		AccountID1:      isoMsg.AccountID1,
		AccountID2:      isoMsg.AccountID2,
		Currency:        isoMsg.Currency,
	}

	payload := models.ISO8583Payload{
		ISOMessage:   response,
		Success:      success,
		ErrorCode:    responseCode,
		ErrorMessage: reason,
	}

	// Отправляем ответ в Kafka
	responseEvent := &models.Event{
		ID:            uuid.New(),
		TransactionID: event.TransactionID,
		Type:          models.EventExternalResponse,
		Source:        models.Source,
		CreatedAt:     time.Now(),
	}

	responseEvent.Payload, _ = json.Marshal(payload)

	key := fmt.Sprintf("%s:%s", event.TransactionID, isoMsg.AccountID1)
	if err := s.producer.Produce(ctx, models.TopicTransaction, key, responseEvent); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	logger.Info("external payment response sent",
		zap.Bool("success", success),
		zap.String("response_code", responseCode),
	)

	return nil
}

// HandleExternalCommit обрабатывает подтверждение успешной транзакции
func (s *ExternalPaymentService) HandleExternalCommit(ctx context.Context, event *models.Event) error {
	const op = "service.HandleExternalCommit"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("received external commit confirmation")

	var payload models.ISO8583Payload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Просто логируем, никаких ответных действий не требуется
	logger.Info("external transaction committed successfully",
		zap.String("auth_id", payload.ISOMessage.AuthorizationID),
		zap.String("response_code", payload.ISOMessage.ResponseCode),
	)

	return nil
}

// HandleExternalRollback обрабатывает запрос на откат транзакции
func (s *ExternalPaymentService) HandleExternalRollback(ctx context.Context, event *models.Event) error {
	const op = "service.HandleExternalRollback"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("processing external rollback request")

	var payload models.ISO8583Payload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Эмулируем откат (95% успех)
	success := s.randomSuccess()

	responsePayload := models.ISO8583Payload{
		ISOMessage:   payload.ISOMessage,
		Success:      success,
		ErrorCode:    "",
		ErrorMessage: "",
	}

	if !success {
		responsePayload.ErrorCode = models.RCSystemError
		responsePayload.ErrorMessage = "Rollback failed"
		logger.Error("external rollback failed")
	}

	// Отправляем ответ о результате отката
	// (Transaction Service ожидает external.response для external.rollback)
	responseEvent := &models.Event{
		ID:            uuid.New(),
		TransactionID: event.TransactionID,
		Type:          models.EventExternalResponse,
		Source:        models.Source,
		CreatedAt:     time.Now(),
	}

	responseEvent.Payload, _ = json.Marshal(responsePayload)

	key := fmt.Sprintf("%s:%s", event.TransactionID, payload.ISOMessage.AccountID1)
	if err := s.producer.Produce(ctx, models.TopicTransaction, key, responseEvent); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	logger.Info("rollback response sent", zap.Bool("success", success))

	return nil
}

// randomSuccess возвращает true в 95% случаев
func (s *ExternalPaymentService) randomSuccess() bool {
	return s.rng.Intn(100) < 95
}

// randomErrorCode возвращает случайный код ошибки для 5% случаев
func (s *ExternalPaymentService) randomErrorCode() string {
	codes := []string{
		models.RCInsufficientFunds,
		models.RCInvalidTransaction,
		models.RCInvalidCard,
		models.RCReferToIssuer,
		models.RCTransactionNotPermitted,
		models.RCTimeout,
		models.RCSystemError,
	}
	return codes[s.rng.Intn(len(codes))]
}

// getReasonForCode возвращает описание для кода ошибки
func (s *ExternalPaymentService) getReasonForCode(code string) string {
	reasons := map[string]string{
		models.RCInsufficientFunds:       "Insufficient funds",
		models.RCInvalidTransaction:      "Invalid transaction",
		models.RCInvalidCard:             "Invalid card",
		models.RCReferToIssuer:           "Refer to issuer",
		models.RCTransactionNotPermitted: "Transaction not permitted",
		models.RCTimeout:                 "Timeout",
		models.RCSystemError:             "System error",
	}
	if reason, ok := reasons[code]; ok {
		return reason
	}
	return "Unknown error"
}
