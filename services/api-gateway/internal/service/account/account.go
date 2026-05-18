package account

/*
import (
	"client-service/internal/models"
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AccountService struct {
	accountManage AccountManage
	logger        *zap.Logger
}

func NewAccountService(
	accountManage AccountManage,
	logger *zap.Logger,
) *AccountService {
	return &AccountService{
		accountManage: accountManage,
		logger:        logger.With(zap.String("component", "client_service")),
	}
}

type AccountManage interface {
	CreateAccount(ctx context.Context, currency, name string) (string, error)
	GetAccount(ctx context.Context, accountID string) (*models.CurrencyAccount, error)
	GetUserAccounts(ctx context.Context, userID uuid.UUID) ([]models.CurrencyAccount, error)
	GetBalance(ctx context.Context, accountID string) (float64, error)
}

// CreateAccount создает новый счет с указанной валютой и именем
func (s *AccountService) CreateAccount(ctx context.Context, req models.CreateAccountRequest) (string, error) {
	const op = "service.account.CreateAccount"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("currency", req.Currency),
		zap.String("name", req.Name),
	)

	logger.Info("Creating account")

	accountID, err := s.accountManage.CreateAccount(ctx, req.Currency, req.Name)
	if err != nil {
		logger.Error("failed to create account", zap.Error(err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return accountID, nil
}

// GetAccount возвращает информацию об аккаунте
func (s *AccountService) GetAccount(ctx context.Context, req models.GetAccountRequest) (*models.CurrencyAccount, error) {
	const op = "service.account.GetAccount"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("account_id", req.AccountID),
	)

	logger.Info("Getting account")

	account, err := s.accountManage.GetAccount(ctx, req.AccountID)
	if err != nil {
		logger.Error("failed to get account", zap.Error(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return account, nil
}

// GetUserAccounts возвращает список аккаунтов пользователя
func (s *AccountService) GetUserAccounts(ctx context.Context, req models.GetUserAccountsRequest) ([]models.CurrencyAccount, error) {
	const op = "service.account.GetUserAccounts"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Getting user accounts")

	accounts, err := s.accountManage.GetUserAccounts(ctx, req.UserID)
	if err != nil {
		logger.Error("failed to get user accounts", zap.Error(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return accounts, nil
}

// GetBalance возвращает баланс по account_id
func (s *AccountService) GetBalance(ctx context.Context, req models.GetBalanceRequest) (float64, error) {
	const op = "service.account.GetBalance"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("account_id", req.AccountID),
	)

	logger.Info("Getting balance")

	balance, err := s.accountManage.GetBalance(ctx, req.AccountID)
	if err != nil {
		logger.Error("failed to get balance", zap.Error(err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return balance, nil
}
*/
