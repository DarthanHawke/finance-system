package user

import (
	"client-service/internal/models"
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type UserService struct {
	userManage UserManage
	logger     *zap.Logger
}

func NewUserService(
	userManage UserManage,
	logger *zap.Logger,
) *UserService {
	return &UserService{
		userManage: userManage,
		logger:     logger.With(zap.String("component", "client_service")),
	}
}

type UserManage interface {
	GetAllUsers(ctx context.Context, limit, offset int) ([]models.User, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*models.User, error)
	UpdateName(ctx context.Context, userID uuid.UUID, fulName string) error
	UpdateEmail(ctx context.Context, userID uuid.UUID, email string) error
	UpdatePassword(ctx context.Context, password string) error
}

// GetAllUsers возвращает список всех пользователей []models.User
func (s *UserService) GetAllUsers(ctx context.Context, req models.GetAllUsersRequest) ([]models.User, error) {
	const op = "service.user.GetAllUsers"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Getting all users")

	users, err := s.userManage.GetAllUsers(ctx, req.Limit, req.Offset)
	if err != nil {
		logger.Error("cant get all users", zap.Error(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	logger.Info("Successfully retrieved users", zap.Int("count", len(users)))
	return users, nil
}

// GetProfile возвращает профиль пользователя *models.User
func (s *UserService) GetProfile(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	const op = "service.user.GetProfile"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Getting user profile")

	user, err := s.userManage.GetProfile(ctx, userID)
	if err != nil {
		logger.Error("cant get profile", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

// UpdateName - обновляет профиль пользователя(models.User)
func (s *UserService) UpdateName(ctx context.Context, req models.UpdateNameRequest) error {
	const op = "service.user.UpdateName"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Updating user profile")

	err := s.userManage.UpdateName(ctx, req.UserID, req.FullName)
	if err != nil {
		logger.Error("cant update profile", zap.Error(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// UpdateEmail - обновляет профиль пользователя(models.User)
func (s *UserService) UpdateEmail(ctx context.Context, req models.UpdateEmailRequest) error {
	const op = "service.user.UpdateEmail"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Updating user profile")

	err := s.userManage.UpdateEmail(ctx, req.UserID, req.Email)
	if err != nil {
		logger.Error("cant update profile", zap.Error(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// UpdatePassword - обновляет профиль пользователя(models.User)
func (s *UserService) UpdatePassword(ctx context.Context, req models.UpdatePasswordRequest) error {
	const op = "service.user.UpdatePassword"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Updating user profile")

	err := s.userManage.UpdatePassword(ctx, req.Password)
	if err != nil {
		logger.Error("cant update profile", zap.Error(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
