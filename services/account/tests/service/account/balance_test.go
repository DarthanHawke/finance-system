// Пакет service_test - тесты для пакета service
package service_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"account-service/internal/models"
	service "account-service/internal/service/account"
	"account-service/tests/service/account/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type TestBalanceService struct {
	BalanceService *service.BalanceService
	MockBM         *mocks.MockBalanceManager
	MockEM         *mocks.MockEventManager
	TransactionID  uuid.UUID
	AccountCode    string
	Amount         float64
}

func newTestBalanceService(t *testing.T) *TestBalanceService {
	t.Helper()

	mockBM := new(mocks.MockBalanceManager)
	mockEM := new(mocks.MockEventManager)
	logger := zap.NewNop()

	svc := service.NewBalanceService(mockBM, mockEM, logger)

	return &TestBalanceService{
		BalanceService: svc,
		MockBM:         mockBM,
		MockEM:         mockEM,
		TransactionID:  uuid.New(),
		AccountCode:    "RUB0000000000000000000000001",
		Amount:         1500.00,
	}
}

// newRequestEvent создаёт входящее событие-запрос с BalanceRequestPayload
func (f *TestBalanceService) newRequestEvent(eventType string) *models.Event {
	payload := models.BalanceRequestPayload{
		AccountCode: f.AccountCode,
		Amount:      f.Amount,
		Currency:    "RUB",
	}
	payloadBytes, _ := json.Marshal(payload)

	return &models.Event{
		ID:            uuid.New(),
		TransactionID: f.TransactionID,
		Type:          eventType,
		Status:        models.EventStatusPending,
		Source:        "transaction-service",
		CreatedAt:     time.Now(),
		Payload:       payloadBytes,
	}
}

// newBlockEvent создаёт событие блокировки с UpdateAccountRequest payload
func (f *TestBalanceService) newBlockEvent() *models.Event {
	payload := models.UpdateAccountRequest{
		Code: f.AccountCode,
	}
	payloadBytes, _ := json.Marshal(payload)

	return &models.Event{
		ID:            uuid.New(),
		TransactionID: f.TransactionID,
		Type:          models.EventBlockRequest,
		Status:        models.EventStatusPending,
		Source:        "transaction-service",
		CreatedAt:     time.Now(),
		Payload:       payloadBytes,
	}
}

// assertCreateEventSuccess проверяет, что CreateEvent вызван с success=true
func (f *TestBalanceService) assertCreateEventSuccess(eventType string) *mock.Call {
	return f.MockEM.On("CreateEvent", mock.Anything, mock.MatchedBy(func(req *models.CreateEventRequest) bool {
		if req.Event.Type != eventType {
			return false
		}
		var payload models.BalanceResponsePayload
		if err := json.Unmarshal(req.Payload, &payload); err != nil {
			return false
		}
		return payload.Success
	})).Return(nil)
}

// assertCreateEventFailure проверяет, что CreateEvent вызван с success=false
func (f *TestBalanceService) assertCreateEventFailure(eventType string, reasonContains string) *mock.Call {
	return f.MockEM.On("CreateEvent", mock.Anything, mock.MatchedBy(func(req *models.CreateEventRequest) bool {
		if req.Event.Type != eventType {
			return false
		}
		var payload models.BalanceResponsePayload
		if err := json.Unmarshal(req.Payload, &payload); err != nil {
			return false
		}
		if payload.Success {
			return false
		}
		if reasonContains != "" {
			// здесь просто проверяем что reason не пустой
			return payload.Reason != ""
		}
		return true
	})).Return(nil)
}

func TestBalanceService_HandleFreezeRequest_Success(t *testing.T) {
	f := newTestBalanceService(t)
	event := f.newRequestEvent(models.EventFreezeRequest)

	f.MockBM.On("FreezeBalance", mock.Anything,
		mock.MatchedBy(func(req *models.BalanceRequest) bool {
			return req.Code == f.AccountCode && req.Amount == f.Amount
		}),
		mock.MatchedBy(func(event *models.CreateEventRequest) bool {
			return event.Event.Type == models.EventFreezeResponse
		}),
	).Return(nil)

	err := f.BalanceService.HandleFreezeRequest(context.Background(), event)
	require.NoError(t, err)

	f.MockBM.AssertExpectations(t)
}

