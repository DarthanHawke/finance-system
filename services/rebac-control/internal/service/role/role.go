package role

import (
	billingerr "billing-service/internal/lib/errors"

	"billing-service/internal/models"
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	admin   = "admin"
	support = "support"
	client  = "client"
	unknown = "unknow"
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
		logger:     logger.With(zap.String("component", "billing_service")),
	}
}

type RoleManage interface {
	GetEntityID(ctx context.Context, entityType string) (uuid.UUID, error)
	GetEntityRelations(ctx context.Context, entityID uuid.UUID) ([]models.Relation, error)
	CreateRelation(ctx context.Context, sourceID, targetID uuid.UUID, relationType string) error
	DeleteRelation(ctx context.Context, sourceID, targetID uuid.UUID, relationType string) error
	AssignPermission(ctx context.Context, permissionID uuid.UUID, relationType string) error
	RevokePermission(ctx context.Context, permissionID uuid.UUID, relationType string) error
	CheckPermission(ctx context.Context, subjectID, objectID uuid.UUID, permissionName string) (bool, error)
	GetAllEntities(ctx context.Context) ([]models.Entity, error)
	GetAllPermissions(ctx context.Context) ([]models.Permission, error)
	GetUserRelations(ctx context.Context, userID uuid.UUID) ([]models.Relation, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error)
	GetPermissionsForRelationType(ctx context.Context, relationType string) ([]models.Permission, error)
}

func (s *RoleService) CreateRelation(
	ctx context.Context,
	sourceID, targetID uuid.UUID,
	relationType string,
) error {
	const op = "service.role.CreateRelation"

	logger := s.logger.With(zap.String("op", op))

	logger.Info("Creating relation")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	relationEntytyId, err := s.roleManage.GetEntityID(ctx, models.RelationEntyty)
	if err != nil {
		return fmt.Errorf("failed to get entity in ReBAC: %v", err)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, relationEntytyId, models.RelationManage)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.RelationManage),
		)
		return fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.RelationManage),
		)
		return fmt.Errorf("permission denied %v: %w", op, err)
	}

	err = s.roleManage.CreateRelation(ctx, sourceID, targetID, relationType)
	if err != nil {
		logger.Error("failed to create relation",
			zap.String("sourceID", sourceID.String()),
			zap.String("targetID", targetID.String()),
			zap.String("relationType", relationType),
			zap.Error(err),
		)
		return fmt.Errorf("%s: %w", op, billingerr.ErrCreateRelation)
	}

	return nil
}

func (s *RoleService) DeleteRelation(
	ctx context.Context,
	sourceID, targetID uuid.UUID,
	relationType string,
) error {
	const op = "service.role.DeleteRelation"

	logger := s.logger.With(zap.String("op", op))

	logger.Info("Deleting relation")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	relationEntytyId, err := s.roleManage.GetEntityID(ctx, models.RelationEntyty)
	if err != nil {
		return fmt.Errorf("failed to get entity in ReBAC: %v", err)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, relationEntytyId, models.RelationManage)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.RelationManage),
		)
		return fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.RelationManage),
		)
		return fmt.Errorf("permission denied %v: %w", op, err)
	}

	err = s.roleManage.DeleteRelation(ctx, sourceID, targetID, relationType)
	if err != nil {
		logger.Error("failed to delete relation",
			zap.String("sourceID", sourceID.String()),
			zap.String("targetID", targetID.String()),
			zap.String("relationType", relationType),
			zap.Error(err),
		)
		return fmt.Errorf("%s: %w", op, billingerr.ErrDeleteRelation)
	}

	return nil
}

