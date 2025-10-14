package role

import (
	"client-service/internal/models"
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type RoleService struct {
	roleManage RoleManage
	logger     *zap.Logger
}

func NewRoleService(
	roleManage RoleManage,
	logger *zap.Logger,
) *RoleService {
	return &RoleService{
		roleManage: roleManage,
		logger:     logger.With(zap.String("component", "client_service")),
	}
}

type RoleManage interface {
	CreateRelation(ctx context.Context, sourceID, targetID uuid.UUID, relationType string) error
	DeleteRelation(ctx context.Context, sourceID, targetID uuid.UUID, relationType string) error
	AssignPermission(ctx context.Context, permissionID uuid.UUID, relationType string) error
	RevokePermission(ctx context.Context, permissionID uuid.UUID, relationType string) error
	CheckPermission(ctx context.Context, subjectID, objectID uuid.UUID, permissionName string) (bool, error)
	GetAllPermissions(ctx context.Context) ([]models.Permission, error)
	GetUserRelations(ctx context.Context, userID uuid.UUID) ([]models.Relation, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error)
	GetAllEntities(ctx context.Context) ([]models.Entity, error)
	GetUserRole(ctx context.Context) (string, error)
	GetPermissionsForRelationType(ctx context.Context, relationType string) ([]models.Permission, error)
}

// CreateRelation создает отношение между сущностями
func (s *RoleService) CreateRelation(ctx context.Context, req models.CreateRelationRequest) error {
	const op = "service.role.CreateRelation"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("source_id", req.SourceID.String()),
		zap.String("target_id", req.TargetID.String()),
		zap.String("relation_type", req.RelationType),
	)

	logger.Info("Creating relation")

	err := s.roleManage.CreateRelation(ctx, req.SourceID, req.TargetID, req.RelationType)
	if err != nil {
		logger.Error("failed to create relation", zap.Error(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// DeleteRelation удаляет отношение между сущностями
func (s *RoleService) DeleteRelation(ctx context.Context, req models.DeleteRelationRequest) error {
	const op = "service.role.DeleteRelation"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("source_id", req.SourceID.String()),
		zap.String("target_id", req.TargetID.String()),
		zap.String("relation_type", req.RelationType),
	)

	logger.Info("Deleting relation")

	err := s.roleManage.DeleteRelation(ctx, req.SourceID, req.TargetID, req.RelationType)
	if err != nil {
		logger.Error("failed to delete relation", zap.Error(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// AssignPermission назначает разрешение для типа отношения
func (s *RoleService) AssignPermission(ctx context.Context, req models.AssignPermissionRequest) error {
	const op = "service.role.AssignPermission"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("permission_id", req.PermissionID.String()),
		zap.String("relation_type", req.RelationType),
	)

	logger.Info("Assigning permission")

	err := s.roleManage.AssignPermission(ctx, req.PermissionID, req.RelationType)
	if err != nil {
		logger.Error("failed to assign permission", zap.Error(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// RevokePermission отзывает разрешение у типа отношения
func (s *RoleService) RevokePermission(ctx context.Context, req models.RevokePermissionRequest) error {
	const op = "service.role.RevokePermission"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("permission_id", req.PermissionID.String()),
		zap.String("relation_type", req.RelationType),
	)

	logger.Info("Revoking permission")

	err := s.roleManage.RevokePermission(ctx, req.PermissionID, req.RelationType)
	if err != nil {
		logger.Error("failed to revoke permission", zap.Error(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// CheckPermission проверяет наличие разрешения у субъекта для объекта
func (s *RoleService) CheckPermission(ctx context.Context, req models.CheckPermissionRequest) (bool, error) {
	const op = "service.role.CheckPermission"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("subject_id", req.SubjectID.String()),
		zap.String("object_id", req.ObjectID.String()),
		zap.String("permission_name", req.PermissionName),
	)

	logger.Info("Checking permission")

	hasPermission, err := s.roleManage.CheckPermission(ctx, req.SubjectID, req.ObjectID, req.PermissionName)
	if err != nil {
		logger.Error("failed to check permission", zap.Error(err))
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return hasPermission, nil
}

// GetAllPermissions возвращает все доступные разрешения
func (s *RoleService) GetAllPermissions(ctx context.Context) ([]models.Permission, error) {
	const op = "service.role.GetAllPermissions"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Getting all permissions")

	permissions, err := s.roleManage.GetAllPermissions(ctx)
	if err != nil {
		logger.Error("failed to get all permissions", zap.Error(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return permissions, nil
}

// GetUserRelations возвращает все отношения пользователя
func (s *RoleService) GetUserRelations(ctx context.Context, userID uuid.UUID) ([]models.Relation, error) {
	const op = "service.role.GetUserRelations"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("user_id", userID.String()),
	)

	logger.Info("Getting user relations")

	relations, err := s.roleManage.GetUserRelations(ctx, userID)
	if err != nil {
		logger.Error("failed to get user relations", zap.Error(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return relations, nil
}

// GetUserPermissions возвращает все разрешения пользователя
func (s *RoleService) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error) {
	const op = "service.role.GetUserPermissions"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("user_id", userID.String()),
	)

	logger.Info("Getting user permissions")

	permissions, err := s.roleManage.GetUserPermissions(ctx, userID)
	if err != nil {
		logger.Error("failed to get user permissions", zap.Error(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return permissions, nil
}

// GetAllEntities возвращает все сущности
func (s *RoleService) GetAllEntities(ctx context.Context) ([]models.Entity, error) {
	const op = "service.role.GetAllEntities"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Getting all entities")

	entities, err := s.roleManage.GetAllEntities(ctx)
	if err != nil {
		logger.Error("failed to get all entities", zap.Error(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return entities, nil
}

// GetUserRole возвращает роль текущего пользователя
func (s *RoleService) GetUserRole(ctx context.Context) (string, error) {
	const op = "service.role.GetUserRole"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Getting current user role")

	role, err := s.roleManage.GetUserRole(ctx)
	if err != nil {
		logger.Error("failed to get user role", zap.Error(err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return role, nil
}

// GetPermissionsForRelationType возвращает разрешения для типа отношения
func (s *RoleService) GetPermissionsForRelationType(ctx context.Context, relationType string) ([]models.Permission, error) {
	const op = "service.role.GetPermissionsForRelationType"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("relation_type", relationType),
	)

	logger.Info("Getting permissions for relation type")

	permissions, err := s.roleManage.GetPermissionsForRelationType(ctx, relationType)
	if err != nil {
		logger.Error("failed to get permissions for relation type", zap.Error(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return permissions, nil
}
