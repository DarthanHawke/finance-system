package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"external-payment-service/internal/models"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type ExternalPaymentHandler struct {
	service ExternalPaymentService
	logger  *zap.Logger
}

type ExternalPaymentService interface {
	CreateExternalPayment(ctx context.Context, req *models.CreateExternalPaymentRequest) (*models.CreateExternalPaymentResponse, error)
}

func NewExternalPaymentHandler(service ExternalPaymentService, logger *zap.Logger) *ExternalPaymentHandler {
	return &ExternalPaymentHandler{
		service: service,
		logger:  logger.With(zap.String("component", "http_handler")),
	}
}

func (h *ExternalPaymentHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/api/v1/external-payment", h.HandleCreateExternalPayment)
	return r
}

// HandleCreateExternalPayment обрабатывает POST /api/v1/external-payment
// @Summary      Создать внешний платёж
// @Description  Инициирует внешний платёж и отправляет его в Transaction Service через Kafka
// @Tags         external-payment
// @Accept       multipart/form-data
// @Param        sender_account_code    formData  string  true   "Код счета отправителя"
// @Param        recipient_account_code formData  string  true   "Код счета получателя"
// @Param        amount                 formData  number  true   "Сумма платежа"
// @Param        currency               formData  string  true   "Валюта платежа"            Enums(RUB, USD, EUR) default(RUB)
// @Param        description            formData  string  false  "Описание платежа"
// @Success      201  {object}  models.CreateExternalPaymentResponse  "Платёж успешно создан"
// @Failure      400  {string}  string  "Неверный запрос"
// @Failure      405  {string}  string  "Метод не разрешён"
// @Failure      500  {string}  string  "Внутренняя ошибка сервера"
// @Router       /api/v1/external-payment [post]
func (h *ExternalPaymentHandler) HandleCreateExternalPayment(w http.ResponseWriter, r *http.Request) {
	var req models.CreateExternalPaymentRequest

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") || strings.Contains(contentType, "application/x-www-form-urlencoded") {
		// Парсим form-data
		r.ParseMultipartForm(10 << 20) // 10 MB max
		req.SenderAccountCode = r.FormValue("sender_account_code")
		req.RecipientAccountCode = r.FormValue("recipient_account_code")
		req.Currency = r.FormValue("currency")
		req.Description = r.FormValue("description")
		if amountStr := r.FormValue("amount"); amountStr != "" {
			if amount, err := strconv.ParseFloat(amountStr, 64); err == nil {
				req.Amount = amount
			}
		}
	} else {
		// Парсим JSON
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.logger.Error("failed to decode request", zap.Error(err))
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	}

	// Простая валидация
	if req.SenderAccountCode == "" || req.RecipientAccountCode == "" || req.Amount <= 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if req.Currency == "" {
		req.Currency = "RUB"
	}

	resp, err := h.service.CreateExternalPayment(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create external payment", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
