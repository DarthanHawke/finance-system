package handlers

import (
	"api-gateway-service/internal/models"
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type AccountManager interface {
	CreateAccount(ctx context.Context, req *models.CreateAccountRequest) error
	GetAccount(ctx context.Context, req *models.GetAccountRequest) (models.GetAccountResponse, error)
	GetAccounts(ctx context.Context, req *models.GetAccountsRequest) (models.GetAccountsResponse, error)
	BlockAccount(ctx context.Context, req *models.UpdateAccountRequest) error
	CloseAccount(ctx context.Context, req *models.UpdateAccountRequest) error
}

type AccountHandler struct {
	account AccountManager
}

func NewAccountHandler(account AccountManager) *AccountHandler {
	return &AccountHandler{
		account: account,
	}
}

func (h *AccountHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/create", h.CreateAccount)
	r.Get("/get", h.GetAccounts)
	r.Get("/{code}", h.GetAccount)
	r.Put("/{code}/block", h.BlockAccount)
	r.Delete("/{code}/close", h.CloseAccount)

	return r
}

// CreateAccount создает новый счет
// @Summary Создать счет
// @Description Создает новый счет с указанной валютой и именем
// @Tags account
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param request body models.CreateAccountRequest true "Данные для создания счета"
// @Success 201 {object} models.SuccessResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /account/create [post]
func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req models.CreateAccountRequest

	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to parse request body: " + err.Error()})
		return
	}

	// Валидация
	if req.Name == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "account name is required"})
		return
	}

	if req.Currency == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "currency is required"})
		return
	}

	if req.UserID == uuid.Nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "user ID is required"})
		return
	}

	err := h.account.CreateAccount(r.Context(), &req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to create account: " + err.Error()})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, models.SuccessResponse{Message: "account created successfully"})
}

// GetAccount возвращает информацию о счете
// @Summary Получить информацию о счете
// @Description Возвращает информацию о счете по его коду
// @Tags account
// @Produce json
// @Security CookieAuth
// @Param code path string true "Код счета"
// @Success 200 {object} models.GetAccountResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /account/{code} [get]
func (h *AccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "account code is required"})
		return
	}

	account, err := h.account.GetAccount(r.Context(), &models.GetAccountRequest{
		Code: code,
	})
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to get account: " + err.Error()})
		return
	}

	render.JSON(w, r, account)
}

// GetAccounts возвращает список счетов пользователя
// @Summary Получить список счетов пользователя
// @Description Возвращает список всех счетов пользователя с пагинацией
// @Tags account
// @Produce json
// @Security CookieAuth
// @Param user_id query string true "ID пользователя"
// @Param limit query int false "Лимит записей" default(10)
// @Param offset query int false "Смещение" default(0)
// @Success 200 {object} models.GetAccountsResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /account/get [get]
func (h *AccountHandler) GetAccounts(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "user_id is required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "invalid user ID format"})
		return
	}

	// Параметры пагинации
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		// TODO: парсинг limit
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		// TODO: парсинг offset
	}

	req := models.GetAccountsRequest{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	}

	accounts, err := h.account.GetAccounts(r.Context(), &req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to get user accounts: " + err.Error()})
		return
	}

	render.JSON(w, r, accounts)
}

// BlockAccount блокирует счет
// @Summary Заблокировать счет
// @Description Блокирует счет по его коду
// @Tags account
// @Produce json
// @Security CookieAuth
// @Param code path string true "Код счета"
// @Success 200 {object} models.SuccessResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /account/{code}/block [put]
func (h *AccountHandler) BlockAccount(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "account code is required"})
		return
	}

	err := h.account.BlockAccount(r.Context(), &models.UpdateAccountRequest{
		Code: code,
	})
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to block account: " + err.Error()})
		return
	}

	render.JSON(w, r, models.SuccessResponse{Message: "account blocked successfully"})
}

// CloseAccount закрывает счет
// @Summary Закрыть счет
// @Description Закрывает счет по его коду
// @Tags account
// @Produce json
// @Security CookieAuth
// @Param code path string true "Код счета"
// @Success 200 {object} models.SuccessResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /account/{code}/close [delete]
func (h *AccountHandler) CloseAccount(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "account code is required"})
		return
	}

	err := h.account.CloseAccount(r.Context(), &models.UpdateAccountRequest{
		Code: code,
	})
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to close account: " + err.Error()})
		return
	}

	render.JSON(w, r, models.SuccessResponse{Message: "account closed successfully"})
}
