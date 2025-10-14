// Пакет service предоставляет бизнес-логику для управления платежами
package service

import (
	"context"
	"errors"
	"fmt"
	"payment-service/internal/lib/errors/apperr"
	"payment-service/internal/models"

	"go.uber.org/zap"
)

// PaymentService реализует бизнес-логику работы с платежами
type OperationService struct {
	operationManager OperationManager
	logger           *zap.Logger
}

// NewOperationService создает новый экземпляр PaymentService
func NewOperationService(
	operationManager OperationManager,
	logger *zap.Logger,
) *OperationService {
	return &OperationService{
		operationManager: operationManager,
		logger:           logger.With(zap.String("component", "payment_service")),
	}
}

// OperationManager определяет методы управления платежами
type OperationManager interface {
	GetPayment(ctx context.Context, req *models.GetPaymentRequest) (models.GetPaymentResponse, error)
	GetPayments(ctx context.Context, req *models.GetPaymentsRequest) (models.GetPaymentsResponse, error)
}

// GetPayment возвращает полную информацию о платеже
func (s *OperationService) GetPayment(
	ctx context.Context,
	req *models.GetPaymentRequest,
) (models.GetPaymentResponse, error) {
	const op = "service.payment.GetPayment"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("payment ID:", req.ID.String()),
	)
	logger.Info("getting payment")

	resp, err := s.operationManager.GetPayment(ctx, req)
	if err != nil {
		if errors.Is(err, apperr.ErrPaymentNotFound) {
			logger.Warn("payment not found",
				zap.Error(err),
			)
			return models.GetPaymentResponse{}, apperr.ErrPaymentNotFound
		}
		logger.Error("failed to get payment",
			zap.Error(err),
		)
		return models.GetPaymentResponse{}, fmt.Errorf("failed to get payment: %w", err)
	}

	logger.Debug("Successfully got payment",
		zap.String("status:", resp.Status),
	)

	return resp, nil
}

// GetPayments возвращает все платежи связанные со счетом
func (s *OperationService) GetPayments(
	ctx context.Context,
	req *models.GetPaymentsRequest,
) (models.GetPaymentsResponse, error) {
	const op = "service.payment.GetPayments"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("account code:", req.AccountCode),
	)
	logger.Info("getting payment")

	resp, err := s.operationManager.GetPayments(ctx, req)
	if err != nil {
		if errors.Is(err, apperr.ErrPaymentNotFound) {
			logger.Warn("payment not found",
				zap.Error(err),
			)
			return models.GetPaymentsResponse{}, apperr.ErrPaymentNotFound
		}
		logger.Error("failed to get payment",
			zap.Error(err),
		)
		return models.GetPaymentsResponse{}, fmt.Errorf("failed to get payment: %w", err)
	}

	logger.Debug("Successfully got payments")

	return resp, nil
}
