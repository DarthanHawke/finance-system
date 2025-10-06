package auth

import (
	billingerr "billing-service/internal/lib/errors"
	"billing-service/internal/models"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AuthService struct {
	userManage    AuthUserManage
	sessionManage AuthSessionManage
	roleManage    RoleManage
	cache         RedisCacheManage
	logger        *zap.Logger
}

func NewAuthService(
	userManage AuthUserManage,
	sessionManage AuthSessionManage,
	roleManage RoleManage,

	cache RedisCacheManage,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		userManage:    userManage,
		sessionManage: sessionManage,
		roleManage:    roleManage,
		cache:         cache,
		logger:        logger.With(zap.String("component", "billing_service")),
	}
}

type AuthUserManage interface {
	Register(ctx context.Context, fullName string, email string, password string) (uuid.UUID, error)
	Login(ctx context.Context, email string, password string) (uuid.UUID, error)
}

type AuthSessionManage interface {
	CreateSession(ctx context.Context, userID uuid.UUID) (*models.UserSession, error)
	RefreshSession(ctx context.Context, userID uuid.UUID, refreshToken string) (*models.UserSession, error)
	Logout(ctx context.Context, userID, sessionID uuid.UUID) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
	GetAll(ctx context.Context, userID uuid.UUID) ([]models.Session, error)
}

type RoleManage interface {
	CreateEntityWithID(ctx context.Context, id uuid.UUID, entityType string) error
	CreateRelation(ctx context.Context, sourceID, targetID uuid.UUID, relationType string) error
	CheckPermission(ctx context.Context, subjectID, objectID uuid.UUID, permissionName string) (bool, error)
	GetEntityID(ctx context.Context, entityType string) (uuid.UUID, error)
}

type RedisCacheManage interface {
	SetWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error
}

// RefreshSession обновляет сессию с помощью refreshToken(вызывать из клиета по истичению срока жизни Access токена)
func (s *AuthService) RefreshSession(
	ctx context.Context,
	refreshToken string,
) (*models.UserSession, error) {
	const op = "service.auth.RefreshSession"

	s.logger.With(
		zap.String("op", op),
	)

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrRefreshSession)
	}

	s.logger.Info("Refreshing tokens")

	return s.sessionManage.RefreshSession(ctx, userID, refreshToken)
}

