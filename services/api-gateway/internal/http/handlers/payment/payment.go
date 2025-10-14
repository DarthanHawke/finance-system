package payment

import (
	"client-service/internal/models"
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type PaymentService interface {
	Transfer(ctx context.Context, req models.TransferRequest) (*models.Payment, error)
	Deposit(ctx context.Context, req models.DepositRequest) (uuid.UUID, error)
	GetPayment(ctx context.Context, req models.PaymentRequest) (*models.Payment, error)
	GetAllPayment(ctx context.Context, userID uuid.UUID) ([]models.Payment, error)
	UpdateStatusPayment(ctx context.Context, req models.UpdateStatusRequest) error
	CancelPayment(ctx context.Context, req models.PaymentRequest) error
	ConvertCurrency(ctx context.Context, req models.ConvertCurrencyRequest) (uuid.UUID, error)
	GetOperationHistory(ctx context.Context, req models.OperationHistoryRequest) ([]models.BalanceOperation, error)
	UpdateCurrencyRate(ctx context.Context, req models.UpdateCurrencyRateRequest) error
	GetCurrencyRate(ctx context.Context, req models.GetCurrencyRateRequest) (float64, error)
}

type PaymentHandler struct {
	paymentService PaymentService
}

func NewPaymentHandler(paymentService PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

func (h *PaymentHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/transfer", h.Transfer)
	r.Post("/deposit", h.Deposit)
	r.Get("/get", h.GetAllPayments)
	r.Get("/{paymentID}", h.GetPayment)
	r.Patch("/{paymentID}/update", h.UpdatePaymentStatus)
	r.Patch("/{paymentID}/cancel", h.CancelPayment)
	r.Post("/convert", h.ConvertCurrency)
	r.Get("/history", h.GetOperationHistory)
	r.Post("/currency-rate", h.UpdateCurrencyRate)
	r.Get("/currency-rate/get", h.GetCurrencyRate)
	return r
}

// Transfer Перевод
// @Summary Перевести на другой счет
// @Description Создаеёт перевод между указанными счетами
// @Tags payments
// @Accept x-www-form-urlencoded
// @Produce json
// @Security CookieAuth
// @Param sender formData string true "Платёжный счет"
// @Param receiver formData string true "Номер счета получателя"
// @Param amount formData number true "Сумма платежа"
// @Param currency formData string true "Валюта" Enums(USD,EUR,RUB)
// @Param description formData string false "Описание платежа"
// @Success 201 {object} models.Payment
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /payments/transfer [post]
func (h *PaymentHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to parse form data"})
		return
	}

	// Получаем значения из формы
	sender := r.FormValue("sender")
	receiver := r.FormValue("receiver")
	amountStr := r.FormValue("amount")
	description := r.FormValue("description")

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{
			Error: "amount must be a positive number",
		})
		return
	}

	req := models.TransferRequest{
		Sender:      sender,
		Receiver:    receiver,
		Amount:      amount,
		Description: description,
	}

	payment, err := h.paymentService.Transfer(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{
			Error: "failed to create payment",
		})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, payment)
}

// Deposit пополняет счет
// @Summary Пополнить счет
// @Description Пополняет счет на указанную сумму
// @Tags payments
// @Accept x-www-form-urlencoded
// @Produce json
// @Security CookieAuth
// @Param account_id formData string true "Номер счета"
// @Param amount formData number true "Сумма платежа"
// @Param description formData string false "Описание платежа"
// @Success 200 {object} models.DepositResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /payments/deposit [post]
func (h *PaymentHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to parse form data"})
		return
	}
	// Получаем значения из формы
	accountID := r.FormValue("account_id")
	amountStr := r.FormValue("amount")
	description := r.FormValue("description")

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{
			Error: "amount must be a positive number",
		})
		return
	}

	req := models.DepositRequest{
		AccountID:   accountID,
		Amount:      amount,
		Description: description,
	}

	paymentID, err := h.paymentService.Deposit(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to deposit"})
		return
	}

	render.JSON(w, r, models.DepositResponse{PaymentID: paymentID})
}

