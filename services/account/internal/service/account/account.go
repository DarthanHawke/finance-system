// Пакет service предоставляет бизнес-логику для управления счетом и балансом счета
package service

import (
	"account-service/internal/lib/errors/apperr"
	"account-service/internal/models"
	"errors"

	"context"
	"fmt"

	"go.uber.org/zap"
)

// AccountService реализует бизнес-логику работы со счетами
type AccountService struct {
	accountManager AccountManager
	ibanGenerator  IbanGenerator
	logger         *zap.Logger
}

// NewAccountService создает новый экземпляр AccountService
func NewAccountService(
	accountManager AccountManager,
	ibanGenerator IbanGenerator,
	logger *zap.Logger,
) *AccountService {
	return &AccountService{
		accountManager: accountManager,
		ibanGenerator:  ibanGenerator,
		logger:         logger.With(zap.String("component", "account")),
	}
}

// AccountManager определяет методы управления счетами
type AccountManager interface {
	CreateAccount(ctx context.Context, req models.CreateAccountRequest) error
	GetAccount(ctx context.Context, req models.GetAccountRequest) (models.GetAccountResponse, error)
	GetAccounts(ctx context.Context, req models.GetAccountsRequest) (models.GetAccountsResponse, error)
	BlockAccount(ctx context.Context, req models.UpdateAccountRequest) error
	CloseAccount(ctx context.Context, req models.UpdateAccountRequest) error
}

// IbanGenerator определяет метод генерации IBAN
type IbanGenerator interface {
	Generate(userID string) (string, error)
}

// CreateAccount создает новый счет пользователя
func (s *AccountService) CreateAccount(ctx context.Context, req models.CreateAccountRequest) error {
	const op = "service.account.CreateAccount"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("code:", req.Code),
		zap.String("name:", req.Name),
		zap.String("user ID:", req.UserID.String()),
		zap.String("currency:", req.Currency),
	)

	logger.Info("creating new account")

	accountCode, err := s.ibanGenerator.Generate(req.UserID.String())
	if err != nil {
		logger.Error("failed to gen account id",
			zap.Error(err),
		)
		return fmt.Errorf("%s: %w", op, err)
	}
	req.Code = accountCode

	err = s.accountManager.CreateAccount(ctx, req)
	if err != nil {
		if errors.Is(err, apperr.ErrAccountCodeNotUnique) {
			logger.Error("collision in gen IBAN number",
				zap.String("account code:", accountCode),
				zap.Error(err),
			)
			return apperr.ErrAccountCodeNotUnique
		}
		logger.Error("failed to create account",
			zap.Error(err),
			zap.String("account code:", accountCode),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	logger.Debug("Successfully created account")

	return nil
}

// GetAccount возвращает полную информацию о счете
func (s *AccountService) GetAccount(
	ctx context.Context,
	req models.GetAccountRequest,
) (models.GetAccountResponse, error) {
	const op = "service.account.GetAccount"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("account code:", req.Code),
	)

	logger.Info("getting account")

	resp, err := s.accountManager.GetAccount(ctx, req)
	if err != nil {
		logger.Error("failed to get account",
			zap.Error(err),
		)
		return models.GetAccountResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	logger.Debug("Successfully got account",
		zap.String("status:", resp.Status),
	)

	return resp, nil
}

// GetAccounts возвращает все счета пользователя
func (s *AccountService) GetAccounts(
	ctx context.Context,
	req models.GetAccountsRequest,
) (models.GetAccountsResponse, error) {
	const op = "service.account.GetAccounts"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("user ID:", req.UserID.String()),
	)

	logger.Info("getting accounts")

	resp, err := s.accountManager.GetAccounts(ctx, req)
	if err != nil {
		logger.Error("failed to get accounts",
			zap.Error(err),
		)
		return models.GetAccountsResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	logger.Debug("Successfully got accounts")

	return resp, nil
}

// BlockAccount размораживает средства и блокирует счет для проведения любых операций
func (s *AccountService) BlockAccount(ctx context.Context, req models.UpdateAccountRequest) error {
	const op = "service.account.BlockAccount"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("account code:", req.Code),
	)

	logger.Info("blocking account")

	err := s.accountManager.BlockAccount(ctx, req)
	if err != nil {
		logger.Error("failed to block account",
			zap.Error(err),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	logger.Debug("Successfully blocked account")

	return nil
}

// CloseAccount закрывает счет если баланс нулевой
func (s *AccountService) CloseAccount(ctx context.Context, req models.UpdateAccountRequest) error {
	const op = "service.account.CloseAccount"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("account code:", req.Code),
	)

	logger.Info("closing account")

	err := s.accountManager.CloseAccount(ctx, req)
	if err != nil {
		logger.Error("failed to close account",
			zap.Error(err),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	logger.Debug("Successfully closed account")

	return nil
}