func TestBalanceService_HandleFreezeRequest_InsufficientFunds(t *testing.T) {
	f := newTestBalanceService(t)
	event := f.newRequestEvent(models.EventFreezeRequest)

	// BalanceManager возвращает ошибку
	f.MockBM.On("FreezeBalance", mock.Anything, mock.Anything, mock.Anything).
		Return(assert.AnError)

	// ожидаем, что CreateEvent вызовется с success=false
	f.assertCreateEventFailure(models.EventFreezeResponse, "insufficient")

	err := f.BalanceService.HandleFreezeRequest(context.Background(), event)
	require.NoError(t, err)

	f.MockBM.AssertExpectations(t)
	f.MockEM.AssertExpectations(t)
}

func TestBalanceService_HandleUnfreezeRequest_Success(t *testing.T) {
	f := newTestBalanceService(t)
	event := f.newRequestEvent(models.EventUnfreezeRequest)

	f.MockBM.On("UnfreezeBalance", mock.Anything,
		mock.MatchedBy(func(req *models.BalanceRequest) bool {
			return req.Code == f.AccountCode && req.Amount == f.Amount
		}),
		mock.Anything,
	).Return(nil)

	err := f.BalanceService.HandleUnfreezeRequest(context.Background(), event)
	require.NoError(t, err)

	f.MockBM.AssertExpectations(t)
}

func TestBalanceService_HandleUnfreezeRequest_Failure(t *testing.T) {
	f := newTestBalanceService(t)
	event := f.newRequestEvent(models.EventUnfreezeRequest)

	f.MockBM.On("UnfreezeBalance", mock.Anything, mock.Anything, mock.Anything).
		Return(assert.AnError)
	f.assertCreateEventFailure(models.EventUnfreezeResponse, "")

	err := f.BalanceService.HandleUnfreezeRequest(context.Background(), event)
	require.NoError(t, err)

	f.MockBM.AssertExpectations(t)
	f.MockEM.AssertExpectations(t)
}

func TestBalanceService_HandleReserveRequest_Success(t *testing.T) {
	f := newTestBalanceService(t)
	event := f.newRequestEvent(models.EventReserveRequest)

	f.MockBM.On("ReserveDeposit", mock.Anything,
		mock.MatchedBy(func(req *models.BalanceRequest) bool {
			return req.Code == f.AccountCode && req.Amount == f.Amount
		}),
		mock.Anything,
	).Return(nil)

	err := f.BalanceService.HandleReserveRequest(context.Background(), event)
	require.NoError(t, err)

	f.MockBM.AssertExpectations(t)
}

func TestBalanceService_HandleReserveRequest_Failure(t *testing.T) {
	f := newTestBalanceService(t)
	event := f.newRequestEvent(models.EventReserveRequest)

	f.MockBM.On("ReserveDeposit", mock.Anything, mock.Anything, mock.Anything).
		Return(assert.AnError)
	f.assertCreateEventFailure(models.EventReserveResponse, "")

	err := f.BalanceService.HandleReserveRequest(context.Background(), event)
	require.NoError(t, err)

	f.MockBM.AssertExpectations(t)
	f.MockEM.AssertExpectations(t)
}

func TestBalanceService_HandleUnreserveRequest_Success(t *testing.T) {
	f := newTestBalanceService(t)
	event := f.newRequestEvent(models.EventUnreserveRequest)

	f.MockBM.On("UnreserveDeposit", mock.Anything,
		mock.MatchedBy(func(req *models.BalanceRequest) bool {
			return req.Code == f.AccountCode && req.Amount == f.Amount
		}),
		mock.Anything,
	).Return(nil)

	err := f.BalanceService.HandleUnreserveRequest(context.Background(), event)
	require.NoError(t, err)

	f.MockBM.AssertExpectations(t)
}

