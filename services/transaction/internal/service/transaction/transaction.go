// Пакет service предоставляет бизнес-логику для управления платежами
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"

	"github.com/google/uuid"
)

// TransactionService реализует бизнес-логику работы с платежами
type TransactionService struct {
	transactionManager TransactionManager
	eventManager       EventManager
	iso8583Manager     Iso8583Manager
	stanManager        StanManager
}

// NewTransactionService создает новый экземпляр TransactionService
func NewTransactionService(
	transactionManager TransactionManager,
	eventManager EventManager,
	iso8583Manager Iso8583Manager,
	stanManager StanManager,
) *TransactionService {
	return &TransactionService{
		transactionManager: transactionManager,
		eventManager:       eventManager,
		iso8583Manager:     iso8583Manager,
		stanManager:        stanManager,
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

// StanManager определяет методы управления System Trace Audit Number
type StanManager interface {
	GetNextSTAN(transactionType, senderType, recipientType string, date time.Time) (string, error)
}

// ==================== ОСНОВНЫЕ ОБРАБОТЧИКИ ====================

// HandleTransactionResponse - создание нового платежа
func (s *TransactionService) HandleTransactionResponse(ctx context.Context, resp *models.Event) error {
	const op = "transaction.HandleTransactionResponse"

	var transaction *models.CreateTransactionRequest
	var err error
	if transaction, err = s.getTransactionPayload(resp); err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	transaction.Status = models.TransactionStatusPending
	if transaction.Stan == "" {
		stan, err := s.stanManager.GetNextSTAN(
			transaction.Type,
			transaction.SenderType,
			transaction.RecipientType,
			time.Now(),
		)
		if err != nil {
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
		transaction.Stan = stan
	}

	var event *models.CreateEventRequest

	// если тип отправителя внутрненний - значит это перевод,
	// замораживаем средства на счете отправителя,
	// если тип отправителя внешний - значит это пополнение счета,
	// резервируем сердства для пополнения у получателя
	switch transaction.SenderType {
	case models.AccountInternal:
		if event, err = s.createFreezeEvent(transaction.Transaction); err != nil {
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	case models.AccountExternal:
		if event, err = s.createReserveEvent(transaction.Transaction); err != nil {
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	default:
		return &apperr.WrappedError{
			Op:  op,
			Err: apperr.ErrUnknownTransactionType,
		}
	}

	if err := s.transactionManager.CreateTransaction(ctx, transaction, event); err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return nil
}

// HandleFreezeResponse - ответ на заморозку средств при переводе
func (s *TransactionService) HandleFreezeResponse(ctx context.Context, resp *models.Event) error {
	const op = "transaction.HandleFreezeResponse"

	// если заморозка средств прошла не успешно(или невозможно
	// разшифровать payload), то компенсируем
	if !s.isSuccessResponse(resp) {
		if err := s.cancelTransaction(ctx, resp); err != nil {
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
		return nil
	}

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	var event *models.CreateEventRequest

	// если тип получателя внутрненний - значит это перевод между счетами сервиса,
	// резервируем средства на счете получателя,
	// если тип получателя внешний - значит это перевод в другую платёжку,
	// отправляем запрос для пополнения получателя на "внешний платёжный шлюз"
	switch transaction.RecipientType {
	case models.AccountInternal:
		if event, err = s.createReserveEvent(transaction.Transaction); err != nil {
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	case models.AccountExternal:
		if event, err = s.createExternalEvent(transaction.Transaction); err != nil {
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	}

	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return nil
}

// HandleReserveResponse
func (s *TransactionService) HandleReserveResponse(ctx context.Context, resp *models.Event) error {
	const op = "transaction.HandleReserveResponse"

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	var event *models.CreateEventRequest

	// если резервирование средств прошло не успешно(или невозможно разшифровать payload),
	// то начинаем компенсацию распределенной транзакции
	if !s.isSuccessResponse(resp) {
		// если тип отправителя внутрненний - значит успели заморозить средства,
		// размораживаем средства отправителя,
		// если тип отправителя внешний - значит это пополнение,
		// никаких изменений в состоянии не произошло, просто отменяем платёж
		// и сообщаем во внешнюю систему об ошибке
		switch transaction.SenderType {
		case models.AccountInternal:
			if event, err = s.createUnfreezeEvent(transaction.Transaction); err != nil {
				return &apperr.WrappedError{
					Op:  op,
					Err: err,
				}
			}
			if err := s.eventManager.CreateEvent(ctx, event); err != nil {
				var appErr *apperr.Error
				if errors.As(err, &appErr) {
					return err
				}
				return &apperr.WrappedError{
					Op:  op,
					Err: err,
				}
			}
		case models.AccountExternal:
			if err := s.cancelTransaction(ctx, resp); err != nil {
				return &apperr.WrappedError{
					Op:  op,
					Err: err,
				}
			}
			if event, err = s.createExternalRollbackEvent(transaction.Transaction, resp); err != nil {
				return &apperr.WrappedError{
					Op:  op,
					Err: err,
				}
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
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	case models.AccountExternal:
		if event, err = s.createExternalEvent(transaction.Transaction); err != nil {
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	}

	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return nil
}

// HandleExternalResponse
func (s *TransactionService) HandleExternalResponse(ctx context.Context, resp *models.Event) error {
	const op = "transaction.HandleExternalResponse"

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	var event *models.CreateEventRequest

	// если внешнний платёжный шлюз прислал отказ(или невозможно разшифровать payload),
	// то начинаем компенсацию распределенной транзакции
	if !s.isSuccessResponse(resp) {
		// если тип отправителя внутрненний - значит успели заморозить средства,
		// размораживаем средства отправителя,
		// если тип отправителя внешний - значит успели зарезервировать средства,
		// убираем резерв средств пополнения у получателя
		switch transaction.SenderType {
		case models.AccountInternal:
			if event, err = s.createUnfreezeEvent(transaction.Transaction); err != nil {
				return &apperr.WrappedError{
					Op:  op,
					Err: err,
				}
			}
		case models.AccountExternal:
			if event, err = s.createUnreserveEvent(transaction.Transaction); err != nil {
				return &apperr.WrappedError{
					Op:  op,
					Err: err,
				}
			}
		}
		if err := s.eventManager.CreateEvent(ctx, event); err != nil {
			var appErr *apperr.Error
			if errors.As(err, &appErr) {
				return err
			}
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
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
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	case models.AccountExternal:
		if event, err = s.createDepositeEvent(transaction.Transaction); err != nil {
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	}

	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return nil
}

// HandleWithdrawResponse
func (s *TransactionService) HandleWithdrawResponse(ctx context.Context, resp *models.Event) error {
	const op = "transaction.HandleWithdrawResponse"

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	var event *models.CreateEventRequest

	// если списание средств прошло не успешно(или невозможно разшифровать payload),
	// то начинаем компенсацию распределенной транзакции
	if !s.isSuccessResponse(resp) {
		// если тип получателя внутрненний - значит успели и заморозить и зарезервировать средства,
		// сначала убираем резерв средств получателя,
		// если тип получателя внешний - значит отправили средства во "внешний платёжный шлюз",
		// просим откатить перевод
		switch transaction.SenderType {
		case models.AccountInternal:
			if event, err = s.createUnreserveEvent(transaction.Transaction); err != nil {
				return &apperr.WrappedError{
					Op:  op,
					Err: err,
				}
			}
		case models.AccountExternal:
			if event, err = s.createExternalRollbackEvent(transaction.Transaction, resp); err != nil {
				return &apperr.WrappedError{
					Op:  op,
					Err: err,
				}
			}
		}
		if err := s.eventManager.CreateEvent(ctx, event); err != nil {
			var appErr *apperr.Error
			if errors.As(err, &appErr) {
				return err
			}
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
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
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	case models.AccountExternal:
		if event, err = s.createExternalCommitEvent(transaction.Transaction, resp); err != nil {
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	}

	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	// если тип получателя внешний, то платёж полностью обработан, обновляем статус
	if transaction.RecipientType == models.AccountExternal {
		req := &models.UpdateTransactionStatusRequest{
			ID:     resp.TransactionID,
			Status: models.TransactionStatusCompleted,
		}
		if err := s.transactionManager.UpdateTransactionStatus(ctx, req); err != nil {
			var appErr *apperr.Error
			if errors.As(err, &appErr) {
				return err
			}
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	}

	return nil
}

// HandleDepositResponse
func (s *TransactionService) HandleDepositResponse(ctx context.Context, resp *models.Event) error {
	const op = "transaction.HandleDepositResponse"

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	var event *models.CreateEventRequest

	// если пополнение средств прошло не успешно(или невозможно разшифровать payload),
	// то начинаем компенсацию распределенной транзакции
	if !s.isSuccessResponse(resp) {
		// если тип отправителя внутрненний - значит успели списать средства,
		// начинаем с возврата средств отправителю,
		// если тип отправителя внешний - значит получили средства от "внешнего платёжного шлюза",
		// просим откатить пополнение
		switch transaction.SenderType {
		case models.AccountInternal:
			if event, err = s.createRefandEvent(transaction.Transaction); err != nil {
				return &apperr.WrappedError{
					Op:  op,
					Err: err,
				}
			}
		case models.AccountExternal:
			if event, err = s.createExternalRollbackEvent(transaction.Transaction, resp); err != nil {
				return &apperr.WrappedError{
					Op:  op,
					Err: err,
				}
			}
		}
		if err := s.eventManager.CreateEvent(ctx, event); err != nil {
			var appErr *apperr.Error
			if errors.As(err, &appErr) {
				return err
			}
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
		return nil
	}

	// если тип отправителя внешний - значит это пополнение счета,
	// отправляем подтверждение успешной транзакции пополнения на "внешний платёжный шлюз"
	if transaction.SenderType == models.AccountExternal {
		if event, err = s.createExternalCommitEvent(transaction.Transaction, resp); err != nil {
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}

		if err := s.eventManager.CreateEvent(ctx, event); err != nil {
			var appErr *apperr.Error
			if errors.As(err, &appErr) {
				return err
			}
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	}

	// платёж обработан, обновляем статус
	req := &models.UpdateTransactionStatusRequest{
		ID:     resp.TransactionID,
		Status: models.TransactionStatusCompleted,
	}
	if err := s.transactionManager.UpdateTransactionStatus(ctx, req); err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return nil
}

// ==================== КОМПЕНСАЦИОННЫЕ ДЕЙСТВИЯ ====================

// HandleUnfreezeResponse
func (s *TransactionService) HandleUnfreezeResponse(ctx context.Context, resp *models.Event) error {
	const op = "transaction.HandleUnfreezeResponse"

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	// есди проваливаем компенсирующие действия, то блочим счета и смотрим вручную
	if !s.isSuccessResponse(resp) {
		req := &models.UpdateAccountRequest{
			TransactionID: transaction.ID,
			Code:          transaction.SenderAccountCode,
		}
		if err := s.HandleBlockAccountRequest(ctx, req); err != nil {
			return &apperr.TerminalError{
				Op: op,
				Context: fmt.Sprintf("failed to block sender account %s after unsuccessful unfreeze for transaction %s",
					transaction.SenderAccountCode, transaction.ID),
				Err: err,
			}
		}

		if transaction.RecipientType == models.AccountInternal {
			req.Code = transaction.RecipientAccountCode
			if err := s.HandleBlockAccountRequest(ctx, req); err != nil {
				return &apperr.TerminalError{
					Op: op,
					Context: fmt.Sprintf("failed to block recipient account %s after unsuccessful unfreeze for transaction %s",
						transaction.RecipientAccountCode, transaction.ID),
					Err: err,
				}
			}
		}
		return apperr.ErrEventUnsuccess
	}

	req := &models.UpdateTransactionStatusRequest{
		ID:     resp.TransactionID,
		Status: models.TransactionStatusCancelled,
	}

	if err := s.transactionManager.UpdateTransactionStatus(ctx, req); err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return nil
}

// HandleUnreserveResponse
func (s *TransactionService) HandleUnreserveResponse(ctx context.Context, resp *models.Event) error {
	const op = "transaction.HandleUnreserveResponse"

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	// есди проваливаем компенсирующие действия, то блочим счета и смотрим вручную
	if !s.isSuccessResponse(resp) {
		req := &models.UpdateAccountRequest{
			TransactionID: transaction.ID,
			Code:          transaction.RecipientAccountCode,
		}
		if err := s.HandleBlockAccountRequest(ctx, req); err != nil {
			return &apperr.TerminalError{
				Op: op,
				Context: fmt.Sprintf("failed to block recipient account %s after unsuccessful unreserve for transaction %s",
					transaction.RecipientAccountCode, transaction.ID),
				Err: err,
			}
		}

		if transaction.SenderType == models.AccountInternal {
			req.Code = transaction.SenderAccountCode
			if err := s.HandleBlockAccountRequest(ctx, req); err != nil {
				return &apperr.TerminalError{
					Op: op,
					Context: fmt.Sprintf("failed to block sender account %s after unsuccessful unreserve for transaction %s",
						transaction.SenderAccountCode, transaction.ID),
					Err: err,
				}
			}
		}
		return apperr.ErrEventUnsuccess
	}

	var event *models.CreateEventRequest

	// если отправитель - внутренний, значит это перевод внутри системы
	// и нужно еще разморозить средства отправителя
	// если отправитель - внешний, значит пополнение компенсировано
	// отменяем платеж
	switch transaction.SenderType {
	case models.AccountInternal:
		if event, err = s.createUnfreezeEvent(transaction.Transaction); err != nil {
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
		if err := s.eventManager.CreateEvent(ctx, event); err != nil {
			var appErr *apperr.Error
			if errors.As(err, &appErr) {
				return err
			}
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
		return nil
	case models.AccountExternal:
		if err := s.cancelTransaction(ctx, resp); err != nil {
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	}
	return nil
}

// HandleExternalRollbackResponse
func (s *TransactionService) HandleExternalRollbackResponse(ctx context.Context, resp *models.Event) error {
	const op = "transaction.HandleExternalRollbackResponse"

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	// есди проваливаем компенсирующие действия, то блочим счет и смотрим вручную
	if !s.isSuccessResponse(resp) {
		req := &models.UpdateAccountRequest{
			TransactionID: transaction.ID,
			Code:          transaction.RecipientAccountCode,
		}
		if err := s.HandleBlockAccountRequest(ctx, req); err != nil {
			return &apperr.TerminalError{
				Op: op,
				Context: fmt.Sprintf("failed to block recipient account %s after unsuccessful external rollback for transaction %s",
					transaction.RecipientAccountCode, transaction.ID),
				Err: err,
			}
		}
		return apperr.ErrEventUnsuccess
	}

	var event *models.CreateEventRequest

	// если отправитель - внутренний, значит это внешний перевод
	// размораживаем средства отправителя
	// если отправитель - внешний, пополнение
	// отменяем резерв для пополняния у получателя
	switch transaction.SenderType {
	case models.AccountInternal:
		if event, err = s.createUnfreezeEvent(transaction.Transaction); err != nil {
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	case models.AccountExternal:
		if event, err = s.createUnreserveEvent(transaction.Transaction); err != nil {
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	}

	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return nil
}

// HandleRefundResponse
func (s *TransactionService) HandleRefundResponse(ctx context.Context, resp *models.Event) error {
	const op = "transaction.HandleRefundResponse"

	transaction, err := s.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: resp.TransactionID})
	if err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	// есди проваливаем компенсирующие действия, то блочим счета и смотрим вручную
	if !s.isSuccessResponse(resp) {
		req := &models.UpdateAccountRequest{
			TransactionID: transaction.ID,
			Code:          transaction.SenderAccountCode,
		}
		if err := s.HandleBlockAccountRequest(ctx, req); err != nil {
			return &apperr.TerminalError{
				Op: op,
				Context: fmt.Sprintf("failed to block sender account %s after unsuccessful refund for transaction %s",
					transaction.SenderAccountCode, transaction.ID),
				Err: err,
			}
		}

		if transaction.RecipientType == models.AccountInternal {
			req.Code = transaction.RecipientAccountCode
			if err := s.HandleBlockAccountRequest(ctx, req); err != nil {
				return &apperr.TerminalError{
					Op: op,
					Context: fmt.Sprintf("failed to block recipient account %s after unsuccessful refund for transaction %s",
						transaction.RecipientAccountCode, transaction.ID),
					Err: err,
				}
			}
		}
		return apperr.ErrEventUnsuccess
	}

	var event *models.CreateEventRequest

	// если получатель - вннутренний, то
	// отменяем резерв его средств
	// если внешний, то послыаем во внешюю платёжку
	// что хотим деньги назад
	switch transaction.RecipientType {
	case models.AccountInternal:
		if event, err = s.createUnreserveEvent(transaction.Transaction); err != nil {
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	case models.AccountExternal:
		if event, err = s.createExternalRollbackEvent(transaction.Transaction, resp); err != nil {
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	}
	if err := s.eventManager.CreateEvent(ctx, event); err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return nil
}

// HandleBlockAccountRequest
func (s *TransactionService) HandleBlockAccountRequest(ctx context.Context, req *models.UpdateAccountRequest) error {
	const op = "transaction.HandleBlockAccountRequest"

	payload := models.UpdateAccountPayload{
		Code: req.Code,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
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
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return nil
}

// HandleBlockAccountResponse
func (s *TransactionService) HandleBlockAccountResponse(ctx context.Context, resp *models.Event) error {
	const op = "transaction.HandleBlockAccountResponse"

	if !s.isSuccessResponse(resp) {
		return apperr.ErrEventUnsuccess
	}

	req := &models.UpdateTransactionStatusRequest{
		ID:     resp.TransactionID,
		Status: models.TransactionStatusBlocked,
	}

	if err := s.transactionManager.UpdateTransactionStatus(ctx, req); err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return nil
}

// ==================== СОЗДАНИЕ СОБЫТИЙ ====================

// createEvent обертка для создания типа событий
func (s *TransactionService) createEvent(transactionID uuid.UUID, accountcode, eventType string, payload any) (*models.CreateEventRequest, error) {
	const op = "transaction.createEvent"

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, &apperr.WrappedError{
			Op:  op,
			Err: fmt.Errorf("marshal payload: %w", err),
		}
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
	const op = "transaction.createExternalEvent"

	isoPayload, err := s.iso8583Manager.CreateFinancialRequest(req)
	if err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return nil, err
		}
		return nil, &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
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
	const op = "transaction.createExternalCommitEvent"

	var payloadResp models.BalanceResponsePayload
	if err := json.Unmarshal(resp.Payload, &payloadResp); err != nil {
		return nil, &apperr.WrappedError{
			Op:  op,
			Err: fmt.Errorf("unmarshal payload: %w", err),
		}
	}

	isoPayload, err := s.iso8583Manager.CreateFinancialResponse(req, payloadResp.Success, payloadResp.Reason)
	if err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return nil, err
		}
		return nil, &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
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
	const op = "transaction.createExternalRollbackEvent"

	var payloadResp models.BalanceResponsePayload
	if err := json.Unmarshal(resp.Payload, &payloadResp); err != nil {
		return nil, &apperr.WrappedError{
			Op:  op,
			Err: fmt.Errorf("unmarshal payload: %w", err),
		}
	}

	isoPayload, err := s.iso8583Manager.CreateFinancialResponse(req, payloadResp.Success, payloadResp.Reason)
	if err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return nil, err
		}
		return nil, &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
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
	const op = "transaction.getTransactionPayload"

	var payloadResp models.CreateTransactionRequest
	if err := json.Unmarshal(resp.Payload, &payloadResp); err == nil && payloadResp.Transaction != nil {
		return &payloadResp, nil
	}

	var payloadIsoResp models.ISO8583Payload
	if err := json.Unmarshal(resp.Payload, &payloadIsoResp); err == nil && payloadIsoResp.ISOMessage != nil {
		return s.iso8583Manager.CreateTransactionFromISO(payloadIsoResp.ISOMessage)
	}

	return nil, &apperr.WrappedError{
		Op:  op,
		Err: apperr.ErrUnknownTransactionType,
	}
}

// isSuccessResponse - проверяет успешность ответа операций
func (s *TransactionService) isSuccessResponse(resp *models.Event) bool {
	var payloadResp models.BalanceResponsePayload
	if err := json.Unmarshal(resp.Payload, &payloadResp); err != nil {
		// TODO: add metrics
		return false
	}
	return payloadResp.Success
}

// cancelTransaction - обновляем статус платежа на отмененный
func (s *TransactionService) cancelTransaction(ctx context.Context, event *models.Event) error {
	const op = "transaction.cancelTransaction"

	//TODO отсылать сообщение в INTERNAL, если нельзя
	req := &models.UpdateTransactionStatusRequest{
		ID:     event.TransactionID,
		Status: models.TransactionStatusCancelled,
	}

	if err := s.transactionManager.UpdateTransactionStatus(ctx, req); err != nil {
		if err, ok := errors.AsType[*apperr.Error](err); ok {
			return err
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}
	return nil
}

// хэш код для ключа партиции из номера счета
func (s *TransactionService) partitionKey(accountCode string) string {
	hash := sha256.Sum256([]byte(accountCode))
	return "acc:" + hex.EncodeToString(hash[:8])
}