// GetPayment Возвращает информацию о платеже
// @Summary Получить платеж
// @Description Возвращает информацию о конкретном платеже
// @Tags payments
// @Produce json
// @Security CookieAuth
// @Param paymentID path string true "ID платежа"
// @Success 200 {object} models.Payment
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /payments/{paymentID} [get]
func (h *PaymentHandler) GetPayment(w http.ResponseWriter, r *http.Request) {
	paymentIDStr := chi.URLParam(r, "paymentID")
	paymentID, err := uuid.Parse(paymentIDStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{
			Error: "invalid payment ID format",
		})
		return
	}

	req := models.PaymentRequest{
		PaymentID: paymentID,
	}

	payment, err := h.paymentService.GetPayment(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{
			Error: "failed to get payment",
		})
		return
	}

	if payment == nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, models.ErrorResponse{
			Error: "payment not found",
		})
		return
	}

	render.JSON(w, r, payment)
}

// GetAllPayments Возвращает все платежи
// @Summary Получить все платежи
// @Description Возвращает список всех платежей
// @Tags payments
// @Produce json
// @Security CookieAuth
// @Param userID query string false "ID пользователя"
// @Success 200 {array} models.Payment
// @Failure 500 {object} models.ErrorResponse
// @Router /payments/get [get]
func (h *PaymentHandler) GetAllPayments(w http.ResponseWriter, r *http.Request) {
	var userID uuid.UUID
	var err error

	// Получаем параметр из URL
	userIDStr := r.URL.Query().Get("userID")

	// Если параметр передан - парсим его
	if userIDStr != "" {
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, models.ErrorResponse{
				Error: "invalid user ID format",
			})
			return
		}
	}

	payments, err := h.paymentService.GetAllPayment(r.Context(), userID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{
			Error: "failed to get payments",
		})
		return
	}

	render.JSON(w, r, payments)
}

// UpdatePaymentStatus Обновляет статус платежа
// @Summary Обновить статус платежа
// @Description Обновляет статус указанного платежа
// @Tags payments
// @Accept x-www-form-urlencoded
// @Produce json
// @Security CookieAuth
// @Param paymentID path string true "ID платежа"
// @Param status formData string true "Статус" Enums(completed,refunded)
// @Success 200
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /payments/{paymentID}/update [patch]
func (h *PaymentHandler) UpdatePaymentStatus(w http.ResponseWriter, r *http.Request) {
	paymentIDStr := chi.URLParam(r, "paymentID")
	paymentID, err := uuid.Parse(paymentIDStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{
			Error: "invalid payment ID format",
		})
		return
	}

	if err := r.ParseForm(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to parse form data"})
		return
	}

	status := r.FormValue("status")

	req := models.UpdateStatusRequest{
		PaymentID: paymentID,
		Status:    status,
	}

	if err := h.paymentService.UpdateStatusPayment(r.Context(), req); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{
			Error: "failed to update payment status",
		})
		return
	}

	render.Status(r, http.StatusOK)
}

// CancelPayment Отменяет платеж
// @Summary Отменить платеж
// @Description Отменяет указанный платеж
// @Tags payments
// @Security CookieAuth
// @Param paymentID path string true "ID платежа"
// @Success 200
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /payments/{paymentID}/cancel [patch]
func (h *PaymentHandler) CancelPayment(w http.ResponseWriter, r *http.Request) {
	paymentIDStr := chi.URLParam(r, "paymentID")
	paymentID, err := uuid.Parse(paymentIDStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{
			Error: "invalid payment ID format",
		})
		return
	}

	req := models.PaymentRequest{
		PaymentID: paymentID,
	}

	if err := h.paymentService.CancelPayment(r.Context(), req); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{
			Error: "failed to cancel payment",
		})
		return
	}

	render.Status(r, http.StatusOK)
}

