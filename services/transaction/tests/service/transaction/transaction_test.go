// пакет service_test - тесты для пакета service
package service_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"transaction-service/internal/models"
	service "transaction-service/internal/service/transaction"
	"transaction-service/tests/service/transaction/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestTransactionService содержит все зависимости для теста
type TestTransactionService struct {
	TransactionService *service.TransactionService
	MockTM             *mocks.MockTransactionManager
	MockEM             *mocks.MockEventManager
	MockISO            *mocks.MockIso8583Manager
	MockStan           *mocks.MockStanManager
	TransactionID      uuid.UUID
	SenderAccount      string
	RecipientAccount   string
	Amount             float64
	Currency           string
}

// newTestTransactionService создаёт новый TestTransactionService
func newTestTransactionService(t *testing.T) *TestTransactionService {
	t.Helper()

	mockTM := new(mocks.MockTransactionManager)
	mockEM := new(mocks.MockEventManager)
	mockISO := new(mocks.MockIso8583Manager)
	mockStan := new(mocks.MockStanManager)

	svc := service.NewTransactionService(mockTM, mockEM, mockISO, mockStan)

	return &TestTransactionService{
		TransactionService: svc,
		MockTM:             mockTM,
		MockEM:             mockEM,
		MockISO:            mockISO,
		TransactionID:      uuid.New(),
		SenderAccount:      "RUB0000000000000000000000001",
		RecipientAccount:   "RUB0000000000000000000000002",
		Amount:             1500.00,
		Currency:           "RUB",
	}
}