func (s *RoleService) AssignPermission(
	ctx context.Context,
	permissionID uuid.UUID,
	relationType string,
) error {
	const op = "service.role.AssignPermission"

	logger := s.logger.With(zap.String("op", op))

	logger.Info("Assigning permission")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	permissionEntityId, err := s.roleManage.GetEntityID(ctx, models.PermissionEntity)
	if err != nil {
		return fmt.Errorf("failed to get entity in ReBAC: %v", err)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, permissionEntityId, models.PermissionManage)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PermissionManage),
		)
		return fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PermissionManage),
		)
		return fmt.Errorf("permission denied %v: %w", op, err)
	}

	err = s.roleManage.AssignPermission(ctx, permissionID, relationType)
	if err != nil {
		logger.Error("failed to assign permission",
			zap.String("permissionID", permissionID.String()),
			zap.String("relationType", relationType),
			zap.Error(err),
		)
		return fmt.Errorf("%s: %w", op, billingerr.ErrAssignPermission)
	}

	return nil
}

func (s *RoleService) RevokePermission(
	ctx context.Context,
	permissionID uuid.UUID,
	relationType string,
) error {
	const op = "service.role.AssignPermission"

	logger := s.logger.With(zap.String("op", op))

	logger.Info("Assigning permission")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	permissionEntityId, err := s.roleManage.GetEntityID(ctx, models.PermissionEntity)
	if err != nil {
		return fmt.Errorf("failed to get entity in ReBAC: %v", err)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, permissionEntityId, models.PermissionManage)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PermissionManage),
		)
		return fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PermissionManage),
		)
		return fmt.Errorf("permission denied %v: %w", op, err)
	}

	err = s.roleManage.RevokePermission(ctx, permissionID, relationType)
	if err != nil {
		logger.Error("failed to revoke permission",
			zap.String("permissionID", permissionID.String()),
			zap.String("relationType", relationType),
			zap.Error(err),
		)
		return fmt.Errorf("%s: %w", op, billingerr.ErrAssignPermission)
	}

	return nil
}

func (s *RoleService) CheckPermission(
	ctx context.Context,
	subjectID, objectID uuid.UUID,
	permissionName string,
) (bool, error) {
	const op = "service.role.CheckPermission"

	logger := s.logger.With(zap.String("op", op))

	logger.Info("Checking permission")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return false, fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	permissionEntityId, err := s.roleManage.GetEntityID(ctx, models.PermissionEntity)
	if err != nil {
		return false, fmt.Errorf("failed to get entity in ReBAC: %v", err)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, permissionEntityId, models.PermissionRead)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PermissionRead),
		)
		return false, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PermissionRead),
		)
		return false, fmt.Errorf("permission denied %v: %w", op, err)
	}

	hasPermission, err := s.roleManage.CheckPermission(ctx, subjectID, objectID, permissionName)
	if err != nil {
		logger.Error("failed to check permission",
			zap.String("subjectID", subjectID.String()),
			zap.String("objectID", objectID.String()),
			zap.String("permission name", permissionName),
			zap.Error(err),
		)
		return false, fmt.Errorf("%s: %w", op, billingerr.ErrCheckPermission)
	}

	return hasPermission, nil
}

func (s *RoleService) GetAllEntities(
	ctx context.Context,
) ([]models.Entity, error) {
	const op = "service.role.GetAllEntities"

	logger := s.logger.With(zap.String("op", op))

	logger.Info("Getting all entities")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	relationEntytyId, err := s.roleManage.GetEntityID(ctx, models.RelationEntyty)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity in ReBAC: %v", err)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, relationEntytyId, models.RelationRead)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.RelationRead),
		)
		return nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.RelationRead),
		)
		return nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	entities, err := s.roleManage.GetAllEntities(ctx)
	if err != nil {
		logger.Error("failed to get all entities",
			zap.Error(err),
		)
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetEntities)
	}

	logger.Debug("Successfully retrieved entities",
		zap.Int("count", len(entities)),
	)
	return entities, nil
}

