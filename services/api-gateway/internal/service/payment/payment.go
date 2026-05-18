package transaction

/*
import (
	"client-service/internal/models"
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TransactionService struct {
	transactionManage TransactionManage
	logger            *zap.Logger
}

func NewTransactionService(
	transactionManage TransactionManage,
	logger *zap.Logger,
) *TransactionService {
	return &TransactionService{
		transactionManage: transactionManage,
		logger:            logger.With(zap.String("component", "client_service")),
	}
}

type TransactionManage interface {
	Transfer(ctx context.Context, sender, receiver string, amount float64, description string) (uuid.UUID, error)
	Deposit(ctx context.Context, accountID string, amount float64, description string) (uuid.UUID, error)
	GetTransaction(ctx context.Context, transactionID uuid.UUID) (*models.Transaction, error)
	GetAllTransaction(ctx context.Context, userID uuid.UUID) ([]models.Transaction, error)
	UpdateStatusTransaction(ctx context.Context, transactionID uuid.UUID, status string) error
	CancelTransaction(ctx context.Context, transactionID uuid.UUID) error
	ConvertCurrency(ctx context.Context, fromAccountID, toAccountID string, amount float64, description string) (uuid.UUID, error)
	GetOperationHistory(ctx context.Context, accountID string, limit, offset int) ([]models.BalanceOperation, error)
	UpdateCurrencyRate(ctx context.Context, fromCurrency, toCurrency string, rate float64) error
	GetCurrencyRate(ctx context.Context, fromCurrency, toCurrency string) (float64, error)
}

// Transfer создаёт платёж пользователя, предварительно валидируя доступ
func (s *TransactionService) Transfer(ctx context.Context, req models.TransferRequest) (*models.Transaction, error) {
	const op = "service.transaction.Transfer"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Creating transaction")

	transactionId, err := s.transactionManage.Transfer(ctx, req.Sender, req.Receiver, req.Amount, req.Description)
	if err != nil {
		logger.Error("cant create transaction", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return s.transactionManage.GetTransaction(ctx, transactionId)
}

// Deposit пополняет баланс
func (s *TransactionService) Deposit(ctx context.Context, req models.DepositRequest) (uuid.UUID, error) {
	const op = "service.transaction.Deposit"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("account_id", req.AccountID),
		zap.Float64("amount", req.Amount),
	)

	logger.Info("Processing deposit")

	transactionID, err := s.transactionManage.Deposit(ctx, req.AccountID, req.Amount, req.Description)
	if err != nil {
		logger.Error("failed to deposit", zap.Error(err))
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return transactionID, nil
}

// GetTransaction - возращает информацию о платеже
func (s *TransactionService) GetTransaction(ctx context.Context, req models.TransactionRequest) (*models.Transaction, error) {
	const op = "service.transaction.GetTransaction"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Getting transaction")

	transaction, err := s.transactionManage.GetTransaction(ctx, req.TransactionID)
	if err != nil {
		logger.Error("cant get transaction", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return transaction, nil
}

// GetAllTransaction - возвращает все платежи пользователя
func (s *TransactionService) GetAllTransaction(ctx context.Context, userID uuid.UUID) ([]models.Transaction, error) {
	const op = "service.transaction.GetAllTransaction"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Getting transaction")

	transactions, err := s.transactionManage.GetAllTransaction(ctx, userID)
	if err != nil {
		logger.Error("cant get transaction", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return transactions, nil
}

func (s *TransactionService) UpdateStatusTransaction(ctx context.Context, req models.UpdateStatusRequest) error {
	const op = "service.transaction.UpdateStatusTransaction"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Updating transaction")

	err := s.transactionManage.UpdateStatusTransaction(ctx, req.TransactionID, req.Status)
	if err != nil {
		logger.Error("cant update transaction", zap.Error(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *TransactionService) CancelTransaction(ctx context.Context, req models.TransactionRequest) error {
	const op = "service.transaction.UpdateStatusTransaction"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Canceling transaction")

	err := s.transactionManage.CancelTransaction(ctx, req.TransactionID)
	if err != nil {
		logger.Error("cant cancel transaction", zap.Error(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// ConvertCurrency конвертирует валюту между счетами
func (s *TransactionService) ConvertCurrency(ctx context.Context, req models.ConvertCurrencyRequest) (uuid.UUID, error) {
	const op = "service.transaction.ConvertCurrency"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("from_account_id", req.FromAccountID),
		zap.String("to_account_id", req.ToAccountID),
		zap.Float64("amount", req.Amount),
	)

	logger.Info("Processing currency conversion")

	operationID, err := s.transactionManage.ConvertCurrency(ctx, req.FromAccountID, req.ToAccountID, req.Amount, req.Description)
	if err != nil {
		logger.Error("failed to convert currency", zap.Error(err))
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return operationID, nil
}

// GetOperationHistory возвращает историю операций
func (s *TransactionService) GetOperationHistory(ctx context.Context, req models.OperationHistoryRequest) ([]models.BalanceOperation, error) {
	const op = "service.transaction.GetOperationHistory"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("account_id", req.AccountID),
		zap.Int("limit", req.Limit),
		zap.Int("offset", req.Offset),
	)

	logger.Info("Getting operation history")

	operations, err := s.transactionManage.GetOperationHistory(ctx, req.AccountID, req.Limit, req.Offset)
	if err != nil {
		logger.Error("failed to get operation history", zap.Error(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return operations, nil
}

// UpdateCurrencyRate обновляет курс валют
func (s *TransactionService) UpdateCurrencyRate(ctx context.Context, req models.UpdateCurrencyRateRequest) error {
	const op = "service.transaction.UpdateCurrencyRate"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("from_currency", req.FromCurrency),
		zap.String("to_currency", req.ToCurrency),
		zap.Float64("rate", req.Rate),
	)

	logger.Info("Updating currency rate")

	err := s.transactionManage.UpdateCurrencyRate(ctx, req.FromCurrency, req.ToCurrency, req.Rate)
	if err != nil {
		logger.Error("failed to update currency rate", zap.Error(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetCurrencyRate возвращает текущий курс обмена между валютами
func (s *TransactionService) GetCurrencyRate(
	ctx context.Context,
	req models.GetCurrencyRateRequest,
) (float64, error) {
	const op = "service.transaction.GetCurrencyRate"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("from_currency", req.FromCurrency),
		zap.String("to_currency", req.ToCurrency),
	)

	logger.Info("Getting currency rate")

	rate, err := s.transactionManage.GetCurrencyRate(ctx, req.FromCurrency, req.ToCurrency)
	if err != nil {
		logger.Error("failed to get currency rate",
			zap.Error(err),
			zap.String("from_currency", req.FromCurrency),
			zap.String("to_currency", req.ToCurrency),
		)
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	if rate <= 0 {
		logger.Warn("invalid currency rate received",
			zap.Float64("rate", rate),
		)
		return 0, fmt.Errorf("%s: invalid currency rate", op)
	}

	return rate, nil
}
*/