// newTransaction возвращает Transaction для тестов
func (f *TestTransactionService) newTransaction(senderType, recipientType string) *models.Transaction {
	return &models.Transaction{
		ID:                   f.TransactionID,
		Type:                 "transfer",
		Amount:               f.Amount,
		Currency:             f.Currency,
		SenderType:           senderType,
		SenderAccountCode:    f.SenderAccount,
		RecipientType:        recipientType,
		RecipientAccountCode: f.RecipientAccount,
		ProcessingCode:       "060000",
		Stan:                 "123456",
		AuthorizationCode:    "AUTH001",
		Status:               models.TransactionStatusPending,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
}

// newEventFromPayload создаёт Event с указанным типом и payload
func (f *TestTransactionService) newEvent(eventType string, payload any) *models.Event {
	payloadBytes, _ := json.Marshal(payload)
	return &models.Event{
		ID:            uuid.New(),
		TransactionID: f.TransactionID,
		Type:          eventType,
		Status:        models.EventStatusPending,
		Source:        models.Source,
		CreatedAt:     time.Now(),
		Payload:       payloadBytes,
	}
}

// assertCreateEvent проверяет, что CreateEvent был вызван с нужным типом
func (f *TestTransactionService) assertCreateEvent(expectedType string) *mock.Call {
	return f.MockEM.On("CreateEvent", mock.Anything, mock.MatchedBy(func(req *models.CreateEventRequest) bool {
		return req.Event.Type == expectedType
	})).Return(nil)
}

func TestTransactionService_InternalTransfer_HappyPath(t *testing.T) {
	f := newTestTransactionService(t)

	tx := f.newTransaction(models.AccountInternal, models.AccountInternal)
	ctx := context.Background()

	// шаг 1: HandleTransactionResponse

	// Событие, которое пришло бы из Kafka (transaction-commands)
	startPayload := &models.CreateTransactionRequest{Transaction: tx}
	startEvent := f.newEvent(models.EventTransactionRequest, startPayload)

	// ожидаем создание транзакции и freeze.request
	f.MockTM.On("CreateTransaction", ctx,
		mock.MatchedBy(func(req *models.CreateTransactionRequest) bool {
			return req.ID == f.TransactionID && req.Status == models.TransactionStatusPending
		}),
		mock.MatchedBy(func(event *models.CreateEventRequest) bool {
			return event.Event.Type == models.EventFreezeRequest
		}),
	).Return(nil)

	err := f.TransactionService.HandleTransactionResponse(ctx, startEvent)
	require.NoError(t, err)

	// шаг 2: HandleFreezeResponse (success)

	freezeRespPayload := &models.BalanceResponsePayload{Success: true}
	freezeRespEvent := f.newEvent(models.EventFreezeResponse, freezeRespPayload)

	// ожидаем: GetTransaction + CreateEvent(reserve.request)
	f.MockTM.On("GetTransaction", ctx,
		mock.MatchedBy(func(req *models.GetTransactionRequest) bool {
			return req.ID == f.TransactionID
		}),
	).Return(models.GetTransactionResponse{Transaction: tx}, nil)

	f.assertCreateEvent(models.EventReserveRequest)

	err = f.TransactionService.HandleFreezeResponse(ctx, freezeRespEvent)
	require.NoError(t, err)

	// шаг 3: HandleReserveResponse (success)

	reserveRespPayload := &models.BalanceResponsePayload{Success: true}
	reserveRespEvent := f.newEvent(models.EventReserveResponse, reserveRespPayload)

	// ожидаем: GetTransaction + CreateEvent(withdraw.request)
	f.MockTM.On("GetTransaction", ctx, mock.Anything).
		Return(models.GetTransactionResponse{Transaction: tx}, nil)

	f.assertCreateEvent(models.EventWithdrawRequest)

	err = f.TransactionService.HandleReserveResponse(ctx, reserveRespEvent)
	require.NoError(t, err)

	// шаг 4: HandleWithdrawResponse (success)

	withdrawRespPayload := &models.BalanceResponsePayload{Success: true}
	withdrawRespEvent := f.newEvent(models.EventWithdrawResponse, withdrawRespPayload)

	// ожидаем: GetTransaction + CreateEvent(deposit.request)
	f.MockTM.On("GetTransaction", ctx, mock.Anything).
		Return(models.GetTransactionResponse{Transaction: tx}, nil)

	f.assertCreateEvent(models.EventDepositRequest)

	err = f.TransactionService.HandleWithdrawResponse(ctx, withdrawRespEvent)
	require.NoError(t, err)

	// шаг 5: HandleDepositResponse (success)

	depositRespPayload := &models.BalanceResponsePayload{Success: true}
	depositRespEvent := f.newEvent(models.EventDepositResponse, depositRespPayload)

	// ожидаем: GetTransaction + UpdateTransactionStatus(completed)
	f.MockTM.On("GetTransaction", ctx, mock.Anything).
		Return(models.GetTransactionResponse{Transaction: tx}, nil)

	f.MockTM.On("UpdateTransactionStatus", ctx,
		mock.MatchedBy(func(req *models.UpdateTransactionStatusRequest) bool {
			return req.ID == f.TransactionID && req.Status == models.TransactionStatusCompleted
		}),
	).Return(nil)

	err = f.TransactionService.HandleDepositResponse(ctx, depositRespEvent)
	require.NoError(t, err)

	// проверяем, что все ожидаемые вызовы состоялись
	f.MockTM.AssertExpectations(t)
	f.MockEM.AssertExpectations(t)
}

func TestTransactionService_InternalTransfer_RollbackAtDeposit(t *testing.T) {
	f := newTestTransactionService(t)

	tx := f.newTransaction(models.AccountInternal, models.AccountInternal)
	ctx := context.Background()

	// шаги 1-4: всё успешно (как в Happy Path)

	startPayload := &models.CreateTransactionRequest{Transaction: tx}
	startEvent := f.newEvent(models.EventTransactionRequest, startPayload)

	f.MockTM.On("CreateTransaction", ctx, mock.Anything, mock.Anything).Return(nil)

	err := f.TransactionService.HandleTransactionResponse(ctx, startEvent)
	require.NoError(t, err)

	// freeze.response success
	freezeRespEvent := f.newEvent(models.EventFreezeResponse, &models.BalanceResponsePayload{Success: true})
	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)
	f.assertCreateEvent(models.EventReserveRequest)
	err = f.TransactionService.HandleFreezeResponse(ctx, freezeRespEvent)
	require.NoError(t, err)

	// reserve.response success
	reserveRespEvent := f.newEvent(models.EventReserveResponse, &models.BalanceResponsePayload{Success: true})
	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)
	f.assertCreateEvent(models.EventWithdrawRequest)
	err = f.TransactionService.HandleReserveResponse(ctx, reserveRespEvent)
	require.NoError(t, err)

	// withdraw.response success
	withdrawRespEvent := f.newEvent(models.EventWithdrawResponse, &models.BalanceResponsePayload{Success: true})
	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)
	f.assertCreateEvent(models.EventDepositRequest)
	err = f.TransactionService.HandleWithdrawResponse(ctx, withdrawRespEvent)
	require.NoError(t, err)

	// шаг 5: HandleDepositResponse (FAIL)

	depositRespPayload := &models.BalanceResponsePayload{
		Success: false,
		Reason:  "insufficient funds in reserve",
	}
	depositRespEvent := f.newEvent(models.EventDepositResponse, depositRespPayload)

	f.MockTM.On("GetTransaction", ctx, mock.Anything).
		Return(models.GetTransactionResponse{Transaction: tx}, nil)

	// ожидаем компенсацию: refund.request
	f.assertCreateEvent(models.EventRefundRequest)

	err = f.TransactionService.HandleDepositResponse(ctx, depositRespEvent)
	require.NoError(t, err)

	// компенсация: HandleRefundResponse (success)

	refundRespPayload := &models.BalanceResponsePayload{Success: true}
	refundRespEvent := f.newEvent(models.EventRefundResponse, refundRespPayload)

	f.MockTM.On("GetTransaction", ctx, mock.Anything).
		Return(models.GetTransactionResponse{Transaction: tx}, nil)

	// после успешного refund -> unreserve.request
	f.assertCreateEvent(models.EventUnreserveRequest)

	err = f.TransactionService.HandleRefundResponse(ctx, refundRespEvent)
	require.NoError(t, err)

	// компенсация: HandleUnreserveResponse (success)

	unreserveRespPayload := &models.BalanceResponsePayload{Success: true}
	unreserveRespEvent := f.newEvent(models.EventUnreserveResponse, unreserveRespPayload)

	f.MockTM.On("GetTransaction", ctx, mock.Anything).
		Return(models.GetTransactionResponse{Transaction: tx}, nil)

	// после успешного unreserve -> unfreeze.request
	f.assertCreateEvent(models.EventUnfreezeRequest)

	err = f.TransactionService.HandleUnreserveResponse(ctx, unreserveRespEvent)
	require.NoError(t, err)

	// компенсация: HandleUnfreezeResponse (success)

	unfreezeRespPayload := &models.BalanceResponsePayload{Success: true}
	unfreezeRespEvent := f.newEvent(models.EventUnfreezeResponse, unfreezeRespPayload)

	f.MockTM.On("GetTransaction", ctx, mock.Anything).
		Return(models.GetTransactionResponse{Transaction: tx}, nil)

	// после успешного unfreeze -> cancelled
	f.MockTM.On("UpdateTransactionStatus", ctx,
		mock.MatchedBy(func(req *models.UpdateTransactionStatusRequest) bool {
			return req.ID == f.TransactionID && req.Status == models.TransactionStatusCancelled
		}),
	).Return(nil)

	err = f.TransactionService.HandleUnfreezeResponse(ctx, unfreezeRespEvent)
	require.NoError(t, err)

	// проверяем
	f.MockTM.AssertExpectations(t)
	f.MockEM.AssertExpectations(t)
}