// Register регестрирует пользователя, а затем создаёт сессию, возращает ID пользователя и токены
func (s *AuthService) Register(ctx context.Context, fullName, email, password string) (*models.UserSession, error) {
	const op = "service.auth.Register"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Registering new user")

	userId, err := s.userManage.Register(ctx, fullName, email, password)
	if err != nil {
		logger.Error("cant register", zap.Error(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	logger.Info("Creating session for new user")

	userSession, err := s.sessionManage.CreateSession(ctx, userId)
	if err != nil {
		logger.Error("cant create session after register", zap.Error(err))
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrLogin)
	}

	userEntityId, err := s.roleManage.GetEntityID(ctx, models.UserEntity)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity in ReBAC: %v", err)
	}
	err = s.roleManage.CreateEntityWithID(ctx, userId, models.UserEntity)
	if err != nil {
		return nil, fmt.Errorf("failed to create user entity in ReBAC: %v", err)
	}
	err = s.roleManage.CreateRelation(ctx, userEntityId, userId, models.ClientRelationType)
	if err != nil {
		logger.Error("failed to create system-ownership relation in ReBAC", zap.Error(err))
		return nil, fmt.Errorf("failed to create system-ownership relation in ReBAC: %v", err)
	}
	err = s.roleManage.CreateRelation(ctx, userId, userId, models.ClientRelationType)
	if err != nil {
		logger.Error("failed to create self-ownership relation in ReBAC", zap.Error(err))
		return nil, fmt.Errorf("failed to create self-ownership relation in ReBAC: %v", err)
	}
	return userSession, nil
}

// Login - верифицирует пользователя, а затем создаёт сессию, возращает ID пользователя и токены
func (s *AuthService) Login(ctx context.Context, email, password string) (*models.UserSession, error) {
	const op = "service.auth.Login"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Starting login user")

	userId, err := s.userManage.Login(ctx, email, password)
	if err != nil {
		logger.Error("cant login", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrLogin)
	}

	userSession, err := s.sessionManage.CreateSession(ctx, userId)
	if err != nil {
		logger.Error("cant create session after login", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrLogin)
	}

	return userSession, nil
}

// Logout - удаляет сессию(токены)
func (s *AuthService) Logout(ctx context.Context, targetUserID, targetSessionID uuid.UUID) error {
	const op = "service.auth.Logout"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Starting logout user")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		logger.Error("cant parse userID")
		return fmt.Errorf("%s: %w", op, billingerr.ErrLogout)
	}

	if targetUserID == uuid.Nil {
		targetUserID = userID
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, targetUserID, models.SessionTerminate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.SessionTerminate),
		)
		return fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.SessionTerminate),
		)
		return fmt.Errorf("permission denied %v: %w", op, err)
	}

	sessionID, ok := ctx.Value(models.SessionIDKey).(uuid.UUID)
	if !ok {
		logger.Error("cant parse sessionid",
			zap.String("userID", userID.String()),
		)
		return fmt.Errorf("%s: %w", op, billingerr.ErrLogout)
	}

	if targetSessionID == uuid.Nil {
		targetSessionID = sessionID
	}

	err = s.sessionManage.Logout(ctx, targetUserID, targetSessionID)
	if err != nil {
		logger.Error("cant logout",
			zap.String("userID", targetUserID.String()),
			zap.String("sessionID", targetSessionID.String()),
			zap.Error(err),
		)

		return fmt.Errorf("%s: %w", op, billingerr.ErrLogout)
	}

	// добавляем id удалённой сессии в Blacklist
	sessionTTL, ok := ctx.Value(models.SessionTTL).(time.Time)
	if !ok {
		logger.Warn("cant get session ttl",
			zap.String("userID", targetUserID.String()),
			zap.String("sessionID", targetSessionID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("%s: %w", op, billingerr.ErrLogout)
	}
	cacheKey := fmt.Sprintf("invalid_session:%s", targetSessionID.String())
	ttl := time.Until(sessionTTL) + 5*time.Minute
	if err := s.cache.SetWithTTL(ctx, cacheKey, []byte("1"), ttl); err != nil {
		logger.Warn("cant add session to blacklist",
			zap.String("userID", targetUserID.String()),
			zap.String("sessionID", targetSessionID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("%s: %w", op, billingerr.ErrLogout)
	}
	return nil
}

// LogoutAll - удаляет все сессии пользователя
func (s *AuthService) LogoutAll(ctx context.Context, targetID uuid.UUID) error {
	const op = "service.auth.LogoutAll"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Starting logout user")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		logger.Error("cant parse userID")
		return fmt.Errorf("%s: %w", op, billingerr.ErrLogout)
	}

	if targetID == uuid.Nil {
		targetID = userID
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, targetID, models.SessionTerminate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.SessionTerminate),
		)
		return fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.SessionTerminate),
		)
		return fmt.Errorf("permission denied %v: %w", op, err)
	}

	sessions, err := s.sessionManage.GetAll(ctx, targetID)
	if err != nil {
		logger.Error("cant get sessions for logout all", zap.Error(err))

		return fmt.Errorf("%s: %w", op, billingerr.ErrLogout)
	}

	err = s.sessionManage.LogoutAll(ctx, targetID)
	if err != nil {
		logger.Error("cant logout",
			zap.String("userID", targetID.String()),
			zap.Error(err),
		)

		return fmt.Errorf("%s: %w", op, billingerr.ErrLogout)
	}
	for _, session := range sessions {
		// добавляем id удалённых сессии в Blacklist
		cacheKey := fmt.Sprintf("invalid_session:%s", session.ID.String())
		ttl := time.Until(session.ExpiresAt) + 5*time.Minute
		if err := s.cache.SetWithTTL(ctx, cacheKey, []byte("1"), ttl); err != nil {
			logger.Warn("cant add session to blacklist",
				zap.String("userID", targetID.String()),
				zap.String("sessionID", session.ID.String()),
				zap.Error(err),
			)
			return fmt.Errorf("%s: %w", op, billingerr.ErrLogout)
		}
	}

	return nil
}

// GetAllSessions - возвращает все сессии пользователя
func (s *AuthService) GetAllSessions(ctx context.Context, targetID uuid.UUID) ([]models.Session, error) {
	const op = "service.auth.GetAllSessions"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Getting sessions")

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		logger.Error("cant parse userID")
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrLogout)
	}

	if targetID == uuid.Nil {
		targetID = userID
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, targetID, models.SessionTerminate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.SessionTerminate),
		)
		return nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.SessionTerminate),
		)
		return nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	sessions, err := s.sessionManage.GetAll(ctx, targetID)
	if err != nil {
		logger.Error("cant get sessions", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrLogout)
	}

	return sessions, nil
}
