package transaction

import (
	billingerr "billing-service/internal/lib/errors"
	"billing-service/internal/models"
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TransactionService struct {
	internalTransaction InternalTransaction
	accountManage       AccountManage
	roleManage          RoleManage
	logger              *zap.Logger
}

func NewTransactionService(
	internalTransaction InternalTransaction,
	accountManage AccountManage,
	roleManage RoleManage,
	logger *zap.Logger,
) *TransactionService {
	return &TransactionService{
		internalTransaction: internalTransaction,
		accountManage:       accountManage,
		roleManage:          roleManage,
		logger:              logger.With(zap.String("component", "billing_service")),
	}
}

type InternalTransaction interface {
	ConvertCurrency(ctx context.Context, operationID uuid.UUID, fromaccountCode, toaccountCode *models.CurrencyAccount, amount float64, description string) error
	UpdateCurrencyRate(ctx context.Context, fromCurrency, toCurrency string, rate float64) error
	GetCurrencyRate(ctx context.Context, fromCurrency, toCurrency string) (float64, error)
}

type AccountManage interface {
	GetAccount(ctx context.Context, accountCode string) (*models.CurrencyAccount, error)
}

type RoleManage interface {
	CreateEntityWithID(ctx context.Context, id uuid.UUID, entityType string) error
	CreateRelation(ctx context.Context, sourceID, targetID uuid.UUID, relationType string) error
	GetEntityID(ctx context.Context, entityType string) (uuid.UUID, error)
	CheckPermission(ctx context.Context, subjectID, objectID uuid.UUID, permissionName string) (bool, error)
}

// ConvertCurrency конвертирует средства между счетами в разных валютах
func (s *TransactionService) ConvertCurrency(ctx context.Context, fromaccountCode, toaccountCode string, amount float64, description string) (uuid.UUID, error) {
	const op = "service.transaction.ConvertCurrency"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("fromaccountCode", fromaccountCode),
		zap.String("toaccountCode", toaccountCode),
		zap.Float64("amount", amount),
	)

	if amount <= 0 {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrInvalidAmount)
	}

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrInvalidUserID)
	}

	// Проверка прав доступа
	allowed, err := s.roleManage.CheckPermission(ctx, userID, userID, models.TransactionCreate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.TransactionCreate),
		)
		return uuid.Nil, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.TransactionCreate),
		)
		return uuid.Nil, fmt.Errorf("permission denied %v: %w", op, err)
	}

	// Проверяем, что оба счета принадлежат пользователю
	fromAccount, err := s.accountManage.GetAccount(ctx, fromaccountCode)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}
	if fromAccount.UserID != userID {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrAccountNotFound)
	}

	toAccount, err := s.accountManage.GetAccount(ctx, toaccountCode)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}
	if toAccount.UserID != userID {
		return uuid.Nil, fmt.Errorf("%s: %w", op, billingerr.ErrAccountNotFound)
	}

	// Генерируем ID операции для идемпотентности
	operationID := uuid.New()

	// Выполняем конвертацию
	if err := s.internalTransaction.ConvertCurrency(ctx, operationID, fromAccount, toAccount, amount, description); err != nil {
		logger.Error("failed to convert currency",
			zap.Error(err),
		)
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return operationID, nil
}

// UpdateCurrencyRate обновляет курс валют (админская функция)
func (s *TransactionService) UpdateCurrencyRate(ctx context.Context, fromCurrency, toCurrency string, rate float64) error {
	const op = "service.transaction.UpdateCurrencyRate"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("fromCurrency", fromCurrency),
		zap.String("toCurrency", toCurrency),
		zap.Float64("rate", rate),
	)

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return fmt.Errorf("%s: %w", op, billingerr.ErrInvalidUserID)
	}

	currencyEntityId, err := s.roleManage.GetEntityID(ctx, models.CurrencyEntity)
	if err != nil {
		return fmt.Errorf("%s: failed to get transaction system entity: %w", op, err)
	}

	// Проверка прав доступа (только для администраторов)
	allowed, err := s.roleManage.CheckPermission(ctx, userID, currencyEntityId, models.CurrencyUpdate)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.CurrencyUpdate),
		)
		return fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.CurrencyUpdate),
		)
		return fmt.Errorf("permission denied %v: %w", op, err)
	}

	if err := s.internalTransaction.UpdateCurrencyRate(ctx, fromCurrency, toCurrency, rate); err != nil {
		logger.Error("failed to update currency rate",
			zap.Error(err),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetCurrencyRate возвращает текущий курс обмена между валютами
func (s *TransactionService) GetCurrencyRate(
	ctx context.Context,
	fromCurrency, toCurrency string,
) (float64, error) {
	const op = "service.transaction.GetCurrencyRate"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("fromCurrency", fromCurrency),
		zap.String("toCurrency", toCurrency),
	)

	userID, ok := ctx.Value(models.UserIDKey).(uuid.UUID)
	if !ok {
		return 0, fmt.Errorf("%s: %w", op, billingerr.ErrInvalidUserID)
	}

	// Проверка прав доступа
	allowed, err := s.roleManage.CheckPermission(ctx, userID, userID, models.CurrencyRead)
	if err != nil {
		logger.Error("failed to check permission",
			zap.Error(err),
			zap.String("UserID", userID.String()),
			zap.String("Permission name", models.CurrencyRead),
		)
		return 0, fmt.Errorf("failed to check permission: %v: %w", op, err)
	}
	if !allowed {
		logger.Error("permission denied",
			zap.Error(err),
			zap.String("Permission name", models.CurrencyRead),
		)
		return 0, fmt.Errorf("permission denied %v: %w", op, err)
	}

	// Получение курса валют
	rate, err := s.internalTransaction.GetCurrencyRate(ctx, fromCurrency, toCurrency)
	if err != nil {
		logger.Error("failed to get currency rate",
			zap.Error(err),
		)
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	if rate <= 0 {
		logger.Warn("invalid currency rate",
			zap.Float64("rate", rate),
		)
		return 0, fmt.Errorf("%s: %w", op, billingerr.ErrCurrencyRateNotFound)
	}

	return rate, nil
}
