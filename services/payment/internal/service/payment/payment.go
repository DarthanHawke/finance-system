// Пакет service предоставляет бизнес-логику для управления платежами
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"payment-service/internal/models"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PaymentService реализует бизнес-логику работы с платежами
type PaymentService struct {
	paymentManager PaymentManager
	eventManager   EventManager
	producer       Producer
	logger         *zap.Logger
}

// NewPaymentService создает новый экземпляр PaymentService
func NewPaymentService(
	paymentManager PaymentManager,
	eventManager EventManager,
	producer Producer,
	logger *zap.Logger,
) *PaymentService {
	return &PaymentService{
		paymentManager: paymentManager,
		eventManager:   eventManager,
		producer:       producer,
		logger:         logger.With(zap.String("component", "payment_service")),
	}
}

type Producer interface {
	ProduceMessage(ctx context.Context, topic string, key string, value any) error
}

// PaymentManager определяет методы управления платежами
type PaymentManager interface {
	CreatePayment(ctx context.Context, req *models.CreatePaymentRequest) error
	GetPayment(ctx context.Context, req *models.GetPaymentRequest) (models.GetPaymentResponse, error)
	UpdatePaymentStatus(ctx context.Context, req *models.UpdatePaymentStatusRequest) error
}

// EventManager определяет методы управления платежами
type EventManager interface {
	CreateEvent(ctx context.Context, req *models.CreateEventRequest) error
	GetPendingEvents(ctx context.Context, req *models.GetEventRequest) (models.GetEventResponse, error)
	UpdateEventStatus(ctx context.Context, req *models.UpdateEventStatusRequest) error
}