func TestBalanceService_HandleWithdrawRequest_Success(t *testing.T) {
	f := newTestBalanceService(t)
	event := f.newRequestEvent(models.EventWithdrawRequest)

	f.MockBM.On("WithdrawBalance", mock.Anything,
		mock.MatchedBy(func(req *models.BalanceRequest) bool {
			return req.Code == f.AccountCode && req.Amount == f.Amount
		}),
		mock.Anything,
	).Return(nil)

	err := f.BalanceService.HandleWithdrawRequest(context.Background(), event)
	require.NoError(t, err)

	f.MockBM.AssertExpectations(t)
}

func TestBalanceService_HandleWithdrawRequest_Failure(t *testing.T) {
	f := newTestBalanceService(t)
	event := f.newRequestEvent(models.EventWithdrawRequest)

	f.MockBM.On("WithdrawBalance", mock.Anything, mock.Anything, mock.Anything).
		Return(assert.AnError)
	f.assertCreateEventFailure(models.EventWithdrawResponse, "")

	err := f.BalanceService.HandleWithdrawRequest(context.Background(), event)
	require.NoError(t, err)

	f.MockBM.AssertExpectations(t)
	f.MockEM.AssertExpectations(t)
}

func TestBalanceService_HandleRefundRequest_Success(t *testing.T) {
	f := newTestBalanceService(t)
	event := f.newRequestEvent(models.EventRefundRequest)

	// Refund использует FreezeBalance для возврата средств
	f.MockBM.On("FreezeBalance", mock.Anything,
		mock.MatchedBy(func(req *models.BalanceRequest) bool {
			return req.Code == f.AccountCode && req.Amount == f.Amount
		}),
		mock.Anything,
	).Return(nil)

	err := f.BalanceService.HandleRefundRequest(context.Background(), event)
	require.NoError(t, err)

	f.MockBM.AssertExpectations(t)
}

func TestBalanceService_HandleDepositRequest_Success(t *testing.T) {
	f := newTestBalanceService(t)
	event := f.newRequestEvent(models.EventDepositRequest)

	f.MockBM.On("DepositBalance", mock.Anything,
		mock.MatchedBy(func(req *models.BalanceRequest) bool {
			return req.Code == f.AccountCode && req.Amount == f.Amount
		}),
		mock.Anything,
	).Return(nil)

	err := f.BalanceService.HandleDepositRequest(context.Background(), event)
	require.NoError(t, err)

	f.MockBM.AssertExpectations(t)
}

func TestBalanceService_HandleDepositRequest_Failure(t *testing.T) {
	f := newTestBalanceService(t)
	event := f.newRequestEvent(models.EventDepositRequest)

	f.MockBM.On("DepositBalance", mock.Anything, mock.Anything, mock.Anything).
		Return(assert.AnError)
	f.assertCreateEventFailure(models.EventDepositResponse, "")

	err := f.BalanceService.HandleDepositRequest(context.Background(), event)
	require.NoError(t, err)

	f.MockBM.AssertExpectations(t)
	f.MockEM.AssertExpectations(t)
}

func TestBalanceService_HandleBlockAccount_Success(t *testing.T) {
	f := newTestBalanceService(t)
	event := f.newBlockEvent()

	f.MockBM.On("BlockAccountWithEvent", mock.Anything,
		mock.MatchedBy(func(req *models.UpdateAccountRequest) bool {
			return req.Code == f.AccountCode
		}),
		mock.Anything,
	).Return(nil)

	err := f.BalanceService.HandleBlockAccount(context.Background(), event)
	require.NoError(t, err)

	f.MockBM.AssertExpectations(t)
}

