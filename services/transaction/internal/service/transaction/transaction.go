// Пакет service предоставляет бизнес-логику для управления платежами
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TransactionService реализует бизнес-логику работы с платежами
type TransactionService struct {
	transactionManager TransactionManager
	eventManager       EventManager
	iso8583Manager     Iso8583Manager
	logger             *zap.Logger
}

// NewTransactionService создает новый экземпляр TransactionService
func NewTransactionService(
	transactionManager TransactionManager,
	eventManager EventManager,
	iso8583Manager Iso8583Manager,
	logger *zap.Logger,
) *TransactionService {
	return &TransactionService{
		transactionManager: transactionManager,
		eventManager:       eventManager,
		iso8583Manager:     iso8583Manager,
		logger:             logger.With(zap.String("component", "transaction_service")),
	}
}

// TransactionManager определяет методы управления платежами
type TransactionManager interface {
	CreateTransaction(ctx context.Context, req *models.CreateTransactionRequest, event *models.CreateEventRequest) error
	GetTransaction(ctx context.Context, req *models.GetTransactionRequest) (models.GetTransactionResponse, error)
	UpdateTransactionStatus(ctx context.Context, req *models.UpdateTransactionStatusRequest) error
}

// EventManager определяет методы управления платежами
type EventManager interface {
	CreateEvent(ctx context.Context, req *models.CreateEventRequest) error
}

// Iso8583Manager определяет методы управления транзакциями в формате iso8583
type Iso8583Manager interface {
	CreateTransactionFromISO(msg *models.ISO8583Message) (*models.CreateTransactionRequest, error)
	CreateFinancialRequest(transaction *models.Transaction) (*models.ISO8583Message, error)
	CreateFinancialResponse(transaction *models.Transaction, success bool, reason string) (*models.ISO8583Message, error)
}

// ==================== ОСНОВНЫЕ ОБРАБОТЧИКИ ====================

// HandleTransactionResponse - создание нового платежа
func (s *TransactionService) HandleTransactionResponse(ctx context.Context, resp *models.Event) error {
	const op = "service.transaction.HandleTransactionResponse"

	logger := s.logger.With(
		zap.String("op:", op),
	)

	logger.Info("transaction saga started")

	var transaction *models.CreateTransactionRequest
	var err error
	if transaction, err = s.getTransactionPayload(resp); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	transaction.Status = models.TransactionStatusPending

	var event *models.CreateEventRequest

	// если тип отправителя внутрненний - значит это перевод,
	// замораживаем средства на счете отправителя,
	// если тип отправителя внешний - значит это пополнение счета,
	// резервируем сердства для пополнения у получателя
	switch transaction.SenderType {
	case models.AccountInternal:
		if event, err = s.createFreezeEvent(transaction.Transaction); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	case models.AccountExternal:
		if event, err = s.createReserveEvent(transaction.Transaction); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	default:
		eventType := "unknow operation"
		return fmt.Errorf("%s: %s", op, eventType)
	}

	if err := s.transactionManager.CreateTransaction(ctx, transaction, event); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	logger.Info("transaction wait account",
		zap.String("event id:", event.ID.String()),
		zap.String("event type:", event.Type),
		zap.String("transaction id:", transaction.ID.String()),
		zap.String("account code:", transaction.SenderAccountCode),
	)

	return nil
}

