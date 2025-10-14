// Пакет service предоставляет бизнес-логику для управления счетом и балансом счета
package service

import (
	"account-service/internal/models"
	"time"

	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// BalanceService реализует бизнес-логику работы с балансом счета
type BalanceService struct {
	balanceManager BalanceManager
	producer       Producer
	logger         *zap.Logger
}

// NewBalanceService создает новый экземпляр BalanceService
func NewBalanceService(
	balanceManager BalanceManager,
	producer Producer,
	logger *zap.Logger,
) *BalanceService {
	return &BalanceService{
		balanceManager: balanceManager,
		producer:       producer,
		logger:         logger.With(zap.String("component", "account")),
	}
}

type Producer interface {
	ProduceMessage(ctx context.Context, topic string, key string, value any) error
}

// BalanceManager определяет методы управления балансом
type BalanceManager interface {
	FreezeBalance(ctx context.Context, req models.BalanceRequest) error
	UnfreezeBalance(ctx context.Context, req models.BalanceRequest) error
	ReserveDeposit(ctx context.Context, req models.BalanceRequest) error
	UnreserveDeposit(ctx context.Context, req models.BalanceRequest) error
	WithdrawBalance(ctx context.Context, req models.BalanceRequest) error
	DepositBalance(ctx context.Context, req models.BalanceRequest) error
}

// HandleFreezeRequest обрабатывает запрос на заморозку средств
func (s *BalanceService) HandleFreezeRequest(ctx context.Context, req models.BalanceRequest) error {
	const op = "service.account.HandleFreezeRequest"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("payment ID:", req.PaymentID.String()),
		zap.String("account code:", req.Code),
	)

	logger.Info("freeze balance started")

	event := models.BalanceResponseEvent{
		Message: models.Message{
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
			PaymentID: req.PaymentID.String(),
			Source:    "account-service",
		},
	}

	if err := s.balanceManager.FreezeBalance(ctx, req); err != nil {
		event.Success = false
		event.EventType = models.EventTypeFreezeFail
		event.Reason = err.Error()

		logger.Error("freeze balance failed",
			zap.Error(err),
		)

	} else {
		event.Success = true
		event.EventType = models.EventTypeFreezeComplete

		logger.Info("balance frozen successfully")
	}

	return s.producer.ProduceMessage(ctx, "payment-events", event.PaymentID, event)
}

// HandleUnfreezeRequest откатываем заморозку средств
func (s *BalanceService) HandleUnfreezeRequest(ctx context.Context, req models.BalanceRequest) error {
	const op = "service.account.HandleUnfreezeRequest"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("payment ID:", req.PaymentID.String()),
		zap.String("account code:", req.Code),
	)

	logger.Info("unfreeze balance started")

	event := models.BalanceResponseEvent{
		Message: models.Message{
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
			PaymentID: req.PaymentID.String(),
			Source:    "account-service",
		},
	}

	if err := s.balanceManager.UnfreezeBalance(ctx, req); err != nil {
		event.Success = false
		event.EventType = models.EventTypeFreezeFail
		event.Reason = err.Error()

		logger.Error("unfreeze balance failed",
			zap.Error(err),
		)

	} else {
		event.Success = true
		event.EventType = models.EventTypeFreezeComplete

		logger.Info("balance unfrozen successfully")
	}

	// Отправляем результат обратно в payment-service
	return s.producer.ProduceMessage(ctx, "payment-events", event.PaymentID, event)
}

// HandleReserveRequest обрабатывает запрос на резервирование средств
func (s *BalanceService) HandleReserveRequest(ctx context.Context, req models.BalanceRequest) error {
	const op = "service.account.HandleReserveRequest"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("payment ID:", req.PaymentID.String()),
		zap.String("account code:", req.Code),
	)

	logger.Info("reserve balance started")

	event := models.BalanceResponseEvent{
		Message: models.Message{
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
			PaymentID: req.PaymentID.String(),
			Source:    "account-service",
		},
	}

	if err := s.balanceManager.ReserveDeposit(ctx, req); err != nil {
		event.Success = false
		event.EventType = models.EventTypeReserveFail
		event.Reason = err.Error()

		logger.Error("reserve balance failed",
			zap.Error(err),
		)
	} else {
		event.Success = true
		event.EventType = models.EventTypeReserveComplete

		logger.Info("balance reserve successfully")
	}

	return s.producer.ProduceMessage(ctx, "payment-events", event.PaymentID, event)
}

// HandleUnreserveRequest откатываем резервирование средств
func (s *BalanceService) HandleUnreserveRequest(ctx context.Context, req models.BalanceRequest) error {
	const op = "service.account.HandleUnreserveRequest"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("payment ID:", req.PaymentID.String()),
		zap.String("account code:", req.Code),
	)

	logger.Info("unreserve balance started")

	event := models.BalanceResponseEvent{
		Message: models.Message{
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
			PaymentID: req.PaymentID.String(),
			Source:    "account-service",
		},
	}

	if err := s.balanceManager.UnreserveDeposit(ctx, req); err != nil {
		event.Success = false
		event.EventType = models.EventTypeReserveFail
		event.Reason = err.Error()

		logger.Error("unreserve balance failed",
			zap.Error(err),
		)
	} else {
		event.Success = true
		event.EventType = models.EventTypeReserveComplete

		logger.Info("balance unreserve successfully")
	}

	return s.producer.ProduceMessage(ctx, "payment-events", event.PaymentID, event)
}

// HandleWithdrawRequest коммитим списание средств (списываем из замороженных)
func (s *BalanceService) HandleWithdrawRequest(ctx context.Context, req models.BalanceRequest) error {
	const op = "service.account.HandleWithdrawRequest"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("payment ID:", req.PaymentID.String()),
		zap.String("account code:", req.Code),
	)

	logger.Info("withdraw balance started")

	event := models.BalanceResponseEvent{
		Message: models.Message{
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
			PaymentID: req.PaymentID.String(),
			Source:    "account-service",
		},
	}

	if err := s.balanceManager.WithdrawBalance(ctx, req); err != nil {
		event.Success = false
		event.EventType = models.EventTypeWithdrawFail
		event.Reason = err.Error()

		logger.Error("withdraw balance failed",
			zap.Error(err),
		)
	} else {
		event.Success = true
		event.EventType = models.EventTypeWithdrawComplete

		logger.Info("withdraw deposit successfully")
	}

	return s.producer.ProduceMessage(ctx, "payment-events", event.PaymentID, event)
}

// HandleDepositRequest коммитим пополнение средств (пополняем из зарезервированных)
func (s *BalanceService) HandleDepositRequest(ctx context.Context, req models.BalanceRequest) error {
	const op = "service.account.HandleDepositRequest"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("payment ID:", req.PaymentID.String()),
		zap.String("account code:", req.Code),
	)

	logger.Info("deposit balance started")

	event := models.BalanceResponseEvent{
		Message: models.Message{
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
			PaymentID: req.PaymentID.String(),
			Source:    "account-service",
		},
	}

	if err := s.balanceManager.DepositBalance(ctx, req); err != nil {
		event.Success = false
		event.EventType = models.EventTypeDepositFail
		event.Reason = err.Error()

		logger.Error("deposit balance failed",
			zap.Error(err),
		)
	} else {
		event.Success = true
		event.EventType = models.EventTypeDepositComplete

		logger.Info("balance deposit successfully")
	}

	return s.producer.ProduceMessage(ctx, "payment-events", event.PaymentID, event)
}
