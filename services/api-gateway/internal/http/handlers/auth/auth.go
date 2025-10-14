package auth

import (
	authCookie "client-service/internal/http/cookie"
	"client-service/internal/models"
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type AuthHandler struct {
	service AuthService
}

func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

type AuthService interface {
	RefreshSession(ctx context.Context) (*models.UserSession, error)
	Register(ctx context.Context, reg models.RegisterRequest) (*models.UserSession, error)
	Login(ctx context.Context, log models.LoginRequest) (*models.UserSession, error)
	Logout(ctx context.Context, userID, sessionID uuid.UUID) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
	GetAllSessions(ctx context.Context, userID uuid.UUID) ([]models.Session, error)
}

func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.RefreshSession)
	r.Post("/logout", h.Logout)
	r.Post("/logout-all", h.LogoutAll)
	r.Get("/sessions", h.GetAllSessions)

	return r
}

// Register Регистрация нового пользователя
// @Summary Зарегистрировать нового пользователя
// @Description Регистрирует нового пользователя и создаёт для него сессию(сохраняет токены в cookies)
// @Tags auth
// @Accept x-www-form-urlencoded
// @Produce json
// @Param name formData string true "User name"
// @Param email formData string true "User email"
// @Param password formData string true "User password"
// @Success 200 {object} models.UserSession
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid form data"})
		return
	}

	req := models.RegisterRequest{
		FullName: r.FormValue("name"),
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
	}

	session, err := h.service.Register(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	authCookie.SetTokenCookies(w, session.AccessToken, session.RefreshToken)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "authenticated"})
}

// Login Вход для зарегестрированного пользователя
// @Summary Войти используя данные пользователя
// @Description Валидирует входные данные и создаёт сессию(сохраняет токены в cookies)
// @Tags auth
// @Accept x-www-form-urlencoded
// @Produce json
// @Param email formData string true "User email"
// @Param password formData string true "User password"
// @Success 200 {object} models.UserSession
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid form data"})
		return
	}

	req := models.LoginRequest{
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
	}

	session, err := h.service.Login(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "invalid credentials"})
		return
	}

	authCookie.SetTokenCookies(w, session.AccessToken, session.RefreshToken)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "authenticated"})
}

// RefreshSession Обновление сессии
// @Summary Обновить сессию
// @Description Обновляет сессию(токены), для получения нового, не просроченного Access токена
// @Tags auth
// @Accept json
// @Produce json
// @Security CookieAuth
// @Success 200 {object} models.UserSession
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshSession(w http.ResponseWriter, r *http.Request) {
	session, err := h.service.RefreshSession(r.Context())
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "refresh failed"})
		return
	}

	authCookie.SetTokenCookies(w, session.AccessToken, session.RefreshToken)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "authenticated"})
}

// Logout Выход из сессии
// @Summary Выйти из текущей сессии
// @Description Удаляет текущую сессию(добавляет токен в Blacklist до истечения его срока жизни, удаляет данные о сессии из базы данных)
// @Tags auth
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param userID query string false "ID пользователя"
// @Param sessionID query string false "ID сессии"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
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

	var sessionID uuid.UUID
	sessionIDStr := r.URL.Query().Get("sessionID")
	if sessionIDStr != "" {
		sessionID, err = uuid.Parse(sessionIDStr)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, models.ErrorResponse{
				Error: "invalid session ID format",
			})
			return
		}
	}

	err = h.service.Logout(r.Context(), userID, sessionID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "logout failed"})
		return
	}

	render.JSON(w, r, map[string]string{"status": "logged out"})
}

// LogoutAll Выход со всех устройств
// @Summary Выйти со всех устройств
// @Description Удаляет все сессии текущего пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param userID query string false "ID пользователя"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/logout-all [post]
func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
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

	err = h.service.LogoutAll(r.Context(), userID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "logout failed"})
		return
	}

	render.JSON(w, r, map[string]string{"status": "logged out from all devices"})
}

// GetAllSessions Получить все сессии
// @Summary Получить все сессии пользователя
// @Description Возвращает все сессии пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param userID query string false "ID пользователя"
// @Success 200 {array} models.Session
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/sessions [get]
func (h *AuthHandler) GetAllSessions(w http.ResponseWriter, r *http.Request) {
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

	sessions, err := h.service.GetAllSessions(r.Context(), userID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "cant get sessions"})
		return
	}

	render.JSON(w, r, sessions)
}