// HandlePaymentRequest
func (s *PaymentService) HandlePaymentRequest(ctx context.Context, req *models.CreatePaymentRequest) error {
	const op = "service.payment.HandlePaymentRequest"

	logger := s.logger.With(
		zap.String("op:", op),
	)

	logger.Info("payment saga started")

	req.Payment.ID = uuid.New()
	req.Payment.Status = models.PaymentStatusPending

	payload := models.BalanceRequestPayload{
		Amount:   req.Amount,
		Currency: req.Currency,
	}

	var eventType string
	switch req.SenderType {
	case models.AccountInternal:
		eventType = models.EventFreezeRequest
		payload.AccountCode = req.SenderAccountCode
	case models.AccountExternal:
		eventType = models.EventReserveRequest
		payload.AccountCode = req.RecipientAccountCode
	default:
		eventType = "unknow operation"
		return fmt.Errorf("%s: %s", op, eventType)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	req.Event = &models.Event{
		ID:        uuid.New(),
		PaymentID: req.Payment.ID,
		Type:      eventType,
		CreatedAt: time.Now(),
		Source:    models.Source,
		Payload:   payloadBytes,
	}

	// TODO Stan, procesing code, auth code

	if err := s.paymentManager.CreatePayment(ctx, req); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	logger.Info("payment wait account",
		zap.String("event id:", req.Event.ID.String()),
		zap.String("event type:", eventType),
		zap.String("payment id:", req.Payment.ID.String()),
		zap.String("account code:", req.SenderAccountCode),
	)

	return nil
}

// HandleFreezeResult
func (s *PaymentService) HandleFreezeResult(ctx context.Context, resp models.Event) error {
	const op = "service.payment.HandleFreezeResult"

	logger := s.logger.With(
		zap.String("op:", op),
	)

	logger.Info("processing result of freeze balance")

	var payloadResp models.BalanceResponsePayload
	if err := json.Unmarshal([]byte(resp.Payload), &payloadResp); err != nil {
		if err := s.compensatingTryBalance(ctx, resp); err != nil {
			return fmt.Errorf("%s: compensating action failed: %w", op, err)
		}
		return fmt.Errorf("failed to unmarshal balance response payload: %w", err)
	}

	if !payloadResp.Success {
		if err := s.compensatingTryBalance(ctx, resp); err != nil {
			return fmt.Errorf("%s: compensating action failed: %w", op, err)
		}
	}

	payment, err := s.paymentManager.GetPayment(ctx, &models.GetPaymentRequest{ID: resp.PaymentID})
	if err != nil {
		// TODO здесь надо что-то придумать на случай ошибки, как минимум ретрай и сообщение в DLQ для ручного вмешательства
		return fmt.Errorf("%s: %w", op, err)
	}

	var eventType string
	var payload any

	switch payment.RecipientType {
	case models.AccountInternal:
		eventType = models.EventReserveRequest
		payload = models.BalanceRequestPayload{
			AccountCode: payment.RecipientAccountCode,
			Amount:      payment.Amount,
			Currency:    payment.Currency,
		}
	case models.AccountExternal:
		eventType = models.EventExternalRequest
		payload = models.ExternalRequestPayload{
			// TODO: ISOmessge
			ISOMessage: "Здесь будет ISO message",
		}
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	event := &models.CreateEventRequest{
		Event: &models.Event{
			ID:        uuid.New(),
			PaymentID: payment.ID,
			Type:      eventType,
			CreatedAt: time.Now(),
			Source:    models.Source,
			Payload:   payloadBytes,
		},
	}

	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		// TODO здесь надо что-то придумать на случай ошибки, как минимум ретрай и сообщение в DLQ для ручного вмешательства
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// HandleUnfreezeResult
func (s *PaymentService) HandleUnfreezeResult(ctx context.Context, resp models.Event) error {
	const op = "service.payment.HandleUnfreezeResult"

	logger := s.logger.With(
		zap.String("op:", op),
	)

	logger.Info("processing result of unfreeze balance")

	var payloadResp models.BalanceResponsePayload
	if err := json.Unmarshal([]byte(resp.Payload), &payloadResp); err != nil {
		//TODO компенсирующие действия для Unfreeze fail
		return fmt.Errorf("failed to unmarshal balance response payload: %w", err)
	}

	if !payloadResp.Success {
		//TODO компенсирующие действия Unfreeze fail
	}

	req := &models.UpdatePaymentStatusRequest{
		ID:     resp.PaymentID,
		Status: models.PaymentStatusCancelled,
	}
	if err := s.paymentManager.UpdatePaymentStatus(ctx, req); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// HandleReserveResult
func (s *PaymentService) HandleReserveResult(ctx context.Context, resp models.Event) error {
	const op = "service.payment.HandleReserveResult"

	logger := s.logger.With(
		zap.String("op:", op),
	)

	logger.Info("processing result of reserve balance")

	var payloadResp models.BalanceResponsePayload
	if err := json.Unmarshal([]byte(resp.Payload), &payloadResp); err != nil {
		if err := s.compensatingTryBalance(ctx, resp); err != nil {
			return fmt.Errorf("%s: compensating action failed: %w", op, err)
		}
		return fmt.Errorf("failed to unmarshal balance response payload: %w", err)
	}

	if !payloadResp.Success {
		if err := s.compensatingTryBalance(ctx, resp); err != nil {
			return fmt.Errorf("%s: compensating action failed: %w", op, err)
		}
	}

	payment, err := s.paymentManager.GetPayment(ctx, &models.GetPaymentRequest{ID: resp.PaymentID})
	if err != nil {
		// TODO здесь надо что-то придумать на случай ошибки, как минимум ретрай и сообщение в DLQ для ручного вмешательства
		return fmt.Errorf("%s: %w", op, err)
	}

	var eventType string
	var payload any

	switch payment.SenderType {
	case models.AccountInternal:
		eventType = models.EventWithdrawRequest
		payload = models.BalanceRequestPayload{
			AccountCode: payment.SenderAccountCode,
			Amount:      payment.Amount,
			Currency:    payment.Currency,
		}
	case models.AccountExternal:
		eventType = models.EventExternalRequest
		payload = models.ExternalRequestPayload{
			// TODO: ISOmessge
			ISOMessage: "Здесь будет ISO message",
		}
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	event := &models.CreateEventRequest{
		Event: &models.Event{
			ID:        uuid.New(),
			PaymentID: payment.ID,
			Type:      eventType,
			CreatedAt: time.Now(),
			Source:    models.Source,
			Payload:   payloadBytes,
		},
	}

	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		// TODO здесь надо что-то придумать на случай ошибки, как минимум ретрай и сообщение в DLQ для ручного вмешательства
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// HandleUnreserveResult
func (s *PaymentService) HandleUnreserveResult(ctx context.Context, resp models.Event) error {
	const op = "service.payment.HandleUnreserveResult"

	logger := s.logger.With(
		zap.String("op:", op),
	)

	logger.Info("processing result of unreserve balance")

	var payloadResp models.BalanceResponsePayload
	if err := json.Unmarshal([]byte(resp.Payload), &payloadResp); err != nil {
		//TODO компенсирующие действия для unreserve fail
		return fmt.Errorf("failed to unmarshal balance response payload: %w", err)
	}

	if !payloadResp.Success {
		//TODO компенсирующие действия unreserve fail
	}

	payment, err := s.paymentManager.GetPayment(ctx, &models.GetPaymentRequest{ID: resp.PaymentID})
	if err != nil {
		//TODO компенсирующие действия
		return fmt.Errorf("%s: %w", op, err)
	}

	switch payment.SenderType {
	case models.AccountInternal:
		payload := models.BalanceRequestPayload{
			AccountCode: payment.SenderAccountCode,
			Amount:      payment.Amount,
			Currency:    payment.Currency,
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		event := &models.CreateEventRequest{
			Event: &models.Event{
				ID:        uuid.New(),
				PaymentID: payment.ID,
				Type:      models.EventUnfreezeRequest,
				CreatedAt: time.Now(),
				Source:    models.Source,
				Payload:   payloadBytes,
			},
		}

		if err := s.eventManager.CreateEvent(ctx, event); err != nil {
			// TODO здесь надо что-то придумать на случай ошибки, как минимум ретрай и сообщение в DLQ для ручного вмешательства
			return fmt.Errorf("%s: %w", op, err)
		}

		return nil
	case models.AccountExternal:
		req := &models.UpdatePaymentStatusRequest{
			ID:     resp.PaymentID,
			Status: models.PaymentStatusCancelled,
		}
		if err := s.paymentManager.UpdatePaymentStatus(ctx, req); err != nil {
			//TODO компенсирующие действия unreserve fail
			return fmt.Errorf("%s: %w", op, err)
		}
	}
	return nil
}

func (s *PaymentService) compensatingTryBalance(ctx context.Context, event models.Event) error {
	const op = "service.payment.compensatingTryBalance"

	logger := s.logger.With(
		zap.String("op:", op),
	)

	logger.Info("result of freeze balance are failed, canceling paymant")

	req := &models.UpdatePaymentStatusRequest{
		ID:     event.PaymentID,
		Status: models.PaymentStatusCancelled,
	}
	// TODO здесь надо что-то придумать на случай ошибки, как минимум ретрай и сообщение в DLQ для ручного вмешательства
	if err := s.paymentManager.UpdatePaymentStatus(ctx, req); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// HandleExternalResult
func (s *PaymentService) HandleExternalResult(ctx context.Context, resp models.Event) error {
	const op = "service.payment.HandleExternalResult"

	logger := s.logger.With(
		zap.String("op:", op),
	)

	logger.Info("processing result of external payment")

	var payloadResp models.BalanceResponsePayload
	if err := json.Unmarshal([]byte(resp.Payload), &payloadResp); err != nil {
		//TODO компенсирующие действия для Extarnal payment fail
		return fmt.Errorf("failed to unmarshal balance response payload: %w", err)
	}

	if !payloadResp.Success {
		//TODO компенсирующие действия для Extarnal payment fail
	}

	payment, err := s.paymentManager.GetPayment(ctx, &models.GetPaymentRequest{ID: resp.PaymentID})
	if err != nil {
		//TODO компенсирующие действия
		return fmt.Errorf("%s: %w", op, err)
	}

	payload := models.BalanceRequestPayload{
		Amount:   payment.Amount,
		Currency: payment.Currency,
	}

	var eventType string
	switch payment.SenderType {
	case models.AccountInternal:
		payload.AccountCode = payment.SenderAccountCode
		eventType = models.EventWithdrawRequest
	case models.AccountExternal:
		payload.AccountCode = payment.RecipientAccountCode
		eventType = models.EventDepositRequest
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	event := &models.CreateEventRequest{
		Event: &models.Event{
			ID:        uuid.New(),
			PaymentID: payment.ID,
			Type:      eventType,
			CreatedAt: time.Now(),
			Source:    models.Source,
			Payload:   payloadBytes,
		},
	}

	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		// TODO здесь надо что-то придумать на случай ошибки, как минимум ретрай и сообщение в DLQ для ручного вмешательства
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// HandleExternalRollbackResult
func (s *PaymentService) HandleExternalRollbackResult(ctx context.Context, resp models.Event) error {
	const op = "service.payment.HandleExternalRollbackResult"

	logger := s.logger.With(
		zap.String("op:", op),
	)

	logger.Info("processing result of external rollback")

	var payloadResp models.BalanceResponsePayload
	if err := json.Unmarshal([]byte(resp.Payload), &payloadResp); err != nil {
		//TODO компенсирующие действия для extarnal rollback fail
		return fmt.Errorf("failed to unmarshal balance response payload: %w", err)
	}

	if !payloadResp.Success {
		//TODO компенсирующие действия для extarnal rollback fail
	}

	payment, err := s.paymentManager.GetPayment(ctx, &models.GetPaymentRequest{ID: resp.PaymentID})
	if err != nil {
		//TODO компенсирующие действия
		return fmt.Errorf("%s: %w", op, err)
	}

	payload := models.BalanceRequestPayload{
		Amount:   payment.Amount,
		Currency: payment.Currency,
	}

	var eventType string
	switch payment.SenderType {
	case models.AccountInternal:
		payload.AccountCode = payment.SenderAccountCode
		eventType = models.EventUnfreezeRequest
	case models.AccountExternal:
		payload.AccountCode = payment.RecipientAccountCode
		eventType = models.EventUnreserveRequest
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	event := &models.CreateEventRequest{
		Event: &models.Event{
			ID:        uuid.New(),
			PaymentID: payment.ID,
			Type:      eventType,
			CreatedAt: time.Now(),
			Source:    models.Source,
			Payload:   payloadBytes,
		},
	}

	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		// TODO здесь надо что-то придумать на случай ошибки, как минимум ретрай и сообщение в DLQ для ручного вмешательства
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// HandleWithdrawResult
func (s *PaymentService) HandleWithdrawResult(ctx context.Context, resp models.Event) error {
	const op = "service.payment.HandleWithdrawResult"

	logger := s.logger.With(
		zap.String("op:", op),
	)

	logger.Info("processing result of withdraw balance")

	var payloadResp models.BalanceResponsePayload
	if err := json.Unmarshal([]byte(resp.Payload), &payloadResp); err != nil {
		//TODO компенсирующие действия для withdraw balance
		return fmt.Errorf("failed to unmarshal balance response payload: %w", err)
	}

	if !payloadResp.Success {
		//TODO компенсирующие действия для withdraw balance
	}

	payment, err := s.paymentManager.GetPayment(ctx, &models.GetPaymentRequest{ID: resp.PaymentID})
	if err != nil {
		//TODO компенсирующие действия
		return fmt.Errorf("%s: %w", op, err)
	}

	var eventType string
	var payload any

	switch payment.RecipientType {
	case models.AccountInternal:
		eventType = models.EventDepositRequest
		payload = models.BalanceRequestPayload{
			AccountCode: payment.RecipientAccountCode,
			Amount:      payment.Amount,
			Currency:    payment.Currency,
		}
	case models.AccountExternal:
		eventType = models.EventExternalCommit
		payload = models.ExternalRequestPayload{
			// TODO: ISOmessge
			ISOMessage: "Здесь будет ISO message",
		}
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	event := &models.CreateEventRequest{
		Event: &models.Event{
			ID:        uuid.New(),
			PaymentID: payment.ID,
			Type:      eventType,
			CreatedAt: time.Now(),
			Source:    models.Source,
			Payload:   payloadBytes,
		},
	}

	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		// TODO здесь надо что-то придумать на случай ошибки, как минимум ретрай и сообщение в DLQ для ручного вмешательства
		return fmt.Errorf("%s: %w", op, err)
	}

	if payment.RecipientType == models.AccountExternal {
		req := &models.UpdatePaymentStatusRequest{
			ID:     resp.PaymentID,
			Status: models.PaymentStatusCompleted,
		}
		if err := s.paymentManager.UpdatePaymentStatus(ctx, req); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		// TODO здесь надо что-то придумать на случай ошибки, как минимум ретрай и сообщение в DLQ для ручного вмешательства
		logger.Info("payment completed")
	}

	return nil
}

// HandleDepositResult
func (s *PaymentService) HandleDepositResult(ctx context.Context, resp models.Event) error {
	const op = "service.payment.HandleDepositResult"

	logger := s.logger.With(
		zap.String("op:", op),
	)

	logger.Info("processing result of deposit balance")

	var payloadResp models.BalanceResponsePayload
	if err := json.Unmarshal([]byte(resp.Payload), &payloadResp); err != nil {
		//TODO компенсирующие действия для deposit balance
		return fmt.Errorf("failed to unmarshal balance response payload: %w", err)
	}

	if !payloadResp.Success {
		//TODO компенсирующие действия для deposit balance
	}

	payment, err := s.paymentManager.GetPayment(ctx, &models.GetPaymentRequest{ID: resp.PaymentID})
	if err != nil {
		// TODO здесь надо что-то придумать на случай ошибки, как минимум ретрай и сообщение в DLQ для ручного вмешательства
		return fmt.Errorf("%s: %w", op, err)
	}

	if payment.SenderType == models.AccountExternal {
		eventType := models.EventExternalCommit
		payload := models.ExternalRequestPayload{
			// TODO: ISOmessge
			ISOMessage: "Здесь будет ISO message",
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		event := &models.CreateEventRequest{
			Event: &models.Event{
				ID:        uuid.New(),
				PaymentID: payment.ID,
				Type:      eventType,
				CreatedAt: time.Now(),
				Source:    models.Source,
				Payload:   payloadBytes,
			},
		}

		if err := s.eventManager.CreateEvent(ctx, event); err != nil {
			// TODO здесь надо что-то придумать на случай ошибки, как минимум ретрай и сообщение в DLQ для ручного вмешательства
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	req := &models.UpdatePaymentStatusRequest{
		ID:     resp.PaymentID,
		Status: models.PaymentStatusCompleted,
	}
	if err := s.paymentManager.UpdatePaymentStatus(ctx, req); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	// TODO здесь надо что-то придумать на случай ошибки, как минимум ретрай и сообщение в DLQ для ручного вмешательства
	logger.Info("payment completed")

	return nil
}