func TestBalanceService_HandleBlockAccount_Failure(t *testing.T) {
	f := newTestBalanceService(t)
	event := f.newBlockEvent()

	f.MockBM.On("BlockAccountWithEvent", mock.Anything, mock.Anything, mock.Anything).
		Return(assert.AnError)
	f.assertCreateEventFailure(models.EventBlockResponse, "")

	err := f.BalanceService.HandleBlockAccount(context.Background(), event)
	require.NoError(t, err)

	f.MockBM.AssertExpectations(t)
	f.MockEM.AssertExpectations(t)
}

func TestBalanceService_FullSagaPath(t *testing.T) {
	f := newTestBalanceService(t)
	ctx := context.Background()

	// шаг 1: freeze.request
	freezeEvent := f.newRequestEvent(models.EventFreezeRequest)
	f.MockBM.On("FreezeBalance", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	err := f.BalanceService.HandleFreezeRequest(ctx, freezeEvent)
	require.NoError(t, err)

	// шаг 2: reserve.request
	reserveEvent := f.newRequestEvent(models.EventReserveRequest)
	f.MockBM.On("ReserveDeposit", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	err = f.BalanceService.HandleReserveRequest(ctx, reserveEvent)
	require.NoError(t, err)

	// шаг 3: withdraw.request
	withdrawEvent := f.newRequestEvent(models.EventWithdrawRequest)
	f.MockBM.On("WithdrawBalance", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	err = f.BalanceService.HandleWithdrawRequest(ctx, withdrawEvent)
	require.NoError(t, err)

	// шаг 4: deposit.request
	depositEvent := f.newRequestEvent(models.EventDepositRequest)
	f.MockBM.On("DepositBalance", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	err = f.BalanceService.HandleDepositRequest(ctx, depositEvent)
	require.NoError(t, err)

	// Проверяем, что все 4 операции были вызваны
	f.MockBM.AssertExpectations(t)
}

func TestBalanceService_FullSagaPath_Rollback(t *testing.T) {
	f := newTestBalanceService(t)
	ctx := context.Background()

	// шаги 1-3: успешно
	freezeEvent := f.newRequestEvent(models.EventFreezeRequest)
	f.MockBM.On("FreezeBalance", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	err := f.BalanceService.HandleFreezeRequest(ctx, freezeEvent)
	require.NoError(t, err)

	reserveEvent := f.newRequestEvent(models.EventReserveRequest)
	f.MockBM.On("ReserveDeposit", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	err = f.BalanceService.HandleReserveRequest(ctx, reserveEvent)
	require.NoError(t, err)

	withdrawEvent := f.newRequestEvent(models.EventWithdrawRequest)
	f.MockBM.On("WithdrawBalance", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	err = f.BalanceService.HandleWithdrawRequest(ctx, withdrawEvent)
	require.NoError(t, err)

	// шаг 4: deposit.request (FAIL)
	depositEvent := f.newRequestEvent(models.EventDepositRequest)
	f.MockBM.On("DepositBalance", mock.Anything, mock.Anything, mock.Anything).
		Return(assert.AnError)
	f.assertCreateEventFailure(models.EventDepositResponse, "")

	err = f.BalanceService.HandleDepositRequest(ctx, depositEvent)
	require.NoError(t, err)

	// компенсация: refund.request
	refundEvent := f.newRequestEvent(models.EventRefundRequest)
	f.MockBM.On("FreezeBalance", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	err = f.BalanceService.HandleRefundRequest(ctx, refundEvent)
	require.NoError(t, err)

	// компенсация: unreserve.request
	unreserveEvent := f.newRequestEvent(models.EventUnreserveRequest)
	f.MockBM.On("UnreserveDeposit", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	err = f.BalanceService.HandleUnreserveRequest(ctx, unreserveEvent)
	require.NoError(t, err)

	// компенсация: unfreeze.request
	unfreezeEvent := f.newRequestEvent(models.EventUnfreezeRequest)
	f.MockBM.On("UnfreezeBalance", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	err = f.BalanceService.HandleUnfreezeRequest(ctx, unfreezeEvent)
	require.NoError(t, err)

	f.MockBM.AssertExpectations(t)
	f.MockEM.AssertExpectations(t)
}