// ConvertCurrency конвертирует валюту
// @Summary Конвертировать валюту
// @Description Конвертирует указанную сумму из одной валюты в другую
// @Tags payments
// @Accept x-www-form-urlencoded
// @Produce json
// @Security CookieAuth
// @Param sender formData string true "Номер счёта с валютой для обмена"
// @Param receiver formData string true "Номер счёта для зачисления новой валюты"
// @Param amount formData number true "Сумма платежа"
// @Param description formData string false "Описание платежа"
// @Success 200 {object} models.ConvertCurrencyResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /payments/convert [post]
func (h *PaymentHandler) ConvertCurrency(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to parse form data"})
		return
	}

	// Получаем значения из формы
	fromAccountID := r.FormValue("sender")
	toAccountID := r.FormValue("receiver")
	amountStr := r.FormValue("amount")
	description := r.FormValue("description")

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{
			Error: "amount must be a positive number",
		})
		return
	}

	req := models.ConvertCurrencyRequest{
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		Amount:        amount,
		Description:   description,
	}

	operationID, err := h.paymentService.ConvertCurrency(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to convert currency"})
		return
	}

	render.JSON(w, r, models.ConvertCurrencyResponse{OperationID: operationID})
}

// GetOperationHistory возвращает историю операций по счету
// @Summary Получить историю операций
// @Description Возвращает историю операций по указанному счету
// @Tags payments
// @Produce json
// @Security CookieAuth
// @Param account_id query string true "Номер счета"
// @Param limit query int false "Лимит (по умолчанию 10)"
// @Param offset query int false "Смещение (по умолчанию 0)"
// @Success 200 {array} models.BalanceOperation
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /payments/history [get]
func (h *PaymentHandler) GetOperationHistory(w http.ResponseWriter, r *http.Request) {
	accountID := r.URL.Query().Get("account_id")
	if accountID == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "account_id is required"})
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if lInt, err := strconv.Atoi(l); err == nil {
			limit = lInt
		} else {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, models.ErrorResponse{Error: "invalid limit format"})
			return
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if oInt, err := strconv.Atoi(o); err == nil {
			offset = oInt
		} else {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, models.ErrorResponse{Error: "invalid offset format"})
			return
		}
	}
	operations, err := h.paymentService.GetOperationHistory(r.Context(), models.OperationHistoryRequest{
		AccountID: accountID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to get operation history"})
		return
	}

	render.JSON(w, r, operations)
}

// UpdateCurrencyRate обновляет курс валют
// @Summary Обновить курс валют
// @Description Обновляет курс конвертации между валютами (Admin only)
// @Tags payments
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param request body models.UpdateCurrencyRateRequest true "Данные для обновления курса"
// @Success 200
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /payments/currency-rate [post]
func (h *PaymentHandler) UpdateCurrencyRate(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateCurrencyRateRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	if err := h.paymentService.UpdateCurrencyRate(r.Context(), req); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to update currency rate"})
		return
	}

	render.Status(r, http.StatusOK)
}

// GetCurrencyRate возвращает текущий курс обмена между валютами
// @Summary Получить курс валют
// @Description Возвращает текущий курс обмена между указанными валютами
// @Tags payments
// @Produce json
// @Security CookieAuth
// @Param from query string true "Исходная валюта (например: USD)"
// @Param to query string true "Целевая валюта (например: EUR)"
// @Success 200 {object} models.CurrencyRateResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /payments/currency-rate/get [get]
func (h *PaymentHandler) GetCurrencyRate(w http.ResponseWriter, r *http.Request) {
	fromCurrency := r.URL.Query().Get("from")
	toCurrency := r.URL.Query().Get("to")

	if fromCurrency == "" || toCurrency == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{
			Error: "both 'from' and 'to' currency parameters are required",
		})
		return
	}

	req := models.GetCurrencyRateRequest{
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
	}

	rate, err := h.paymentService.GetCurrencyRate(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{
			Error: "failed to get currency rate",
		})
		return
	}

	render.JSON(w, r, models.CurrencyRateResponse{
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		Rate:         rate,
	})
}
