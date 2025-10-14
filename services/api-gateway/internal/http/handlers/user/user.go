package user

import (
	"client-service/internal/models"
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type UserService interface {
	GetAllUsers(ctx context.Context, req models.GetAllUsersRequest) ([]models.User, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*models.User, error)
	UpdateName(ctx context.Context, req models.UpdateNameRequest) error
	UpdateEmail(ctx context.Context, req models.UpdateEmailRequest) error
	UpdatePassword(ctx context.Context, req models.UpdatePasswordRequest) error
}

type UserHandler struct {
	userService UserService
}

func NewUserHandler(userService UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

func (h *UserHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/all", h.GetAllUsers)
	r.Get("/profile", h.GetProfile)
	r.Patch("/profile/changename", h.UpdateName)
	r.Patch("/profile/changeemail", h.UpdateEmail)
	r.Patch("/profile/changepassword", h.UpdatePassword)

	return r
}

// GetAllUsers возвращает список всех пользователей
// @Summary Получить всех пользователей
// @Description Возвращает список всех зарегистрированных пользователей
// @Tags user
// @Produce json
// @Security CookieAuth
// @Param limit query int false "Лимит (по умолчанию нет лимита)"
// @Param offset query int false "Смещение (по умолчанию 0)"
// @Success 200 {array} models.User
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /user/all [get]
func (h *UserHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	limit := 0
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

	users, err := h.userService.GetAllUsers(r.Context(), models.GetAllUsersRequest{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{
			Error: "failed to get users list",
		})
		return
	}

	render.JSON(w, r, users)
}

// GetProfile Возвращает профиль текущего пользователя
// @Summary Получить профиль
// @Description Возвращает профиль текущего аутентифицированного пользователя
// @Tags user
// @Produce json
// @Security CookieAuth
// @Param userID query string false "ID пользователя"
// @Success 200 {object} models.User
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /user/profile [get]
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
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
	userProfile, err := h.userService.GetProfile(r.Context(), userID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{
			Error: "failed to get user profile",
		})
		return
	}

	render.JSON(w, r, userProfile)
}

// UpdateName Обновляет имя пользователя
// @Summary Изменить имя
// @Description Изменяет имя текущего аутентифицированного пользователя
// @Tags user
// @Accept x-www-form-urlencoded
// @Produce json
// @Security CookieAuth
// @Param userID query string false "ID пользователя"
// @Param name formData string true "User name"
// @Success 200 {object} models.User
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /user/profile/changename [patch]
func (h *UserHandler) UpdateName(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid form data"})
		return
	}
	var userID uuid.UUID
	var err error
	userIDStr := r.URL.Query().Get("userID")
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

	req := models.UpdateNameRequest{
		UserID:   userID,
		FullName: r.FormValue("name"),
	}

	err = h.userService.UpdateName(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{
			Error: "failed to update user profile",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"update name": "success"})
}

// UpdateEmail Обновляет Email пользователя
// @Summary Изменить Email
// @Description Изменяет Email текущего аутентифицированного пользователя
// @Tags user
// @Accept x-www-form-urlencoded
// @Produce json
// @Security CookieAuth
// @Param userID query string false "ID пользователя"
// @Param email formData string true "User email"
// @Success 200 {object} models.User
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /user/profile/changeemail [patch]
func (h *UserHandler) UpdateEmail(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid form data"})
		return
	}
	var userID uuid.UUID
	var err error
	userIDStr := r.URL.Query().Get("userID")
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

	req := models.UpdateEmailRequest{
		UserID: userID,
		Email:  r.FormValue("email"),
	}

	err = h.userService.UpdateEmail(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{
			Error: "failed to update user profile",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"update email": "success"})
}

// UpdatePassword Обновляет пароль пользователя
// @Summary Изменить пароль
// @Description Обновляет пароль текущего аутентифицированного пользователя
// @Tags user
// @Accept x-www-form-urlencoded
// @Produce json
// @Security CookieAuth
// @Param password formData string true "User password"
// @Success 200 {object} models.User
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /user/profile/changepassword [patch]
func (h *UserHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid form data"})
		return
	}

	req := models.UpdatePasswordRequest{
		Password: r.FormValue("password"),
	}

	err := h.userService.UpdatePassword(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{
			Error: "failed to update user profile",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"update password": "success"})
}