func TestTransactionService_OutgoingTransfer_HappyPath(t *testing.T) {
	f := newTestTransactionService(t)

	tx := f.newTransaction(models.AccountInternal, models.AccountExternal)
	ctx := context.Background()

	// шаг 1: HandleTransactionResponse
	startPayload := &models.CreateTransactionRequest{Transaction: tx}
	startEvent := f.newEvent(models.EventTransactionRequest, startPayload)

	f.MockTM.On("CreateTransaction", ctx, mock.Anything,
		mock.MatchedBy(func(event *models.CreateEventRequest) bool {
			return event.Event.Type == models.EventFreezeRequest
		}),
	).Return(nil)

	err := f.TransactionService.HandleTransactionResponse(ctx, startEvent)
	require.NoError(t, err)

	// шаг 2: HandleFreezeResponse (success)
	freezeRespEvent := f.newEvent(models.EventFreezeResponse, &models.BalanceResponsePayload{Success: true})
	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)

	// мокаем ISO: для external.request нужен ISO-запрос
	isoRequest := &models.ISO8583Message{
		MTI:      "0200",
		STAN:     tx.Stan,
		Currency: "643",
		Amount:   "000000001500",
	}
	f.MockISO.On("CreateFinancialRequest", tx).Return(isoRequest, nil)
	f.assertCreateEvent(models.EventExternalRequest)

	err = f.TransactionService.HandleFreezeResponse(ctx, freezeRespEvent)
	require.NoError(t, err)

	// шаг 3: HandleExternalResponse (success)
	isoRespPayload := &models.ISO8583Payload{
		ISOMessage: &models.ISO8583Message{
			MTI:          "0210",
			ResponseCode: models.RCSuccess,
		},
		Success: true,
	}
	externalRespEvent := f.newEvent(models.EventExternalResponse, isoRespPayload)

	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)
	f.assertCreateEvent(models.EventWithdrawRequest)

	err = f.TransactionService.HandleExternalResponse(ctx, externalRespEvent)
	require.NoError(t, err)

	// шаг 4: HandleWithdrawResponse (success)
	withdrawRespEvent := f.newEvent(models.EventWithdrawResponse, &models.BalanceResponsePayload{Success: true})
	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)

	// мокаем ISO для external.commit
	isoCommit := &models.ISO8583Message{
		MTI:          "0210",
		ResponseCode: models.RCSuccess,
	}
	f.MockISO.On("CreateFinancialResponse", tx, true, "").Return(isoCommit, nil)
	f.assertCreateEvent(models.EventExternalCommit)

	// после external.commit сразу completed
	f.MockTM.On("UpdateTransactionStatus", ctx,
		mock.MatchedBy(func(req *models.UpdateTransactionStatusRequest) bool {
			return req.ID == f.TransactionID && req.Status == models.TransactionStatusCompleted
		}),
	).Return(nil)

	err = f.TransactionService.HandleWithdrawResponse(ctx, withdrawRespEvent)
	require.NoError(t, err)

	f.MockTM.AssertExpectations(t)
	f.MockEM.AssertExpectations(t)
	f.MockISO.AssertExpectations(t)
}

