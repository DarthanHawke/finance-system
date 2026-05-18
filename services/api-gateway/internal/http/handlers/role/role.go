package role

/*
import (
	"client-service/internal/models"
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type RoleService interface {
	CreateRelation(ctx context.Context, req models.CreateRelationRequest) error
	DeleteRelation(ctx context.Context, req models.DeleteRelationRequest) error
	AssignPermission(ctx context.Context, req models.AssignPermissionRequest) error
	RevokePermission(ctx context.Context, req models.RevokePermissionRequest) error
	CheckPermission(ctx context.Context, req models.CheckPermissionRequest) (bool, error)
	GetAllPermissions(ctx context.Context) ([]models.Permission, error)
	GetUserRelations(ctx context.Context, userID uuid.UUID) ([]models.Relation, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error)
	GetAllEntities(ctx context.Context) ([]models.Entity, error)
	GetUserRole(ctx context.Context) (string, error)
	GetPermissionsForRelationType(ctx context.Context, relationType string) ([]models.Permission, error)
}

type RoleHandler struct {
	roleService RoleService
}

func NewRoleHandler(roleService RoleService) *RoleHandler {
	return &RoleHandler{
		roleService: roleService,
	}
}

func (h *RoleHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/relations", h.CreateRelation)
	r.Delete("/relations", h.DeleteRelation)
	r.Post("/permissions/assign", h.AssignPermission)
	r.Delete("/permissions/revoke", h.RevokePermission)
	r.Post("/permissions/check", h.CheckPermission)
	r.Get("/permissions", h.GetAllPermissions)
	r.Get("/users/{userID}/relations", h.GetUserRelations)
	r.Get("/users/{userID}/permissions", h.GetUserPermissions)
	r.Get("/relations/{relationType}/permissions", h.GetPermissionsForRelationType)
	r.Get("/entities", h.GetAllEntities)
	r.Get("/role", h.GetUserRole)
	return r
}

// CreateRelation создает новую связь между сущностями
// @Summary Создать связь
// @Description Создает связь между source и target сущностями (Admin only)
// @Tags role
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param request body models.CreateRelationRequest true "Данные связи"
// @Success 201
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /role/relations [post]
func (h *RoleHandler) CreateRelation(w http.ResponseWriter, r *http.Request) {
	var req models.CreateRelationRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	if err := h.roleService.CreateRelation(r.Context(), req); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to create relation"})
		return
	}

	render.Status(r, http.StatusCreated)
}

// DeleteRelation удаляет связь между сущностями
// @Summary Удалить связь
// @Description Удаляет связь между source и target сущностями (Admin only)
// @Tags role
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param request body models.DeleteRelationRequest true "Данные связи"
// @Success 204
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /role/relations [delete]
func (h *RoleHandler) DeleteRelation(w http.ResponseWriter, r *http.Request) {
	var req models.DeleteRelationRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	if err := h.roleService.DeleteRelation(r.Context(), req); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to delete relation"})
		return
	}

	render.Status(r, http.StatusNoContent)
}

// AssignPermission назначает разрешение для типа связи
// @Summary Назначить разрешение
// @Description Назначает разрешение для указанного типа связи (Admin only)
// @Tags role
// @Accept json
// @Produce json
// @Param request body models.AssignPermissionRequest true "Данные разрешения"
// @Success 201
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /role/permissions/assign [post]
func (h *RoleHandler) AssignPermission(w http.ResponseWriter, r *http.Request) {
	var req models.AssignPermissionRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	if err := h.roleService.AssignPermission(r.Context(), req); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to assign permission"})
		return
	}

	render.Status(r, http.StatusCreated)
}

// RevokePermission отзывает разрешение для типа связи
// @Summary Отозвать разрешение
// @Description Отзывает разрешение для указанного типа связи (Admin only)
// @Tags role
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param request body models.RevokePermissionRequest true "Данные разрешения"
// @Success 204
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /role/permissions/revoke [delete]
func (h *RoleHandler) RevokePermission(w http.ResponseWriter, r *http.Request) {
	var req models.RevokePermissionRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	if err := h.roleService.RevokePermission(r.Context(), req); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to revoke permission"})
		return
	}

	render.Status(r, http.StatusNoContent)
}

// CheckPermission проверяет наличие разрешения у субъекта для объекта
// @Summary Проверить разрешение
// @Description Проверяет наличие указанного разрешения у субъекта для объекта (Admin only)
// @Tags role
// @Accept json
// @Produce json
// @Security CookieAuth
// @Param request body models.CheckPermissionRequest true "Данные проверки разрешения"
// @Success 200 {object} models.CheckPermissionResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /role/permissions/check [post]
func (h *RoleHandler) CheckPermission(w http.ResponseWriter, r *http.Request) {
	var req models.CheckPermissionRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	hasPermission, err := h.roleService.CheckPermission(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to check permission"})
		return
	}

	render.JSON(w, r, models.CheckPermissionResponse{HasPermission: hasPermission})
}

// GetAllPermissions возвращает все доступные разрешения
// @Summary Получить все разрешения
// @Description Возвращает список всех доступных разрешений (Admin only)
// @Tags role
// @Produce json
// @Security CookieAuth
// @Success 200 {array} models.Permission
// @Failure 500 {object} models.ErrorResponse
// @Router /role/permissions [get]
func (h *RoleHandler) GetAllPermissions(w http.ResponseWriter, r *http.Request) {
	permissions, err := h.roleService.GetAllPermissions(r.Context())
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to get permissions"})
		return
	}

	render.JSON(w, r, permissions)
}

// GetUserRelations возвращает все связи пользователя
// @Summary Получить связи пользователя
// @Description Возвращает список всех связей указанного пользователя (Admin only)
// @Tags role
// @Produce json
// @Security CookieAuth
// @Param userID path string true "ID пользователя"
// @Success 200 {array} models.Relation
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /role/users/{userID}/relations [get]
func (h *RoleHandler) GetUserRelations(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "invalid user ID format"})
		return
	}

	relations, err := h.roleService.GetUserRelations(r.Context(), userID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to get user relations"})
		return
	}

	render.JSON(w, r, relations)
}

// GetUserPermissions возвращает все разрешения пользователя
// @Summary Получить разрешения пользователя
// @Description Возвращает список всех разрешений указанного пользователя (Admin only)
// @Tags role
// @Produce json
// @Security CookieAuth
// @Param userID path string true "ID пользователя"
// @Success 200 {array} models.Permission
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /role/users/{userID}/permissions [get]
func (h *RoleHandler) GetUserPermissions(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "invalid user ID format"})
		return
	}

	permissions, err := h.roleService.GetUserPermissions(r.Context(), userID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to get user permissions"})
		return
	}

	render.JSON(w, r, permissions)
}

// GetPermissionsForRelationType возвращает разрешения для типа связи
// @Summary Получить разрешения для типа связи
// @Description Возвращает список разрешений для указанного типа связи (Admin only)
// @Tags role
// @Produce json
// @Security CookieAuth
// @Param relationType path string true "Тип связи"
// @Success 200 {array} models.Permission
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /role/relations/{relationType}/permissions [get]
func (h *RoleHandler) GetPermissionsForRelationType(w http.ResponseWriter, r *http.Request) {
	relationType := chi.URLParam(r, "relationType")
	if relationType == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, models.ErrorResponse{Error: "relation type is required"})
		return
	}

	permissions, err := h.roleService.GetPermissionsForRelationType(r.Context(), relationType)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to get permissions for relation type"})
		return
	}

	render.JSON(w, r, permissions)
}

// GetAllEntities возвращает все сущности
// @Summary Получить все сущности
// @Description Возвращает список всех сущностей (Admin only)
// @Tags role
// @Produce json
// @Security CookieAuth
// @Success 200 {array} models.Entity
// @Failure 500 {object} models.ErrorResponse
// @Router /role/entities [get]
func (h *RoleHandler) GetAllEntities(w http.ResponseWriter, r *http.Request) {
	entities, err := h.roleService.GetAllEntities(r.Context())
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to get entities"})
		return
	}

	render.JSON(w, r, entities)
}

// GetUserRole возвращает роль текущего пользователя
// @Summary Получить роль пользователя
// @Description Возвращает роль текущего аутентифицированного пользователя
// @Tags role
// @Produce json
// @Security CookieAuth
// @Success 200 {object} models.UserRoleResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /role/role [get]
func (h *RoleHandler) GetUserRole(w http.ResponseWriter, r *http.Request) {
	role, err := h.roleService.GetUserRole(r.Context())
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, models.ErrorResponse{Error: "failed to get user role"})
		return
	}

	render.JSON(w, r, models.UserRoleResponse{Role: role})
}
*/
