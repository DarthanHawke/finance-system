package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"api-gateway-service/internal/models"
)

type TransactionManager interface {
	GetTransaction(ctx context.Context, req *models.GetTransactionRequest) (models.GetTransactionResponse, error)
	GetTransactions(ctx context.Context, req *models.GetTransactionsRequest) (models.GetTransactionsResponse, error)
}

type ProducerManager interface {
	Produce(ctx context.Context, topic, key string, event any) error
}

type RedisManager interface {
	IsDuplicate(ctx context.Context, key string) (bool, error)
	Remove(ctx context.Context, key string) error
}

type TransactionHandler struct {
	transactionClient TransactionManager
	producer          ProducerManager
	deduplicator      RedisManager
	logger            *zap.Logger
}

func NewTransactionHandler(
	transactionClient TransactionManager,
	producer ProducerManager,
	deduplicator RedisManager,
	logger *zap.Logger,
) *TransactionHandler {
	return &TransactionHandler{
		transactionClient: transactionClient,
		producer:          producer,
		deduplicator:      deduplicator,
		logger:            logger,
	}
}

func (h *TransactionHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Создание транзакции (асинхронное через Kafka)
	r.Post("/create", h.CreateTransaction)

	// Получение информации о транзакциях (синхронное через gRPC)
	r.Get("/{transaction_id}", h.GetTransaction)
	r.Get("/account/{account_code}", h.GetTransactions)

	return r
}

// CreateTransaction создает новую транзакцию (асинхронно через Kafka)
// @Summary Создать транзакцию
// @Description Создает новую транзакцию между счетами
// @Tags transaction
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param request body models.CreateTransactionRequest true "Данные для создания транзакции"
// @Success 202 {object} models.CreateTransactionResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /transaction/create [post]
func (h *TransactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTransactionRequest

	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "invalid request body: " + err.Error()})
		return
	}

	// Валидация запроса
	if err := h.validateTransactionRequest(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Получение ключа идемпотентности
	idempotencyKey := h.getIdempotencyKey(r)
	if idempotencyKey == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "idempotency key is required"})
		return
	}

	// Проверка на дубликат
	isDuplicate, err := h.deduplicator.IsDuplicate(r.Context(), idempotencyKey)
	if err != nil {
		h.logger.Error("failed to check idempotency",
			zap.String("idempotency_key", idempotencyKey),
			zap.Error(err),
		)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "internal server error"})
		return
	}

	if isDuplicate {
		h.logger.Info("duplicate request detected",
			zap.String("idempotency_key", idempotencyKey),
		)
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, models.ErrorResponse{Error: "duplicate request already processed"})
		return
	}

	// Генерация ID транзакции
	transactionID := uuid.New()

	// Создание события
	event := h.createTransactionEvent(transactionID, &req)

	// Отправка в Kafka
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err = h.producer.Produce(
		ctx,
		models.TopicTransaction,
		event.PartitionKey,
		event,
	)

	if err != nil {
		h.logger.Error("failed to produce to kafka",
			zap.Error(err),
			zap.String("transaction_id", transactionID.String()),
		)

		// Удаляем ключ идемпотентности при ошибке
		_ = h.deduplicator.Remove(r.Context(), idempotencyKey)

		render.Status(r, http.StatusServiceUnavailable)
		render.JSON(w, r, models.ErrorResponse{Error: "payment service temporarily unavailable"})
		return
	}

	// Успешный ответ
	render.Status(r, http.StatusAccepted)
	render.JSON(w, r, models.CreateTransactionResponse{
		TransactionID: transactionID.String(),
		Status:        string(models.TransactionStatusPending),
		Message:       "Transaction accepted for processing",
	})
}