// HandleFreezeResponse - ответ на заморозку средств при переводе
func (s *TransactionService) HandleFreezeResponse(ctx context.Context, resp *models.Event) error {
	const op = "service.transaction.HandleFreezeResponse"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("transaction ID:", resp.TransactionID.String()),
	)

	logger.Info("processing response of freeze balance")

	// если заморозка средств прошла не успешно(или невозможно
	// разшифровать payload), то компенсируем
	if !s.isSuccessBalanceResponse(resp) {
		if err := s.cancelTransaction(ctx, resp); err != nil {
			return fmt.Errorf("%s: compensating action failed: %w", op, err)
		}
		return nil
	}

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	var event *models.CreateEventRequest

	// если тип получателя внутрненний - значит это перевод между счетами сервиса,
	// резервируем средства на счете получателя,
	// если тип получателя внешний - значит это перевод в другую платёжку,
	// отправляем запрос для пополнения получателя на "внешний платёжный шлюз"
	switch transaction.RecipientType {
	case models.AccountInternal:
		if event, err = s.createReserveEvent(transaction.Transaction); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	case models.AccountExternal:
		if event, err = s.createExternalEvent(transaction.Transaction); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// HandleReserveResponse
func (s *TransactionService) HandleReserveResponse(ctx context.Context, resp *models.Event) error {
	const op = "service.transaction.HandleReserveResponse"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("transaction ID:", resp.TransactionID.String()),
	)

	logger.Info("processing response of reserve balance")

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	var event *models.CreateEventRequest

	// если резервирование средств прошло не успешно(или невозможно разшифровать payload),
	// то начинаем компенсацию распределенной транзакции
	if !s.isSuccessBalanceResponse(resp) {
		// если тип отправителя внутрненний - значит успели заморозить средства,
		// размораживаем средства отправителя,
		// если тип отправителя внешний - значит это пополнение,
		// никаких изменений в состоянии не произошло, просто отменяем платёж
		// и сообщаем во внешнюю систему об ошибке
		switch transaction.SenderType {
		case models.AccountInternal:
			if event, err = s.createUnfreezeEvent(transaction.Transaction); err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
			if err := s.eventManager.CreateEvent(ctx, event); err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
		case models.AccountExternal:
			if err := s.cancelTransaction(ctx, resp); err != nil {
				return fmt.Errorf("%s: compensating action failed: %w", op, err)
			}
			if event, err = s.createExternalRollbackEvent(transaction.Transaction, resp); err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
		}
		return nil
	}

	// если тип отправителя внутрненний - значит это перевод,
	// списываем средства со счета отправителя,
	// если тип отправителя внешний - значит это пополнение счета,
	// запрашиваем сердства для пополнения у отправителя в "внешнем платёжном шлюзе"
	switch transaction.SenderType {
	case models.AccountInternal:
		if event, err = s.createWithdrawEvent(transaction.Transaction); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	case models.AccountExternal:
		if event, err = s.createExternalEvent(transaction.Transaction); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// HandleExternalResponse
func (s *TransactionService) HandleExternalResponse(ctx context.Context, resp *models.Event) error {
	const op = "service.transaction.HandleExternalResponse"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("transaction ID:", resp.TransactionID.String()),
	)

	logger.Info("processing response of external transaction")

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	var event *models.CreateEventRequest

	// если внешнний платёжный шлюз прислал отказ(или невозможно разшифровать payload),
	// то начинаем компенсацию распределенной транзакции
	if !s.isSuccessExternalResponse(resp) {
		// если тип отправителя внутрненний - значит успели заморозить средства,
		// размораживаем средства отправителя,
		// если тип отправителя внешний - значит успели зарезервировать средства,
		// убираем резерв средств пополнения у получателя
		switch transaction.SenderType {
		case models.AccountInternal:
			if event, err = s.createUnfreezeEvent(transaction.Transaction); err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
		case models.AccountExternal:
			if event, err = s.createUnreserveEvent(transaction.Transaction); err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
		}
		if err := s.eventManager.CreateEvent(ctx, event); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		return nil
	}

	// если тип отправителя внутрненний - значит это перевод,
	// списываем средства со счета отправителя,
	// если тип отправителя внешний - значит это пополнение счета,
	// пополняем сердства на счете получателя
	switch transaction.SenderType {
	case models.AccountInternal:
		if event, err = s.createWithdrawEvent(transaction.Transaction); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	case models.AccountExternal:
		if event, err = s.createDepositeEvent(transaction.Transaction); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// HandleWithdrawResponse
func (s *TransactionService) HandleWithdrawResponse(ctx context.Context, resp *models.Event) error {
	const op = "service.transaction.HandleWithdrawResponse"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("transaction ID:", resp.TransactionID.String()),
	)

	logger.Info("processing response of withdraw balance")

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	var event *models.CreateEventRequest

	// если списание средств прошло не успешно(или невозможно разшифровать payload),
	// то начинаем компенсацию распределенной транзакции
	if !s.isSuccessExternalResponse(resp) {
		// если тип получателя внутрненний - значит успели и заморозить и зарезервировать средства,
		// сначала убираем резерв средств получателя,
		// если тип получателя внешний - значит отправили средства во "внешний платёжный шлюз",
		// просим откатить перевод
		switch transaction.SenderType {
		case models.AccountInternal:
			if event, err = s.createUnreserveEvent(transaction.Transaction); err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
		case models.AccountExternal:
			if event, err = s.createExternalRollbackEvent(transaction.Transaction, resp); err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
		}
		if err := s.eventManager.CreateEvent(ctx, event); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		return nil
	}

	// если тип получателя внутрненний - значит это перевод,
	// пополняем сердства на счете получателя,
	// если тип получателя внешний - значит это перевод в другую систему,
	// отправляем подтверждение успешной транзакции пополнения на "внешний платёжный шлюз"
	switch transaction.RecipientType {
	case models.AccountInternal:
		if event, err = s.createDepositeEvent(transaction.Transaction); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	case models.AccountExternal:
		if event, err = s.createExternalCommitEvent(transaction.Transaction, resp); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// если тип получателя внешний, то платёж полностью обработан, обновляем статус
	if transaction.RecipientType == models.AccountExternal {
		req := &models.UpdateTransactionStatusRequest{
			ID:     resp.TransactionID,
			Status: models.TransactionStatusCompleted,
		}
		if err := s.transactionManager.UpdateTransactionStatus(ctx, req); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		logger.Info("transaction completed")
	}

	return nil
}

// HandleDepositResponse
func (s *TransactionService) HandleDepositResponse(ctx context.Context, resp *models.Event) error {
	const op = "service.transaction.HandleDepositResponse"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("transaction ID:", resp.TransactionID.String()),
	)

	logger.Info("processing response of deposit balance")

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	var event *models.CreateEventRequest

	// если пополнение средств прошло не успешно(или невозможно разшифровать payload),
	// то начинаем компенсацию распределенной транзакции
	if !s.isSuccessExternalResponse(resp) {
		// если тип отправителя внутрненний - значит успели списать средства,
		// начинаем с возврата средств отправителю,
		// если тип отправителя внешний - значит получили средства от "внешнего платёжного шлюза",
		// просим откатить пополнение
		switch transaction.SenderType {
		case models.AccountInternal:
			if event, err = s.createRefandEvent(transaction.Transaction); err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
		case models.AccountExternal:
			if event, err = s.createExternalRollbackEvent(transaction.Transaction, resp); err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
		}
		if err := s.eventManager.CreateEvent(ctx, event); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		return nil
	}

	// если тип отправителя внешний - значит это пополнение счета,
	// отправляем подтверждение успешной транзакции пополнения на "внешний платёжный шлюз"
	if transaction.SenderType == models.AccountExternal {
		var event *models.CreateEventRequest

		if event, err = s.createExternalCommitEvent(transaction.Transaction, resp); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		if err := s.eventManager.CreateEvent(ctx, event); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	// платёж обработан, обновляем статус
	req := &models.UpdateTransactionStatusRequest{
		ID:     resp.TransactionID,
		Status: models.TransactionStatusCompleted,
	}
	if err := s.transactionManager.UpdateTransactionStatus(ctx, req); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	logger.Info("transaction completed")

	return nil
}

// ==================== КОМПЕНСАЦИОННЫЕ ДЕЙСТВИЯ ====================

// HandleUnfreezeResponse
func (s *TransactionService) HandleUnfreezeResponse(ctx context.Context, resp *models.Event) error {
	const op = "service.transaction.HandleUnfreezeResponse"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("transaction ID:", resp.TransactionID.String()),
	)

	logger.Info("processing response of unfreeze balance")

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	var payloadResp models.BalanceResponsePayload
	if err := json.Unmarshal([]byte(resp.Payload), &payloadResp); err != nil {
		return fmt.Errorf("failed to unmarshal balance response payload: %w", err)
	}

	// есди проваливаем компенсирующие действия, то блочим счета и смотрим вручную
	if !s.isSuccessBalanceResponse(resp) {
		req := &models.UpdateAccountRequest{
			TransactionID: transaction.ID,
			Code:          transaction.SenderAccountCode,
		}
		s.HandleBlockAccountRequest(ctx, req)
		if transaction.RecipientType == models.AccountInternal {
			req.Code = transaction.RecipientAccountCode
			s.HandleBlockAccountRequest(ctx, req)
		}
		return fmt.Errorf("%s: faild to unfreeze balance: %w", op, apperr.ErrEventUnsuccess)
	}

	req := &models.UpdateTransactionStatusRequest{
		ID:     resp.TransactionID,
		Status: models.TransactionStatusCancelled,
	}

	if err := s.transactionManager.UpdateTransactionStatus(ctx, req); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// HandleUnreserveResponse
func (s *TransactionService) HandleUnreserveResponse(ctx context.Context, resp *models.Event) error {
	const op = "service.transaction.HandleUnreserveResponse"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("transaction ID:", resp.TransactionID.String()),
	)

	logger.Info("processing response of unreserve balance")

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	var payloadResp models.BalanceResponsePayload
	if err := json.Unmarshal([]byte(resp.Payload), &payloadResp); err != nil {
		return fmt.Errorf("failed to unmarshal balance response payload: %w", err)
	}

	// есди проваливаем компенсирующие действия, то блочим счета и смотрим вручную
	if !s.isSuccessBalanceResponse(resp) {
		req := &models.UpdateAccountRequest{
			TransactionID: transaction.ID,
			Code:          transaction.RecipientAccountCode,
		}
		s.HandleBlockAccountRequest(ctx, req)
		if transaction.SenderType == models.AccountInternal {
			req.Code = transaction.SenderAccountCode
			s.HandleBlockAccountRequest(ctx, req)
		}
		return fmt.Errorf("%s: faild to unfreeze balance: %w", op, apperr.ErrEventUnsuccess)
	}

	var event *models.CreateEventRequest

	// если отправитель - внутренний, значит это перевод внутри системы
	// и нужно еще разморозить средства отправителя
	// если отправитель - внешний, значит пополнение компенсировано
	// отменяем платеж
	switch transaction.SenderType {
	case models.AccountInternal:
		if event, err = s.createUnfreezeEvent(transaction.Transaction); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		if err := s.eventManager.CreateEvent(ctx, event); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		return nil
	case models.AccountExternal:
		if err := s.cancelTransaction(ctx, resp); err != nil {
			return fmt.Errorf("%s: compensating action failed: %w", op, err)
		}
	}
	return nil
}

// HandleExternalRollbackResponse
func (s *TransactionService) HandleExternalRollbackResponse(ctx context.Context, resp *models.Event) error {
	const op = "service.transaction.HandleExternalRollbackResponse"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("transaction ID:", resp.TransactionID.String()),
	)

	logger.Info("processing response of external rollback")

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// есди проваливаем компенсирующие действия, то блочим счет и смотрим вручную
	if !s.isSuccessExternalResponse(resp) {
		req := &models.UpdateAccountRequest{
			TransactionID: transaction.ID,
			Code:          transaction.RecipientAccountCode,
		}
		s.HandleBlockAccountRequest(ctx, req)
		return fmt.Errorf("%s: faild to unfreeze balance: %w", op, apperr.ErrEventUnsuccess)
	}

	var event *models.CreateEventRequest

	// если отправитель - внутренний, значит это внешний перевод
	// размораживаем средства отправителя
	// если отправитель - внешний, пополнение
	// отменяем резерв для пополняния у получателя
	switch transaction.SenderType {
	case models.AccountInternal:
		if event, err = s.createUnfreezeEvent(transaction.Transaction); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	case models.AccountExternal:
		if event, err = s.createUnreserveEvent(transaction.Transaction); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// HandleRefundResponse
func (s *TransactionService) HandleRefundResponse(ctx context.Context, resp *models.Event) error {
	const op = "service.transaction.HandleRefundResponse"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("transaction ID:", resp.TransactionID.String()),
	)

	logger.Info("processing response of refund balance")

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// есди проваливаем компенсирующие действия, то блочим счета и смотрим вручную
	if !s.isSuccessBalanceResponse(resp) {
		req := &models.UpdateAccountRequest{
			TransactionID: transaction.ID,
			Code:          transaction.SenderAccountCode,
		}
		s.HandleBlockAccountRequest(ctx, req)
		if transaction.RecipientType == models.AccountInternal {
			req.Code = transaction.RecipientAccountCode
			s.HandleBlockAccountRequest(ctx, req)
		}
		return fmt.Errorf("%s: faild to unfreeze balance: %w", op, apperr.ErrEventUnsuccess)
	}

	var event *models.CreateEventRequest

	// если получатель - вннутренний, то
	// отменяем резерв его средств
	// если внешний, то послыаем во внешюю платёжку
	// что хотим деньги назад
	switch transaction.RecipientType {
	case models.AccountInternal:
		if event, err = s.createUnreserveEvent(transaction.Transaction); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	case models.AccountExternal:
		if event, err = s.createExternalRollbackEvent(transaction.Transaction, resp); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}
	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// HandleBlockAccountRequest
func (s *TransactionService) HandleBlockAccountRequest(ctx context.Context, req *models.UpdateAccountRequest) error {
	const op = "service.transaction.HandleBlockAccountRequest"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("transaction ID:", req.TransactionID.String()),
	)

	logger.Info("blocking account started")

	payload := models.UpdateAccountPayload{
		Code: req.Code,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	event := &models.CreateEventRequest{
		Event: &models.Event{
			ID:            uuid.New(),
			TransactionID: req.TransactionID,
			Type:          models.EventBlockRequest,
			Status:        models.EventStatusPending,
			CreatedAt:     time.Now(),
			Source:        models.Source,
			Payload:       payloadBytes,
		},
	}

	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// HandleBlockAccountResponse
func (s *TransactionService) HandleBlockAccountResponse(ctx context.Context, resp *models.Event) error {
	const op = "service.transaction.HandleBlockAccountResponse"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("transaction ID:", resp.TransactionID.String()),
	)

	logger.Info("processing response of blocking account")

	if !s.isSuccessBalanceResponse(resp) {
		return fmt.Errorf("%s: faild to block account: %w", op, apperr.ErrEventUnsuccess)
	}

	req := &models.UpdateTransactionStatusRequest{
		ID:     resp.TransactionID,
		Status: models.TransactionStatusBlocked,
	}

	if err := s.transactionManager.UpdateTransactionStatus(ctx, req); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// ==================== СОЗДАНИЕ СОБЫТИЙ ====================

// createEvent обертка для создания типа событий
func (s *TransactionService) createEvent(transactionID uuid.UUID, accountcode, eventType string, payload any) (*models.CreateEventRequest, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &models.CreateEventRequest{
		Event: &models.Event{
			ID:            uuid.New(),
			TransactionID: transactionID,
			PartitionKey:  s.partitionKey(accountcode),
			Type:          eventType,
			Status:        models.EventStatusPending,
			Source:        models.Source,
			CreatedAt:     time.Now(),
			Payload:       payloadBytes,
		},
	}, nil
}

// createFreezeEvent - создает событие заморозки средств отправителя
func (s *TransactionService) createFreezeEvent(req *models.Transaction) (*models.CreateEventRequest, error) {
	payload := models.BalanceRequestPayload{
		AccountCode: req.SenderAccountCode,
		Amount:      req.Amount,
		Currency:    req.Currency,
	}

	return s.createEvent(req.ID, req.SenderAccountCode, models.EventFreezeRequest, payload)
}

// createReserveEvent - создает событие резерва средств получателя
func (s *TransactionService) createReserveEvent(req *models.Transaction) (*models.CreateEventRequest, error) {
	payload := models.BalanceRequestPayload{
		AccountCode: req.RecipientAccountCode,
		Amount:      req.Amount,
		Currency:    req.Currency,
	}

	return s.createEvent(req.ID, req.RecipientAccountCode, models.EventReserveRequest, payload)
}

// createExternalEvent - создает событие для обращения к "внешнему платёжному шлюзу"
func (s *TransactionService) createExternalEvent(req *models.Transaction) (*models.CreateEventRequest, error) {
	isoPayload, err := s.iso8583Manager.CreateFinancialRequest(req)
	if err != nil {
		return nil, err
	}
	payload := models.ExternalRequestPayload{
		ISOMessage: isoPayload,
	}

	return s.createEvent(req.ID, req.RecipientAccountCode, models.EventExternalRequest, payload)
}

// createWithdrawEvent - создает событие списания средств отправителя
func (s *TransactionService) createWithdrawEvent(req *models.Transaction) (*models.CreateEventRequest, error) {
	payload := models.BalanceRequestPayload{
		AccountCode: req.SenderAccountCode,
		Amount:      req.Amount,
		Currency:    req.Currency,
	}

	return s.createEvent(req.ID, req.SenderAccountCode, models.EventWithdrawRequest, payload)
}

// createDepositeEvent - создает событие пополнение средств получателя
func (s *TransactionService) createDepositeEvent(req *models.Transaction) (*models.CreateEventRequest, error) {
	payload := models.BalanceRequestPayload{
		AccountCode: req.RecipientAccountCode,
		Amount:      req.Amount,
		Currency:    req.Currency,
	}

	return s.createEvent(req.ID, req.RecipientAccountCode, models.EventDepositRequest, payload)
}

// createExternalCommitEvent - создает событие для обращения к "внешнему платёжному шлюзу"
// с подтвержденем успешной транзакции
func (s *TransactionService) createExternalCommitEvent(
	req *models.Transaction,
	resp *models.Event,
) (*models.CreateEventRequest, error) {
	var payloadResp *models.BalanceResponsePayload
	if err := json.Unmarshal(resp.Payload, &payloadResp); err != nil {
		return nil, err
	}

	isoPayload, err := s.iso8583Manager.CreateFinancialResponse(req, payloadResp.Success, payloadResp.Reason)
	if err != nil {
		return nil, err
	}

	payload := models.ISO8583Payload{
		ISOMessage:   isoPayload,
		Success:      payloadResp.Success,
		ErrorCode:    isoPayload.ResponseCode,
		ErrorMessage: payloadResp.Reason,
	}

	return s.createEvent(req.ID, req.RecipientAccountCode, models.EventExternalCommit, payload)
}

// createUnfreezeEvent - создает событие разморозки средств отправителя
func (s *TransactionService) createUnfreezeEvent(req *models.Transaction) (*models.CreateEventRequest, error) {
	payload := models.BalanceRequestPayload{
		AccountCode: req.SenderAccountCode,
		Amount:      req.Amount,
		Currency:    req.Currency,
	}

	return s.createEvent(req.ID, req.SenderAccountCode, models.EventUnfreezeRequest, payload)
}

// createUnreserveEvent - создает событие для отката создания резерва средств получателя
func (s *TransactionService) createUnreserveEvent(req *models.Transaction) (*models.CreateEventRequest, error) {
	payload := models.BalanceRequestPayload{
		AccountCode: req.RecipientAccountCode,
		Amount:      req.Amount,
		Currency:    req.Currency,
	}

	return s.createEvent(req.ID, req.RecipientAccountCode, models.EventUnreserveRequest, payload)
}

// createExternalRollbackEvent - создает событие для обращения к "внешнему платёжному шлюзу"
func (s *TransactionService) createExternalRollbackEvent(
	req *models.Transaction,
	resp *models.Event,
) (*models.CreateEventRequest, error) {
	var payloadResp *models.BalanceResponsePayload
	if err := json.Unmarshal(resp.Payload, &payloadResp); err != nil {
		return nil, err
	}

	isoPayload, err := s.iso8583Manager.CreateFinancialResponse(req, payloadResp.Success, payloadResp.Reason)
	if err != nil {
		return nil, err
	}

	payload := models.ISO8583Payload{
		ISOMessage:   isoPayload,
		Success:      payloadResp.Success,
		ErrorCode:    isoPayload.ResponseCode,
		ErrorMessage: payloadResp.Reason,
	}
	return s.createEvent(req.ID, req.RecipientAccountCode, models.EventExternalRollback, payload)
}

// createRefandEvent - создает событие возврата средств отправителя
func (s *TransactionService) createRefandEvent(req *models.Transaction) (*models.CreateEventRequest, error) {
	payload := models.BalanceRequestPayload{
		AccountCode: req.SenderAccountCode,
		Amount:      req.Amount,
		Currency:    req.Currency,
	}

	return s.createEvent(req.ID, req.SenderAccountCode, models.EventRefundRequest, payload)
}

// ==================== ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ====================

// getTransactionPayload - возвращает payload из события создания платежа
func (s *TransactionService) getTransactionPayload(resp *models.Event) (*models.CreateTransactionRequest, error) {
	const op = "service.transaction.getTransactionPayload"

	var payloadIsoResp *models.ISO8583Payload
	if err := json.Unmarshal(resp.Payload, &payloadIsoResp); err == nil {
		return s.iso8583Manager.CreateTransactionFromISO(payloadIsoResp.ISOMessage)
	}

	var payloadResp *models.CreateTransactionRequest
	if err := json.Unmarshal(resp.Payload, &payloadResp); err == nil {
		return payloadResp, nil
	}

	return nil, fmt.Errorf("%s: %s", op, "Unknow Transaction Type")
}

// isSuccessBalanceResponse - проверяет успешность ответа операций с балансом счета
func (s *TransactionService) isSuccessBalanceResponse(resp *models.Event) bool {
	var payloadResp *models.BalanceResponsePayload
	if err := json.Unmarshal(resp.Payload, &payloadResp); err != nil {
		return false
	}
	return payloadResp.Success
}

// isSuccessExternalResponse - проверяет успешность ответа внещнего платёжного шлюза
func (s *TransactionService) isSuccessExternalResponse(resp *models.Event) bool {
	var payloadResp *models.BalanceResponsePayload
	if err := json.Unmarshal(resp.Payload, &payloadResp); err != nil {
		return false
	}
	return payloadResp.Success
}

// cancelTransaction - обновляем статус платежа на отмененный
func (s *TransactionService) cancelTransaction(ctx context.Context, event *models.Event) error {
	const op = "service.transaction.cancelTransaction"

	logger := s.logger.With(
		zap.String("op:", op),
		zap.String("transaction ID:", event.TransactionID.String()),
	)

	logger.Info("canceling paymant")
	//TODO отсылать сообщение в INTERNAL, если нельзя
	req := &models.UpdateTransactionStatusRequest{
		ID:     event.TransactionID,
		Status: models.TransactionStatusCancelled,
	}

	if err := s.transactionManager.UpdateTransactionStatus(ctx, req); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// хэш код для ключа партиции из номера счета
func (s *TransactionService) partitionKey(accountCode string) string {
	hash := sha256.Sum256([]byte(accountCode))
	return "acc:" + hex.EncodeToString(hash[:8])
}
