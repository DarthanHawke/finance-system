// Пакет service предоставляет бизнес-логику для управления платежами
package service

import (
	"context"
	"errors"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"
)

// OperationService реализует бизнес-логику работы с платежами
type OperationService struct {
	operationManager OperationManager
}

// NewOperationService создает новый экземпляр OperationService
func NewOperationService(
	operationManager OperationManager,
) *OperationService {
	return &OperationService{
		operationManager: operationManager,
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
	const op = "transaction.GetTransaction"

	resp, err := s.operationManager.GetTransaction(ctx, req)
	if err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return models.GetTransactionResponse{}, err
		}
		return models.GetTransactionResponse{}, &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return resp, nil
}

// GetTransactions возвращает все платежи связанные со счетом
func (s *OperationService) GetTransactions(
	ctx context.Context,
	req *models.GetTransactionsRequest,
) (models.GetTransactionsResponse, error) {
	const op = "transaction.GetTransactions"

	resp, err := s.operationManager.GetTransactions(ctx, req)
	if err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return models.GetTransactionsResponse{}, err
		}
		return models.GetTransactionsResponse{}, &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return resp, nil
}