// GetTransaction получает информацию о транзакции по ID
// @Summary Получить транзакцию
// @Description Возвращает информацию о транзакции по её ID
// @Tags transaction
// @Produce json
// @Security CookieAuth
// @Param transaction_id path string true "ID транзакции"
// @Success 200 {object} models.GetTransactionResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /transaction/{transaction_id} [get]
func (h *TransactionHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	transactionIDStr := chi.URLParam(r, "transaction_id")
	if transactionIDStr == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "transaction_id is required"})
		return
	}

	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "invalid transaction_id format"})
		return
	}

	response, err := h.transactionClient.GetTransaction(r.Context(), &models.GetTransactionRequest{
		ID: transactionID,
	})
	if err != nil {
		h.logger.Error("failed to get transaction",
			zap.String("transaction_id", transactionIDStr),
			zap.Error(err),
		)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to get transaction: " + err.Error()})
		return
	}

	render.JSON(w, r, response)
}

// GetTransactions получает список транзакций по коду счета
// @Summary Получить транзакции счета
// @Description Возвращает список транзакций для указанного счета с пагинацией
// @Tags transaction
// @Produce json
// @Security CookieAuth
// @Param account_code path string true "Код счета"
// @Param limit query int false "Лимит записей" default(10)
// @Param offset query int false "Смещение" default(0)
// @Success 200 {object} models.GetTransactionsResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /transaction/account/{account_code} [get]
func (h *TransactionHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	accountCode := chi.URLParam(r, "account_code")
	if accountCode == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "account_code is required"})
		return
	}

	// Параметры пагинации
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	response, err := h.transactionClient.GetTransactions(r.Context(), &models.GetTransactionsRequest{
		AccountCode: accountCode,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		h.logger.Error("failed to get transactions",
			zap.String("account_code", accountCode),
			zap.Error(err),
		)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to get transactions: " + err.Error()})
		return
	}

	render.JSON(w, r, response)
}

// createTransactionEvent — упаковывает запрос в Event для Transaction Service
func (h *TransactionHandler) createTransactionEvent(
	transactionID uuid.UUID,
	req *models.CreateTransactionRequest,
) *models.Event {
	transaction := &models.Transaction{
		ID:                   transactionID,
		Amount:               req.Amount,
		Currency:             req.Currency,
		SenderAccountCode:    req.SenderAccountCode,
		RecipientAccountCode: req.RecipientAccountCode,
		Description:          req.Description,
		Status:               models.TransactionStatusPending,
		CreatedAt:            time.Now(),
	}

	payload, _ := json.Marshal(transaction)

	return &models.Event{
		ID:            uuid.New(),
		TransactionID: transactionID,
		PartitionKey:  h.partitionKey(req.SenderAccountCode),
		Type:          models.EventTransactionRequest,
		Status:        models.EventStatusPending,
		Source:        models.Source,
		CreatedAt:     time.Now(),
		Payload:       payload,
	}
}

// validateTransactionRequest — структурная валидация запроса
func (h *TransactionHandler) validateTransactionRequest(req *models.CreateTransactionRequest) error {
	if req.SenderAccountCode == "" {
		return fmt.Errorf("sender_account_code is required")
	}
	if req.RecipientAccountCode == "" {
		return fmt.Errorf("recipient_account_code is required")
	}
	if req.SenderAccountCode == req.RecipientAccountCode {
		return fmt.Errorf("sender and recipient accounts must be different")
	}
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if req.Currency == "" {
		return fmt.Errorf("currency is required")
	}
	if req.Currency != "USD" && req.Currency != "EUR" && req.Currency != "RUB" {
		return fmt.Errorf("invalid currency, supported: USD, EUR, RUB")
	}
	return nil
}

// getIdempotencyKey — пока что просто загулшка, а потом буду из middleware поддтягивать
func (h *TransactionHandler) getIdempotencyKey(r *http.Request) (userID string) {
	// Заглушка: генерируем из user_id + случайного числа
	if uid := r.Context().Value("user_id"); uid != nil {
		userID = uid.(string)
	}
	return fmt.Sprintf("%s-%d", userID, time.Now().UnixNano())
}

// partitionKey — хэш код для ключа партиции из номера счета
func (h *TransactionHandler) partitionKey(accountCode string) string {
	hash := sha256.Sum256([]byte(accountCode))
	return "acc:" + hex.EncodeToString(hash[:8])
}
