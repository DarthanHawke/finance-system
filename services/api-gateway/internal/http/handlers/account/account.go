package account

import (
	"client-service/internal/models"
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type AccountService interface {
	CreateAccount(ctx context.Context, req models.CreateAccountRequest) (string, error)
	GetAccount(ctx context.Context, req models.GetAccountRequest) (*models.CurrencyAccount, error)
	GetUserAccounts(ctx context.Context, req models.GetUserAccountsRequest) ([]models.CurrencyAccount, error)
	GetBalance(ctx context.Context, req models.GetBalanceRequest) (float64, error)
}

type AccountHandler struct {
	accountService AccountService
}

func NewAccountHandler(accountService AccountService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
	}
}

func (h *AccountHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/create", h.CreateAccount)
	r.Get("/get", h.GetUserAccounts)
	r.Get("/{account_id}", h.GetAccount)
	r.Get("/{account_id}/balance", h.GetBalance)
	return r
}

// CreateAccount создает новый счет
// @Summary Создать счет
// @Description Создает новый счет с указанной валютой и именем
// @Tags account
// @Accept x-www-form-urlencoded
// @Produce json
// @Security CookieAuth
// @Param currency formData string true "Валюта" Enums(USD,EUR,RUB)
// @Param account_name formData string true "Имя счета"
// @Success 201 {object} models.CreateAccountResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /account/create [post]
func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to parse form data"})
		return
	}

	currency := r.FormValue("currency")
	name := r.FormValue("account_name")
	req := models.CreateAccountRequest{
		Currency: currency,
		Name:     name,
	}

	accountID, err := h.accountService.CreateAccount(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to create account"})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, models.CreateAccountResponse{AccountID: accountID})
}

// GetAccount возвращает информацию о счете
// @Summary Получить информацию о счете
// @Description Возвращает информацию о счете по его ID
// @Tags account
// @Produce json
// @Security CookieAuth
// @Param account_id path string true "Номер счета"
// @Success 200 {object} models.CurrencyAccount
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /account/{account_id} [get]
func (h *AccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "account_id")
	if accountID == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "accountID is required"})
		return
	}

	account, err := h.accountService.GetAccount(r.Context(), models.GetAccountRequest{
		AccountID: accountID,
	})
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to get account"})
		return
	}

	render.JSON(w, r, account)
}

// GetUserAccounts возвращает список счетов пользователя
// @Summary Получить список счетов пользователя
// @Description Возвращает список всех счетов текущего пользователя
// @Tags account
// @Produce json
// @Security CookieAuth
// @Param user_id query string false "ID пользователя"
// @Success 200 {array} models.CurrencyAccount
// @Failure 500 {object} models.ErrorResponse
// @Router /account/get [get]
func (h *AccountHandler) GetUserAccounts(w http.ResponseWriter, r *http.Request) {
	var userID uuid.UUID
	var err error
	userIDStr := r.URL.Query().Get("user_id")
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

	req := models.GetUserAccountsRequest{
		UserID: userID,
	}
	accounts, err := h.accountService.GetUserAccounts(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to get user accounts"})
		return
	}

	render.JSON(w, r, accounts)
}

// GetBalance возвращает баланс по номеру счета
// @Summary Получить баланс
// @Description Возвращает баланс счета по номеру счета
// @Tags account
// @Produce json
// @Security CookieAuth
// @Param account_id path string true "Номер счета"
// @Success 200 {object} models.GetBalanceResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /account/{account_id}/balance [get]
func (h *AccountHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "account_id")
	if accountID == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "accountID is required"})
		return
	}

	balance, err := h.accountService.GetBalance(r.Context(), models.GetBalanceRequest{
		AccountID: accountID,
	})
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to get balance"})
		return
	}

	render.JSON(w, r, models.GetBalanceResponse{Balance: balance})
}
