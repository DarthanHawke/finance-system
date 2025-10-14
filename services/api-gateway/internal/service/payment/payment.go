package payment

import (
	"client-service/internal/models"
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PaymentService struct {
	paymentManage PaymentManage
	logger        *zap.Logger
}

func NewPaymentService(
	paymentManage PaymentManage,
	logger *zap.Logger,
) *PaymentService {
	return &PaymentService{
		paymentManage: paymentManage,
		logger:        logger.With(zap.String("component", "client_service")),
	}
}

type PaymentManage interface {
	Transfer(ctx context.Context, sender, receiver string, amount float64, description string) (uuid.UUID, error)
	Deposit(ctx context.Context, accountID string, amount float64, description string) (uuid.UUID, error)
	GetPayment(ctx context.Context, paymentID uuid.UUID) (*models.Payment, error)
	GetAllPayment(ctx context.Context, userID uuid.UUID) ([]models.Payment, error)
	UpdateStatusPayment(ctx context.Context, paymentID uuid.UUID, status string) error
	CancelPayment(ctx context.Context, paymentID uuid.UUID) error
	ConvertCurrency(ctx context.Context, fromAccountID, toAccountID string, amount float64, description string) (uuid.UUID, error)
	GetOperationHistory(ctx context.Context, accountID string, limit, offset int) ([]models.BalanceOperation, error)
	UpdateCurrencyRate(ctx context.Context, fromCurrency, toCurrency string, rate float64) error
	GetCurrencyRate(ctx context.Context, fromCurrency, toCurrency string) (float64, error)
}

// Transfer создаёт платёж пользователя, предварительно валидируя доступ
func (s *PaymentService) Transfer(ctx context.Context, req models.TransferRequest) (*models.Payment, error) {
	const op = "service.payment.Transfer"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Creating payment")

	paymentId, err := s.paymentManage.Transfer(ctx, req.Sender, req.Receiver, req.Amount, req.Description)
	if err != nil {
		logger.Error("cant create payment", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return s.paymentManage.GetPayment(ctx, paymentId)
}

// Deposit пополняет баланс
func (s *PaymentService) Deposit(ctx context.Context, req models.DepositRequest) (uuid.UUID, error) {
	const op = "service.payment.Deposit"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("account_id", req.AccountID),
		zap.Float64("amount", req.Amount),
	)

	logger.Info("Processing deposit")

	paymentID, err := s.paymentManage.Deposit(ctx, req.AccountID, req.Amount, req.Description)
	if err != nil {
		logger.Error("failed to deposit", zap.Error(err))
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return paymentID, nil
}

// GetPayment - возращает информацию о платеже
func (s *PaymentService) GetPayment(ctx context.Context, req models.PaymentRequest) (*models.Payment, error) {
	const op = "service.payment.GetPayment"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Getting payment")

	payment, err := s.paymentManage.GetPayment(ctx, req.PaymentID)
	if err != nil {
		logger.Error("cant get payment", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return payment, nil
}

// GetAllPayment - возвращает все платежи пользователя
func (s *PaymentService) GetAllPayment(ctx context.Context, userID uuid.UUID) ([]models.Payment, error) {
	const op = "service.payment.GetAllPayment"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Getting payment")

	payments, err := s.paymentManage.GetAllPayment(ctx, userID)
	if err != nil {
		logger.Error("cant get payment", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return payments, nil
}

func (s *PaymentService) UpdateStatusPayment(ctx context.Context, req models.UpdateStatusRequest) error {
	const op = "service.payment.UpdateStatusPayment"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Updating payment")

	err := s.paymentManage.UpdateStatusPayment(ctx, req.PaymentID, req.Status)
	if err != nil {
		logger.Error("cant update payment", zap.Error(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *PaymentService) CancelPayment(ctx context.Context, req models.PaymentRequest) error {
	const op = "service.payment.UpdateStatusPayment"

	logger := s.logger.With(
		zap.String("op", op),
	)

	logger.Info("Canceling payment")

	err := s.paymentManage.CancelPayment(ctx, req.PaymentID)
	if err != nil {
		logger.Error("cant cancel payment", zap.Error(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// ConvertCurrency конвертирует валюту между счетами
func (s *PaymentService) ConvertCurrency(ctx context.Context, req models.ConvertCurrencyRequest) (uuid.UUID, error) {
	const op = "service.payment.ConvertCurrency"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("from_account_id", req.FromAccountID),
		zap.String("to_account_id", req.ToAccountID),
		zap.Float64("amount", req.Amount),
	)

	logger.Info("Processing currency conversion")

	operationID, err := s.paymentManage.ConvertCurrency(ctx, req.FromAccountID, req.ToAccountID, req.Amount, req.Description)
	if err != nil {
		logger.Error("failed to convert currency", zap.Error(err))
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return operationID, nil
}

// GetOperationHistory возвращает историю операций
func (s *PaymentService) GetOperationHistory(ctx context.Context, req models.OperationHistoryRequest) ([]models.BalanceOperation, error) {
	const op = "service.payment.GetOperationHistory"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("account_id", req.AccountID),
		zap.Int("limit", req.Limit),
		zap.Int("offset", req.Offset),
	)

	logger.Info("Getting operation history")

	operations, err := s.paymentManage.GetOperationHistory(ctx, req.AccountID, req.Limit, req.Offset)
	if err != nil {
		logger.Error("failed to get operation history", zap.Error(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return operations, nil
}

// UpdateCurrencyRate обновляет курс валют
func (s *PaymentService) UpdateCurrencyRate(ctx context.Context, req models.UpdateCurrencyRateRequest) error {
	const op = "service.payment.UpdateCurrencyRate"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("from_currency", req.FromCurrency),
		zap.String("to_currency", req.ToCurrency),
		zap.Float64("rate", req.Rate),
	)

	logger.Info("Updating currency rate")

	err := s.paymentManage.UpdateCurrencyRate(ctx, req.FromCurrency, req.ToCurrency, req.Rate)
	if err != nil {
		logger.Error("failed to update currency rate", zap.Error(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetCurrencyRate возвращает текущий курс обмена между валютами
func (s *PaymentService) GetCurrencyRate(
	ctx context.Context,
	req models.GetCurrencyRateRequest,
) (float64, error) {
	const op = "service.payment.GetCurrencyRate"

	logger := s.logger.With(
		zap.String("op", op),
		zap.String("from_currency", req.FromCurrency),
		zap.String("to_currency", req.ToCurrency),
	)

	logger.Info("Getting currency rate")

	rate, err := s.paymentManage.GetCurrencyRate(ctx, req.FromCurrency, req.ToCurrency)
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