func TestTransactionService_OutgoingTransfer_RollbackAtExternal(t *testing.T) {
	f := newTestTransactionService(t)

	tx := f.newTransaction(models.AccountInternal, models.AccountExternal)
	ctx := context.Background()

	// шаг 1: HandleTransactionResponse
	startEvent := f.newEvent(models.EventTransactionRequest, &models.CreateTransactionRequest{Transaction: tx})
	f.MockTM.On("CreateTransaction", ctx, mock.Anything, mock.Anything).Return(nil)
	err := f.TransactionService.HandleTransactionResponse(ctx, startEvent)
	require.NoError(t, err)

	// шаг 2: HandleFreezeResponse (success)
	freezeRespEvent := f.newEvent(models.EventFreezeResponse, &models.BalanceResponsePayload{Success: true})
	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)

	isoRequest := &models.ISO8583Message{MTI: "0200", STAN: tx.Stan}
	f.MockISO.On("CreateFinancialRequest", tx).Return(isoRequest, nil)
	f.assertCreateEvent(models.EventExternalRequest)
	err = f.TransactionService.HandleFreezeResponse(ctx, freezeRespEvent)
	require.NoError(t, err)

	// шаг 3: HandleExternalResponse (FAIL)
	isoRespPayload := &models.ISO8583Payload{
		ISOMessage: &models.ISO8583Message{
			MTI:          "0210",
			ResponseCode: models.RCInsufficientFunds,
		},
		Success:      false,
		ErrorCode:    models.RCInsufficientFunds,
		ErrorMessage: "insufficient funds in external bank",
	}
	externalRespEvent := f.newEvent(models.EventExternalResponse, isoRespPayload)

	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)

	// компенсация: unfreeze.request
	f.assertCreateEvent(models.EventUnfreezeRequest)

	err = f.TransactionService.HandleExternalResponse(ctx, externalRespEvent)
	require.NoError(t, err)

	// компенсация: HandleUnfreezeResponse (success)
	unfreezeRespEvent := f.newEvent(models.EventUnfreezeResponse, &models.BalanceResponsePayload{Success: true})
	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)

	f.MockTM.On("UpdateTransactionStatus", ctx,
		mock.MatchedBy(func(req *models.UpdateTransactionStatusRequest) bool {
			return req.ID == f.TransactionID && req.Status == models.TransactionStatusCancelled
		}),
	).Return(nil)

	err = f.TransactionService.HandleUnfreezeResponse(ctx, unfreezeRespEvent)
	require.NoError(t, err)

	f.MockTM.AssertExpectations(t)
	f.MockEM.AssertExpectations(t)
	f.MockISO.AssertExpectations(t)
}