func (s *RoleService) GetAllPermissions(
	ctx context.Context,
) ([]models.Permission, error) {
	const op = "service.role.GetAllPermissions"

	logger := s.logger.With(zap.String("op", op))

	logger.Info("Getting all permissions")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	permissionEntityId, err := s.roleManage.GetEntityID(ctx, models.PermissionEntity)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity in ReBAC: %v", err)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, permissionEntityId, models.PermissionRead)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PermissionRead),
		)
		return nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PermissionRead),
		)
		return nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	permissions, err := s.roleManage.GetAllPermissions(ctx)
	if err != nil {
		logger.Error("failed to get all permissions",
			zap.Error(err),
		)
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetPermissions)
	}

	return permissions, nil
}

func (s *RoleService) GetUserRelations(
	ctx context.Context,
	targetUserID uuid.UUID,
) ([]models.Relation, error) {
	const op = "service.role.GetUserRelations"

	logger := s.logger.With(zap.String("op", op))

	logger.Info("Getting user relations")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	relationEntytyId, err := s.roleManage.GetEntityID(ctx, models.RelationEntyty)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity in ReBAC: %v", err)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, relationEntytyId, models.RelationRead)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.RelationRead),
		)
		return nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.RelationRead),
		)
		return nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	relations, err := s.roleManage.GetUserRelations(ctx, targetUserID)
	if err != nil {
		logger.Error("failed to get user relations",
			zap.String("userID", targetUserID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetRelations)
	}

	return relations, nil
}

func (s *RoleService) GetUserPermissions(
	ctx context.Context,
	targetUserID uuid.UUID,
) ([]models.Permission, error) {
	const op = "service.role.GetUserPermissions"

	logger := s.logger.With(zap.String("op", op))

	logger.Info("Getting user permissions")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	permissionEntityId, err := s.roleManage.GetEntityID(ctx, models.PermissionEntity)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity in ReBAC: %v", err)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, permissionEntityId, models.PermissionRead)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PermissionRead),
		)
		return nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PermissionRead),
		)
		return nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	permissions, err := s.roleManage.GetUserPermissions(ctx, targetUserID)
	if err != nil {
		logger.Error("failed to get user permissions",
			zap.String("userID", targetUserID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetPermissions)
	}

	return permissions, nil
}

func (s *RoleService) GetPermissionsForRelationType(
	ctx context.Context,
	relationType string,
) ([]models.Permission, error) {
	const op = "service.role.GetPermissionsForRelationType"

	logger := s.logger.With(zap.String("op", op))

	logger.Info("Getting permissions for relation type")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	permissionEntityId, err := s.roleManage.GetEntityID(ctx, models.PermissionEntity)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity in ReBAC: %v", err)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, permissionEntityId, models.PermissionRead)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PermissionRead),
		)
		return nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.PermissionRead),
		)
		return nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	permissions, err := s.roleManage.GetPermissionsForRelationType(ctx, relationType)
	if err != nil {
		logger.Error("failed to get permissions for relation type",
			zap.String("relationType", relationType),
			zap.Error(err),
		)
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetPermissions)
	}

	return permissions, nil
}

func (s *RoleService) GetUserRole(ctx context.Context) (string, error) {
	const op = "service.role.GetUserRole"

	logger := s.logger.With(zap.String("op", op))

	logger.Info("Getting user role")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return unknown, fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}
	financeSystemEntityId, err := s.roleManage.GetEntityID(ctx, models.FinanceSystemEntity)
	if err != nil {
		return unknown, fmt.Errorf("failed to get entity in ReBAC: %v", err)
	}

	relations, err := s.roleManage.GetEntityRelations(ctx, financeSystemEntityId)
	if err != nil {
		logger.Error("failed to get entity relations",
			zap.String("entityID", financeSystemEntityId.String()),
			zap.Error(err),
		)
		return unknown, err
	}

	for _, rel := range relations {
		if rel.TargetID == userID || rel.SourceID == userID {
			switch rel.RelationType {
			case models.AdminRelationType:
				return admin, nil
			case models.SupportRelationType:
				return support, nil
			case models.ClientRelationType:
				return client, nil
			default:
				return unknown, nil
			}
		}
	}

	return unknown, err
}
