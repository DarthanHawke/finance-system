// Пакет service предоставляет бизнес-логику для управления счетом и балансом счета
package service

import (
	"account-service/internal/models"
	"encoding/json"
	"fmt"
	"time"

	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// BalanceService реализует бизнес-логику работы с балансом счета
type BalanceService struct {
	balanceManager BalanceManager
	eventManager   EventManager
	logger         *zap.Logger
}

// NewBalanceService создает новый экземпляр BalanceService
func NewBalanceService(
	balanceManager BalanceManager,
	eventManager EventManager,
	logger *zap.Logger,
) *BalanceService {
	return &BalanceService{
		balanceManager: balanceManager,
		eventManager:   eventManager,
		logger:         logger.With(zap.String("component", "account")),
	}
}

// BalanceManager определяет методы управления балансом
type BalanceManager interface {
	FreezeBalance(ctx context.Context, req *models.BalanceRequest, event *models.CreateEventRequest) error
	UnfreezeBalance(ctx context.Context, req *models.BalanceRequest, event *models.CreateEventRequest) error
	ReserveDeposit(ctx context.Context, req *models.BalanceRequest, event *models.CreateEventRequest) error
	UnreserveDeposit(ctx context.Context, req *models.BalanceRequest, event *models.CreateEventRequest) error
	WithdrawBalance(ctx context.Context, req *models.BalanceRequest, event *models.CreateEventRequest) error
	DepositBalance(ctx context.Context, req *models.BalanceRequest, event *models.CreateEventRequest) error
}

// EventManager определяет методы управления платежами
type EventManager interface {
	CreateEvent(ctx context.Context, req *models.CreateEventRequest) error
}

// handleOperationError
func (s *BalanceService) handleOperationError(ctx context.Context, err error, event *models.CreateEventRequest) error {
	const op = "service.account.handleOperationError"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("payment ID:", event.PaymentID.String()),
	)

	payload := models.BalanceResponsePayload{
		Success: false,
		Reason:  err.Error(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	event.Payload = payloadBytes

	err = s.eventManager.CreateEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	logger.Error("freeze balance failed",
		zap.Error(err),
	)

	return nil
}

func (s *BalanceService) createSuccessEvent(paymentID uuid.UUID, eventType string) (*models.CreateEventRequest, error) {
	payload := models.BalanceResponsePayload{Success: true}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &models.CreateEventRequest{
		Event: &models.Event{
			ID:        uuid.New(),
			PaymentID: paymentID,
			Type:      eventType,
			CreatedAt: time.Now(),
			Source:    models.Source,
			Payload:   payloadBytes,
		},
	}, nil
}

func (s *BalanceService) extractBalanceRequest(event *models.Event) (*models.BalanceRequest, error) {
	var payloadReq models.BalanceRequestPayload
	if err := json.Unmarshal(event.Payload, &payloadReq); err != nil {
		return nil, err
	}

	return &models.BalanceRequest{
		Code:      payloadReq.AccountCode,
		Amount:    payloadReq.Amount,
		PaymentID: event.PaymentID,
	}, nil
}

// HandleFreezeRequest обрабатывает запрос на заморозку средств
func (s *BalanceService) HandleFreezeRequest(ctx context.Context, event *models.Event) error {
	const op = "service.account.HandleFreezeRequest"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("payment ID:", event.PaymentID.String()),
	)

	logger.Info("freeze balance started")

	successEvent, err := s.createSuccessEvent(event.PaymentID, models.EventFreezeResponse)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	balanceReq, err := s.extractBalanceRequest(event)
	if err != nil {
		return s.handleOperationError(ctx, err, successEvent)
	}

	if err := s.balanceManager.FreezeBalance(ctx, balanceReq, successEvent); err != nil {
		return s.handleOperationError(ctx, err, successEvent)
	}

	return nil
}

// HandleUnfreezeRequest откатываем заморозку средств
func (s *BalanceService) HandleUnfreezeRequest(ctx context.Context, event *models.Event) error {
	const op = "service.account.HandleUnfreezeRequest"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("payment ID:", event.PaymentID.String()),
	)

	logger.Info("unfreeze balance started")

	successEvent, err := s.createSuccessEvent(event.PaymentID, models.EventUnfreezeResponse)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	balanceReq, err := s.extractBalanceRequest(event)
	if err != nil {
		return s.handleOperationError(ctx, err, successEvent)
	}

	if err := s.balanceManager.UnfreezeBalance(ctx, balanceReq, successEvent); err != nil {
		return s.handleOperationError(ctx, err, successEvent)
	}

	return nil
}

// HandleReserveRequest обрабатывает запрос на резервирование средств
func (s *BalanceService) HandleReserveRequest(ctx context.Context, event *models.Event) error {
	const op = "service.account.HandleReserveRequest"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("payment ID:", event.PaymentID.String()),
	)

	logger.Info("reserve balance started")

	successEvent, err := s.createSuccessEvent(event.PaymentID, models.EventReserveResponse)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	balanceReq, err := s.extractBalanceRequest(event)
	if err != nil {
		return s.handleOperationError(ctx, err, successEvent)
	}

	if err := s.balanceManager.ReserveDeposit(ctx, balanceReq, successEvent); err != nil {
		return s.handleOperationError(ctx, err, successEvent)
	}

	return nil
}

// HandleUnreserveRequest откатываем резервирование средств
func (s *BalanceService) HandleUnreserveRequest(ctx context.Context, event *models.Event) error {
	const op = "service.account.HandleUnreserveRequest"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("payment ID:", event.PaymentID.String()),
	)

	logger.Info("unreserve balance started")

	successEvent, err := s.createSuccessEvent(event.PaymentID, models.EventUnreserveResponse)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	balanceReq, err := s.extractBalanceRequest(event)
	if err != nil {
		return s.handleOperationError(ctx, err, successEvent)
	}

	if err := s.balanceManager.UnreserveDeposit(ctx, balanceReq, successEvent); err != nil {
		return s.handleOperationError(ctx, err, successEvent)
	}

	return nil
}

// HandleWithdrawRequest коммитим списание средств (списываем из замороженных)
func (s *BalanceService) HandleWithdrawRequest(ctx context.Context, event *models.Event) error {
	const op = "service.account.HandleWithdrawRequest"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("payment ID:", event.PaymentID.String()),
	)

	logger.Info("withdraw balance started")

	successEvent, err := s.createSuccessEvent(event.PaymentID, models.EventWithdrawResponse)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	balanceReq, err := s.extractBalanceRequest(event)
	if err != nil {
		return s.handleOperationError(ctx, err, successEvent)
	}

	if err := s.balanceManager.WithdrawBalance(ctx, balanceReq, successEvent); err != nil {
		return s.handleOperationError(ctx, err, successEvent)
	}

	return nil
}

// HandleDepositRequest коммитим пополнение средств (пополняем из зарезервированных)
func (s *BalanceService) HandleDepositRequest(ctx context.Context, event *models.Event) error {
	const op = "service.account.HandleDepositRequest"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("payment ID:", event.PaymentID.String()),
	)

	logger.Info("deposit balance started")

	successEvent, err := s.createSuccessEvent(event.PaymentID, models.EventDepositResponse)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	balanceReq, err := s.extractBalanceRequest(event)
	if err != nil {
		return s.handleOperationError(ctx, err, successEvent)
	}

	if err := s.balanceManager.DepositBalance(ctx, balanceReq, successEvent); err != nil {
		return s.handleOperationError(ctx, err, successEvent)
	}

	return nil
}