func TestTransactionService_IncomingTransfer_HappyPath(t *testing.T) {
	f := newTestTransactionService(t)

	tx := f.newTransaction(models.AccountExternal, models.AccountInternal)
	ctx := context.Background()

	// шаг 1: HandleTransactionResponse (SenderType=external -> reserve)
	startPayload := &models.CreateTransactionRequest{Transaction: tx}
	startEvent := f.newEvent(models.EventTransactionRequest, startPayload)

	f.MockTM.On("CreateTransaction", ctx, mock.Anything,
		mock.MatchedBy(func(event *models.CreateEventRequest) bool {
			return event.Event.Type == models.EventReserveRequest
		}),
	).Return(nil)

	err := f.TransactionService.HandleTransactionResponse(ctx, startEvent)
	require.NoError(t, err)

	// шаг 2: HandleReserveResponse (success)
	reserveRespEvent := f.newEvent(models.EventReserveResponse, &models.BalanceResponsePayload{Success: true})
	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)

	isoRequest := &models.ISO8583Message{MTI: "0200", STAN: tx.Stan}
	f.MockISO.On("CreateFinancialRequest", tx).Return(isoRequest, nil)
	f.assertCreateEvent(models.EventExternalRequest)

	err = f.TransactionService.HandleReserveResponse(ctx, reserveRespEvent)
	require.NoError(t, err)

	// шаг 3: HandleExternalResponse (success)
	externalRespEvent := f.newEvent(models.EventExternalResponse, &models.ISO8583Payload{
		ISOMessage: &models.ISO8583Message{MTI: "0210", ResponseCode: models.RCSuccess},
		Success:    true,
	})
	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)
	f.assertCreateEvent(models.EventDepositRequest)

	err = f.TransactionService.HandleExternalResponse(ctx, externalRespEvent)
	require.NoError(t, err)

	// шаг 4: HandleDepositResponse (success)
	depositRespEvent := f.newEvent(models.EventDepositResponse, &models.BalanceResponsePayload{Success: true})
	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)

	isoCommit := &models.ISO8583Message{MTI: "0210", ResponseCode: models.RCSuccess}
	f.MockISO.On("CreateFinancialResponse", tx, true, "").Return(isoCommit, nil)
	f.assertCreateEvent(models.EventExternalCommit)

	f.MockTM.On("UpdateTransactionStatus", ctx,
		mock.MatchedBy(func(req *models.UpdateTransactionStatusRequest) bool {
			return req.ID == f.TransactionID && req.Status == models.TransactionStatusCompleted
		}),
	).Return(nil)

	err = f.TransactionService.HandleDepositResponse(ctx, depositRespEvent)
	require.NoError(t, err)

	f.MockTM.AssertExpectations(t)
	f.MockEM.AssertExpectations(t)
	f.MockISO.AssertExpectations(t)
}

