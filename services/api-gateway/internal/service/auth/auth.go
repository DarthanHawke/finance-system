package auth

import (
	"client-service/internal/models"
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AuthService struct {
	authManage AuthManage
	logger     *zap.Logger
}

func NewAuthService(
	authManage AuthManage,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		authManage: authManage,
		logger:     logger.With(zap.String("component", "client_service")),
	}
}

type AuthManage interface {
	RefreshSession(ctx context.Context) (*models.UserSession, error)
	Register(ctx context.Context, fullName, email, password string) (*models.UserSession, error)
	Login(ctx context.Context, email string, password string) (*models.UserSession, error)
	Logout(ctx context.Context, userID, sessionID uuid.UUID) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
	GetAllSessions(ctx context.Context, userID uuid.UUID) ([]models.Session, error)
}

// RefreshSession обновляет сессию
func (s *AuthService) RefreshSession(ctx context.Context) (*models.UserSession, error) {
	const op = "service.auth.RefreshSession"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Refreshing tokens")

	return s.authManage.RefreshSession(ctx)
}

// Register регестрирует пользователя, а затем создаёт сессию, возращает ID пользователя и токены
func (s *AuthService) Register(ctx context.Context, req models.RegisterRequest) (*models.UserSession, error) {
	const op = "service.auth.Register"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Registering new user")

	userSession, err := s.authManage.Register(ctx, req.FullName, req.Email, req.Password)
	if err != nil {
		logger.Error("cant register", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return userSession, nil
}

// Login - верифицирует пользователя, а затем создаёт сессию, возращает ID пользователя и токены
func (s *AuthService) Login(ctx context.Context, req models.LoginRequest) (*models.UserSession, error) {
	const op = "service.auth.Login"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Starting login user")

	userSession, err := s.authManage.Login(ctx, req.Email, req.Password)
	if err != nil {
		logger.Error("cant login", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return userSession, nil
}

// Logout - удаляет сессию(токены)
func (s *AuthService) Logout(ctx context.Context, userID, sessionID uuid.UUID) error {
	const op = "service.auth.Logout"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Starting logout user")

	err := s.authManage.Logout(ctx, userID, sessionID)
	if err != nil {
		s.logger.Error("cant logout",
			zap.Error(err),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// LogoutAll - удаляет все сессии пользователя
func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	const op = "service.auth.LogoutAll"

	logger := s.logger.With(
		zap.String("op", op),
	)

	err := s.authManage.LogoutAll(ctx, userID)
	if err != nil {
		logger.Error("cant logout", zap.Error(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetAllSessions - возвращвет все сессии пользователя
func (s *AuthService) GetAllSessions(ctx context.Context, userID uuid.UUID) ([]models.Session, error) {
	const op = "service.auth.GetAllSessions"

	logger := s.logger.With(
		zap.String("op", op),
	)

	sessions, err := s.authManage.GetAllSessions(ctx, userID)
	if err != nil {
		logger.Error("cant get sessions", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return sessions, nil
}
