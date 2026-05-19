package http

import (
	"context"
	"encoding/json"
	"net/http"

	"external-payment-service/internal/models"

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

// RegisterRoutes регистрирует HTTP ручки
func (h *ExternalPaymentHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/external-payment", h.HandleCreateExternalPayment)
}

// HandleCreateExternalPayment обрабатывает POST /api/v1/external-payment
func (h *ExternalPaymentHandler) HandleCreateExternalPayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateExternalPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
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
