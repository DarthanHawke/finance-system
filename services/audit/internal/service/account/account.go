package account

/* legacy code
import (
	billingerr "billing-service/internal/lib/errors"
	"errors"

	"billing-service/internal/models"
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	floatNil  = 0.0
	stringNil = ""
)

type AccountService struct {
	accountManage AccountManage
	roleManage    RoleManage
	ibanGenerator IbanGenerator
	logger        *zap.Logger
}

func NewAccountService(
	accountManage AccountManage,
	roleManage RoleManage,
	ibanGenerator IbanGenerator,
	logger *zap.Logger,
) *AccountService {
	return &AccountService{
		accountManage: accountManage,
		roleManage:    roleManage,
		ibanGenerator: ibanGenerator,
		logger:        logger.With(zap.String("component", "account_service")),
	}
}

type AccountManage interface {
	CreateAccount(ctx context.Context, accountCode string, userID uuid.UUID, currency, name string) error
	GetAccount(ctx context.Context, accountCode string) (*models.CurrencyAccount, error)
	GetAccountId(ctx context.Context, account_code string) (uuid.UUID, error)
	GetUserAccounts(ctx context.Context, userID uuid.UUID) ([]models.CurrencyAccount, error)
	GetBalance(ctx context.Context, accountCode string) (float64, error)
}

type RoleManage interface {
	CreateEntityWithID(ctx context.Context, id uuid.UUID, entityType string) error
	CreateRelation(ctx context.Context, sourceID, targetID uuid.UUID, relationType string) error
	CheckPermission(ctx context.Context, subjectID, objectID uuid.UUID, permissionName string) (bool, error)
	GetEntityID(ctx context.Context, entityType string) (uuid.UUID, error)
}

type IbanGenerator interface {
	Generate(userID string) (string, error)
}

// CreateAccount создает новый валютный счет для пользователя
func (s *AccountService) CreateAccount(ctx context.Context, currency, name string) (string, error) {
	const op = "service.account.CreateAccount"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("currency", currency),
		zap.String("name", name),
	)

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return stringNil, fmt.Errorf("%s: %w", op, billingerr.ErrInvalidUserID)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, userID, models.AccountCreate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.AccountCreate),
		)
		return stringNil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.AccountCreate),
		)
		return stringNil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	accountCode, err := s.ibanGenerator.Generate(userID.String())
	if err != nil {
		logger.Error("failed to gen account id",
			zap.Error(err),
			zap.String("userID", userID.String()),
		)
		return stringNil, fmt.Errorf("%s: %w", op, err)
	}

	// Создание счета
	err = s.accountManage.CreateAccount(ctx, accountCode, userID, currency, name)
	if err != nil {
		if errors.Is(err, billingerr.ErrAccountCodeUnique) {
			logger.Error("collision in gen IBAN number",
				zap.Error(err),
				zap.String("accountCode", accountCode),
				zap.String("userID", userID.String()),
			)
			return stringNil, fmt.Errorf("%s: %w", op, err)
		}
		logger.Error("failed to create account",
			zap.Error(err),
			zap.String("accountCode", accountCode),
			zap.String("userID", userID.String()),
		)
		return stringNil, fmt.Errorf("%s: %w", op, err)
	}

	accountId, err := s.accountManage.GetAccountId(ctx, accountCode)
	if err != nil {
		logger.Error("failed to get account uuid",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Account code", accountCode),
		)
		return stringNil, fmt.Errorf("failed to get account uuid: %v: %w", op, err)
	}

	err = s.roleManage.CreateEntityWithID(ctx, accountId, models.TransactionEntity)
	if err != nil {
		return stringNil, fmt.Errorf("failed to create user entity in ReBAC: %v", err)
	}

	err = s.roleManage.CreateRelation(ctx, userID, accountId, models.ClientRelationType)
	if err != nil {
		logger.Error("failed to create self-ownership relation in ReBAC", zap.Error(err))
		return stringNil, fmt.Errorf("failed to create self-ownership relation in ReBAC: %v", err)
	}
	return accountCode, nil
}

// GetAccount возвращает информацию о счете
func (s *AccountService) GetAccount(ctx context.Context, accountCode string) (*models.CurrencyAccount, error) {
	const op = "service.account.GetAccount"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("accountCode", accountCode),
	)

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrInvalidUserID)
	}

	accountId, err := s.accountManage.GetAccountId(ctx, accountCode)
	if err != nil {
		logger.Error("failed to get account uuid",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Account code", accountCode),
		)
		return nil, fmt.Errorf("failed to get account uuid: %v: %w", op, err)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, accountId, models.AccountRead)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.AccountRead),
		)
		return nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.AccountRead),
		)
		return nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	account, err := s.accountManage.GetAccount(ctx, accountCode)
	if err != nil {
		logger.Error("failed to get account",
			zap.Error(err),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return account, nil
}

// GetUserAccounts возвращает все счета пользователя
func (s *AccountService) GetUserAccounts(ctx context.Context, targetID uuid.UUID) ([]models.CurrencyAccount, error) {
	const op = "service.account.GetUserAccounts"

	logger := s.logger.With(
		zap.String("op", op),
	)

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, billingerr.ErrInvalidUserID)
	}

	if targetID == uuid.Nil {
		targetID = userID
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, userID, models.AccountRead)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.AccountRead),
		)
		return nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.AccountRead),
		)
		return nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	accounts, err := s.accountManage.GetUserAccounts(ctx, targetID)
	if err != nil {
		logger.Error("failed to get user accounts",
			zap.Error(err),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return accounts, nil
}

// GetBalance возвращает баланс счета
func (s *AccountService) GetBalance(ctx context.Context, accountCode string) (float64, error) {
	const op = "service.account.GetBalance"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("accountCode", accountCode),
	)

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return floatNil, fmt.Errorf("%s: %w", op, billingerr.ErrInvalidUserID)
	}

	accountId, err := s.accountManage.GetAccountId(ctx, accountCode)
	if err != nil {
		logger.Error("failed to get account uuid",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Account code", accountCode),
		)
		return floatNil, fmt.Errorf("failed to get account uuid: %v: %w", op, err)
	}

	allowed, err := s.roleManage.CheckPermission(ctx, userID, accountId, models.AccountRead)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.AccountRead),
		)
		return floatNil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.AccountRead),
		)
		return floatNil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	// Проверяем, что счет принадлежит пользователю
	account, err := s.accountManage.GetAccount(ctx, accountCode)
	if err != nil {
		return floatNil, fmt.Errorf("%s: %w", op, err)
	}
	if account.UserID != userID {
		return floatNil, fmt.Errorf("%s: %w", op, billingerr.ErrAccountNotFound)
	}

	// Получение баланса
	balance, err := s.accountManage.GetBalance(ctx, accountCode)
	if err != nil {
		logger.Error("failed to get balance",
			zap.Error(err),
		)
		return floatNil, fmt.Errorf("%s: %w", op, err)
	}

	return balance, nil
}
*/
