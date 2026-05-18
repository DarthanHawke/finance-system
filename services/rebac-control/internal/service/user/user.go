package user

/* legacy code
import (
	billingerr "billing-service/internal/lib/errors"
	"billing-service/internal/models"
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type UserService struct {
	userManage UserManage
	roleManage RoleManage
	logger     *zap.Logger
}

func NewUserService(
	userManage UserManage,
	roleManage RoleManage,
	logger *zap.Logger,
) *UserService {
	return &UserService{
		userManage: userManage,
		roleManage: roleManage,
		logger:     logger.With(zap.String("component", "billing_service")),
	}
}

type UserManage interface {
	GetAllUsers(ctx context.Context, limit, offset int) ([]models.User, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*models.User, error)
	UpdateName(ctx context.Context, userID uuid.UUID, fulName string) error
	UpdateEmail(ctx context.Context, userID uuid.UUID, email string) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, password string) error
}

type RoleManage interface {
	GetEntityID(ctx context.Context, entityType string) (uuid.UUID, error)
	CreateRelation(ctx context.Context, sourceID, targetID uuid.UUID, relationType string) error
	AssignPermission(ctx context.Context, permissionID uuid.UUID, relationType string) error
	CheckPermission(ctx context.Context, subjectID, objectID uuid.UUID, permissionName string) (bool, error)
}

// GetAllUsers возвращает список всех пользователей с пагинацией
func (s *UserService) GetAllUsers(
	ctx context.Context,
	limit, offset int,
) ([]models.User, error) {
	const op = "service.user.GetAllUsers"

	logger := s.logger.With(
		zap.String("op", op),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	logger.Info("Getting all users")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrInavlidToken)
	}

	userEntityId, err := s.roleManage.GetEntityID(ctx, models.UserEntity)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get entity in ReBAC: %w", op, err)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, userEntityId, models.UserReadAll)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.UserReadAll),
		)
		return nil, fmt.Errorf("%s: failed to check permission: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.UserReadAll),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	users, err := s.userManage.GetAllUsers(ctx, limit, offset)
	if err != nil {
		logger.Error("failed to get users",
			zap.Error(err),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	logger.Debug("Successfully got users list",
		zap.Int("count", len(users)),
	)

	return users, nil
}

// GetProfile возвращает профиль пользователя *models.User
func (s *UserService) GetProfile(ctx context.Context, targetID uuid.UUID) (*models.User, error) {
	const op = "service.user.GetProfile"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Getting user profile")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	if targetID == uuid.Nil {
		targetID = userID
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, targetID, models.UserRead)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.UserRead),
		)
		return nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.UserRead),
		)
		return nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	user, err := s.userManage.GetProfile(ctx, targetID)
	if err != nil {
		logger.Error("cant get profile", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	return user, nil
}

// UpdateName - обновляет профиль пользователя(models.User)
func (s *UserService) UpdateName(ctx context.Context, targetID uuid.UUID, fulName string) error {
	const op = "service.user.UpdateName"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Updating user profile")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	if targetID == uuid.Nil {
		targetID = userID
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, targetID, models.UserUpdate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.UserUpdate),
		)
		return fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.UserUpdate),
		)
		return fmt.Errorf("permission denied %v: %w", op, err)
	}

	err = s.userManage.UpdateName(ctx, targetID, fulName)
	if err != nil {
		logger.Error("cant update profile", zap.Error(err))

		return fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	return nil
}

// UpdateEmail - обновляет профиль пользователя(models.User)
func (s *UserService) UpdateEmail(ctx context.Context, targetID uuid.UUID, email string) error {
	const op = "service.user.UpdateEmail"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Updating user profile")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	if targetID == uuid.Nil {
		targetID = userID
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, targetID, models.UserUpdate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.UserUpdate),
		)
		return fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.UserUpdate),
		)
		return fmt.Errorf("permission denied %v: %w", op, err)
	}

	err = s.userManage.UpdateEmail(ctx, targetID, email)
	if err != nil {
		logger.Error("cant update profile", zap.Error(err))

		return fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	return nil
}

// UpdatePassword - обновляет профиль пользователя(models.User)
func (s *UserService) UpdatePassword(ctx context.Context, password string) error {
	const op = "service.user.UpdatePassword"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Updating user profile")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, userID, models.UserUpdate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.UserUpdate),
		)
		return fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.UserUpdate),
		)
		return fmt.Errorf("permission denied %v: %w", op, err)
	}
	err = s.userManage.UpdatePassword(ctx, userID, password)
	if err != nil {
		logger.Error("cant update profile", zap.Error(err))

		return fmt.Errorf("%s: %w", op, billingerr.ErrGetProfile)
	}

	return nil
}
*/