func TestTransactionService_IncomingTransfer_RollbackAtDeposit(t *testing.T) {
	f := newTestTransactionService(t)

	tx := f.newTransaction(models.AccountExternal, models.AccountInternal)
	ctx := context.Background()

	// шаги 1-3: успешно
	startEvent := f.newEvent(models.EventTransactionRequest, &models.CreateTransactionRequest{Transaction: tx})
	f.MockTM.On("CreateTransaction", ctx, mock.Anything, mock.Anything).Return(nil)
	err := f.TransactionService.HandleTransactionResponse(ctx, startEvent)
	require.NoError(t, err)

	reserveRespEvent := f.newEvent(models.EventReserveResponse, &models.BalanceResponsePayload{Success: true})
	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)
	f.MockISO.On("CreateFinancialRequest", tx).Return(&models.ISO8583Message{}, nil)
	f.assertCreateEvent(models.EventExternalRequest)
	err = f.TransactionService.HandleReserveResponse(ctx, reserveRespEvent)
	require.NoError(t, err)

	externalRespEvent := f.newEvent(models.EventExternalResponse, &models.ISO8583Payload{
		ISOMessage: &models.ISO8583Message{ResponseCode: models.RCSuccess},
		Success:    true,
	})
	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)
	f.assertCreateEvent(models.EventDepositRequest)
	err = f.TransactionService.HandleExternalResponse(ctx, externalRespEvent)
	require.NoError(t, err)

	// шаг 4: HandleDepositResponse (FAIL!)
	depositRespEvent := f.newEvent(models.EventDepositResponse, &models.BalanceResponsePayload{
		Success: false,
		Reason:  "account blocked",
	})
	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)

	// компенсация: external.rollback
	isoRollback := &models.ISO8583Message{MTI: "0420", ResponseCode: models.RCSystemError}
	f.MockISO.On("CreateFinancialResponse", tx, false, "account blocked").Return(isoRollback, nil)
	f.assertCreateEvent(models.EventExternalRollback)

	err = f.TransactionService.HandleDepositResponse(ctx, depositRespEvent)
	require.NoError(t, err)

	// компенсация: HandleExternalRollbackResponse (success)
	rollbackRespEvent := f.newEvent(models.EventExternalRollback, &models.BalanceResponsePayload{Success: true})
	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)

	// после успешного external.rollback -> unreserve.request
	f.assertCreateEvent(models.EventUnreserveRequest)

	err = f.TransactionService.HandleExternalRollbackResponse(ctx, rollbackRespEvent)
	require.NoError(t, err)

	// компенсация: HandleUnreserveResponse (success) -> cancelled
	unreserveRespEvent := f.newEvent(models.EventUnreserveResponse, &models.BalanceResponsePayload{Success: true})
	f.MockTM.On("GetTransaction", ctx, mock.Anything).Return(models.GetTransactionResponse{Transaction: tx}, nil)

	f.MockTM.On("UpdateTransactionStatus", ctx,
		mock.MatchedBy(func(req *models.UpdateTransactionStatusRequest) bool {
			return req.ID == f.TransactionID && req.Status == models.TransactionStatusCancelled
		}),
	).Return(nil)

	err = f.TransactionService.HandleUnreserveResponse(ctx, unreserveRespEvent)
	require.NoError(t, err)

	f.MockTM.AssertExpectations(t)
	f.MockEM.AssertExpectations(t)
	f.MockISO.AssertExpectations(t)
}
