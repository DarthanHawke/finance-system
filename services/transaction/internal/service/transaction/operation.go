// Пакет service предоставляет бизнес-логику для управления платежами
package service

import (
	"context"
	"errors"
	"fmt"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"

	"go.uber.org/zap"
)

// TransactionService реализует бизнес-логику работы с платежами
type OperationService struct {
	operationManager OperationManager
	logger           *zap.Logger
}

// NewOperationService создает новый экземпляр TransactionService
func NewOperationService(
	operationManager OperationManager,
	logger *zap.Logger,
) *OperationService {
	return &OperationService{
		operationManager: operationManager,
		logger:           logger.With(zap.String("component", "transaction_service")),
	}
}

// OperationManager определяет методы управления платежами
type OperationManager interface {
	GetTransaction(ctx context.Context, req *models.GetTransactionRequest) (models.GetTransactionResponse, error)
	GetTransactions(ctx context.Context, req *models.GetTransactionsRequest) (models.GetTransactionsResponse, error)
}

// GetTransaction возвращает полную информацию о платеже
func (s *OperationService) GetTransaction(
	ctx context.Context,
	req *models.GetTransactionRequest,
) (models.GetTransactionResponse, error) {
	const op = "service.transaction.GetTransaction"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("transaction ID:", req.ID.String()),
	)
	logger.Info("getting transaction")

	resp, err := s.operationManager.GetTransaction(ctx, req)
	if err != nil {
		if errors.Is(err, apperr.ErrTransactionNotFound) {
			logger.Warn("transaction not found",
				zap.Error(err),
			)
			return models.GetTransactionResponse{}, apperr.ErrTransactionNotFound
		}
		logger.Error("failed to get transaction",
			zap.Error(err),
		)
		return models.GetTransactionResponse{}, fmt.Errorf("failed to get transaction: %w", err)
	}

	logger.Debug("Successfully got transaction",
		zap.String("status:", resp.Status),
	)

	return resp, nil
}

// GetTransactions возвращает все платежи связанные со счетом
func (s *OperationService) GetTransactions(
	ctx context.Context,
	req *models.GetTransactionsRequest,
) (models.GetTransactionsResponse, error) {
	const op = "service.transaction.GetTransactions"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("account code:", req.AccountCode),
	)
	logger.Info("getting transaction")

	resp, err := s.operationManager.GetTransactions(ctx, req)
	if err != nil {
		if errors.Is(err, apperr.ErrTransactionNotFound) {
			logger.Warn("transaction not found",
				zap.Error(err),
			)
			return models.GetTransactionsResponse{}, apperr.ErrTransactionNotFound
		}
		logger.Error("failed to get transaction",
			zap.Error(err),
		)
		return models.GetTransactionsResponse{}, fmt.Errorf("failed to get transaction: %w", err)
	}

	logger.Debug("Successfully got transactions")

	return resp, nil
}
