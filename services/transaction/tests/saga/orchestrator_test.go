package saga_test

import (
	"context"
	"testing"
	"time"

	"transaction-service/internal/models"
	"transaction-service/internal/saga"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockSagaManager struct {
	mock.Mock
}

func (m *MockSagaManager) Advance(ctx context.Context, req *models.AdvanceRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockSagaManager) AdvanceWithFirstStep(ctx context.Context, req *models.AdvanceWithFirstStepRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func newTestTransaction(txnType string) *models.Transaction {
	return &models.Transaction{
		ID:             uuid.New(),
		Type:           txnType,
		Status:         models.TransactionStatusProcessing,
		Amount:         1500.00,
		SourceCurrency: "RUB",
		TargetCurrency: "RUB",
		Parties: []models.TransactionParty{
			{
				PartyRole: models.PartyRoleSender,
				PartyType: models.PartyTypeInternal,
				Identifiers: map[string]string{
					"ACCOUNT_CODE": "40817810500000000001",
				},
			},
			{
				PartyRole: models.PartyRoleRecipient,
				PartyType: models.PartyTypeInternal,
				Identifiers: map[string]string{
					"ACCOUNT_CODE": "40817810500000000002",
				},
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func newParentTransaction(txnType string) *models.Transaction {
	return &models.Transaction{
		ID:     uuid.New(),
		Type:   txnType,
		Status: models.TransactionStatusCompleted,
	}
}

func assertEvents(t *testing.T, events []models.InsertEventRequest, expectedTypes ...string) {
	t.Helper()
	assert.Len(t, events, len(expectedTypes), "unexpected number of events")
	for i, eType := range expectedTypes {
		assert.Equal(t, eType, events[i].Event.Type, "event[%d] type mismatch", i)
	}
}

func assertSteps(t *testing.T, steps []models.InsertSagaStepRequest, expectedNames ...string) {
	t.Helper()
	assert.Len(t, steps, len(expectedNames), "unexpected number of steps")
	for i, name := range expectedNames {
		assert.Equal(t, name, steps[i].SagaStep.StepName, "step[%d] name mismatch", i)
	}
}

func TestOrchestrator_OnUsTransfer_HappyPath(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeOnUsTransfer)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Times(3)

	// 1. INIT → SENDER_FREEZING
	err := orch.ApplyWithFirstStep(context.Background(), tx, nil, saga.Apply{
		Outcome: saga.OutcomeSuccess,
		TraceID: "trace-1",
		SpanID:  "span-1",
	})
	require.NoError(t, err)
	tx.SagaType = models.SagaOnUsTransfer
	tx.SagaState = saga.StateSenderFreezing

	// 2. SENDER_FREEZING → CREDITING_RECIPIENT
	tx.UpdatedAt = time.Now()
	err = orch.Apply(context.Background(), tx, saga.Apply{
		TransactionID: tx.ID,
		Outcome:       saga.OutcomeSuccess,
		TraceID:       "trace-1",
		SpanID:        "span-1",
	})
	require.NoError(t, err)
	tx.SagaState = saga.StateCreditingRecipient

	// 3. CREDITING_RECIPIENT → SENDER_CAPTURING
	tx.UpdatedAt = time.Now()
	err = orch.Apply(context.Background(), tx, saga.Apply{
		TransactionID: tx.ID,
		Outcome:       saga.OutcomeSuccess,
	})
	require.NoError(t, err)
	tx.SagaState = saga.StateSenderCapturing

	// 4. SENDER_CAPTURING → COMPLETED
	tx.UpdatedAt = time.Now()
	err = orch.Apply(context.Background(), tx, saga.Apply{
		TransactionID: tx.ID,
		Outcome:       saga.OutcomeSuccess,
	})
	require.NoError(t, err)

	require.Len(t, calls, 4)

	// Init
	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventFreezeRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "freeze_sender")
	assert.Equal(t, saga.StateSenderFreezing, req1.UpdateSagaStateRequest.SagaState)

	// Credit
	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventCreditRequest)
	assertSteps(t, req2.Steps, "credit_recipient")
	assert.Equal(t, saga.StateCreditingRecipient, req2.UpdateSagaStateRequest.SagaState)

	// Capture
	req3 := calls[2].(*models.AdvanceRequest)
	assertEvents(t, req3.Events, models.EventCaptureRequest)
	assertSteps(t, req3.Steps, "capture_sender")
	assert.Equal(t, saga.StateSenderCapturing, req3.UpdateSagaStateRequest.SagaState)

	// Completed
	req4 := calls[3].(*models.AdvanceRequest)
	assertEvents(t, req4.Events, models.EventTransactionCompleted)
	assert.Empty(t, req4.Steps)
	assert.Equal(t, saga.StateCompleted, req4.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCompleted, *req4.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_OnUsFXTransfer_HappyPath(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeOnUsFXTransfer)
	fxDealID := uuid.New()
	tx.FXDealID = &fxDealID

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Times(5)

	// 1. INIT → SENDER_FREEZING
	err := orch.ApplyWithFirstStep(context.Background(), tx, nil, saga.Apply{
		Outcome: saga.OutcomeSuccess,
		TraceID: "trace-1",
		SpanID:  "span-1",
	})
	require.NoError(t, err)
	tx.SagaType = models.SagaOnUsFXTransfer
	tx.SagaState = saga.StateSenderFreezing

	// 2. SENDER_FREEZING → ACTIVATING_FX
	tx.UpdatedAt = time.Now()
	err = orch.Apply(context.Background(), tx, saga.Apply{
		TransactionID: tx.ID,
		Outcome:       saga.OutcomeSuccess,
	})
	require.NoError(t, err)
	tx.SagaState = saga.StateActivatingFX

	// 3. ACTIVATING_FX → CREDITING_RECIPIENT
	tx.UpdatedAt = time.Now()
	err = orch.Apply(context.Background(), tx, saga.Apply{
		TransactionID: tx.ID,
		Outcome:       saga.OutcomeSuccess,
	})
	require.NoError(t, err)
	tx.SagaState = saga.StateCreditingRecipient

	// 4. CREDITING_RECIPIENT → SENDER_CAPTURING
	tx.UpdatedAt = time.Now()
	err = orch.Apply(context.Background(), tx, saga.Apply{
		TransactionID: tx.ID,
		Outcome:       saga.OutcomeSuccess,
	})
	require.NoError(t, err)
	tx.SagaState = saga.StateSenderCapturing

	// 5. SENDER_CAPTURING → COMMITTING_FX
	tx.UpdatedAt = time.Now()
	err = orch.Apply(context.Background(), tx, saga.Apply{
		TransactionID: tx.ID,
		Outcome:       saga.OutcomeSuccess,
	})
	require.NoError(t, err)
	tx.SagaState = saga.StateCommittingFX

	// 6. COMMITTING_FX → COMPLETED
	tx.UpdatedAt = time.Now()
	err = orch.Apply(context.Background(), tx, saga.Apply{
		TransactionID: tx.ID,
		Outcome:       saga.OutcomeSuccess,
	})
	require.NoError(t, err)

	require.Len(t, calls, 6)

	// Init
	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventFreezeRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "freeze_sender")
	assert.Equal(t, saga.StateSenderFreezing, req1.UpdateSagaStateRequest.SagaState)

	// Activate FX
	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventFXActivateRequest)
	assertSteps(t, req2.Steps, "activate_fx")
	assert.Equal(t, saga.StateActivatingFX, req2.UpdateSagaStateRequest.SagaState)

	// Credit Recipient
	req3 := calls[2].(*models.AdvanceRequest)
	assertEvents(t, req3.Events, models.EventCreditRequest)
	assertSteps(t, req3.Steps, "credit_recipient")
	assert.Equal(t, saga.StateCreditingRecipient, req3.UpdateSagaStateRequest.SagaState)

	// Capture Sender
	req4 := calls[3].(*models.AdvanceRequest)
	assertEvents(t, req4.Events, models.EventCaptureRequest)
	assertSteps(t, req4.Steps, "capture_sender")
	assert.Equal(t, saga.StateSenderCapturing, req4.UpdateSagaStateRequest.SagaState)

	// Commit FX
	req5 := calls[4].(*models.AdvanceRequest)
	assertEvents(t, req5.Events, models.EventFXCommitRequest)
	assertSteps(t, req5.Steps, "commit_fx")
	assert.Equal(t, saga.StateCommittingFX, req5.UpdateSagaStateRequest.SagaState)

	// Completed
	req6 := calls[5].(*models.AdvanceRequest)
	assertEvents(t, req6.Events, models.EventTransactionCompleted)
	assert.Empty(t, req6.Steps)
	assert.Equal(t, saga.StateCompleted, req6.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCompleted, *req6.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_OnUsTransfer_FreezeFails(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeOnUsTransfer)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	// 1. INIT → SENDER_FREEZING
	err := orch.ApplyWithFirstStep(context.Background(), tx, nil, saga.Apply{
		Outcome: saga.OutcomeSuccess,
		TraceID: "trace-1",
		SpanID:  "span-1",
	})
	require.NoError(t, err)
	tx.SagaState = saga.StateSenderFreezing

	// 2. Freeze fails → CANCELLED
	tx.UpdatedAt = time.Now()
	err = orch.Apply(context.Background(), tx, saga.Apply{
		TransactionID: tx.ID,
		Outcome:       saga.OutcomeFailed,
	})
	require.NoError(t, err)

	require.Len(t, calls, 2)

	// Init
	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventFreezeRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "freeze_sender")
	assert.Equal(t, saga.StateSenderFreezing, req1.UpdateSagaStateRequest.SagaState)

	// Cancelled
	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventTransactionCancelled)
	assert.Empty(t, req2.Steps)
	assert.Equal(t, saga.StateCancelled, req2.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCancelled, *req2.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_OnUsTransfer_CreditFails_Compensates(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeOnUsTransfer)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Times(3)

	// 1. INIT → SENDER_FREEZING
	orch.ApplyWithFirstStep(context.Background(), tx, nil, saga.Apply{Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateSenderFreezing

	// 2. SENDER_FREEZING → CREDITING_RECIPIENT
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateCreditingRecipient

	// 3. CREDITING_RECIPIENT → COMPENSATING_SENDER (credit fails → unfreeze)
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeFailed})
	tx.SagaState = saga.StateCompensatingSender

	// 4. COMPENSATING_SENDER → CANCELLED (unfreeze success)
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})

	require.Len(t, calls, 4)

	// Init
	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventFreezeRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "freeze_sender")

	// Credit
	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventCreditRequest)
	assertSteps(t, req2.Steps, "credit_recipient")
	assert.Equal(t, saga.StateCreditingRecipient, req2.UpdateSagaStateRequest.SagaState)

	// Unfreeze (compensation)
	req3 := calls[2].(*models.AdvanceRequest)
	assertEvents(t, req3.Events, models.EventUnfreezeRequest)
	assertSteps(t, req3.Steps, "unfreeze_sender")
	assert.Equal(t, saga.StateCompensatingSender, req3.UpdateSagaStateRequest.SagaState)

	// Cancelled
	req4 := calls[3].(*models.AdvanceRequest)
	assertEvents(t, req4.Events, models.EventTransactionCancelled)
	assert.Empty(t, req4.Steps)
	assert.Equal(t, saga.StateCancelled, req4.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCancelled, *req4.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_OnUsTransfer_CaptureFails_Compensates(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeOnUsTransfer)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Times(5)

	// 1. INIT → SENDER_FREEZING
	orch.ApplyWithFirstStep(context.Background(), tx, nil, saga.Apply{Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateSenderFreezing

	// 2. SENDER_FREEZING → CREDITING_RECIPIENT
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateCreditingRecipient

	// 3. CREDITING_RECIPIENT → SENDER_CAPTURING
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateSenderCapturing

	// 4. SENDER_CAPTURING → COMPENSATING_RECIPIENT (capture fails → reverse)
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeFailed})
	tx.SagaState = saga.StateCompensatingRecipient

	// 5. COMPENSATING_RECIPIENT → COMPENSATING_SENDER (reverse success → unfreeze)
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateCompensatingSender

	// 6. COMPENSATING_SENDER → CANCELLED (unfreeze success)
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})

	require.Len(t, calls, 6)

	// Init
	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventFreezeRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "freeze_sender")

	// Credit
	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventCreditRequest)
	assertSteps(t, req2.Steps, "credit_recipient")
	assert.Equal(t, saga.StateCreditingRecipient, req2.UpdateSagaStateRequest.SagaState)

	// Capture
	req3 := calls[2].(*models.AdvanceRequest)
	assertEvents(t, req3.Events, models.EventCaptureRequest)
	assertSteps(t, req3.Steps, "capture_sender")
	assert.Equal(t, saga.StateSenderCapturing, req3.UpdateSagaStateRequest.SagaState)

	// Reverse (compensation)
	req4 := calls[3].(*models.AdvanceRequest)
	assertEvents(t, req4.Events, models.EventDebitRequest)
	assertSteps(t, req4.Steps, "reversing_recipient")
	assert.Equal(t, saga.StateCompensatingRecipient, req4.UpdateSagaStateRequest.SagaState)

	// Unfreeze (compensation)
	req5 := calls[4].(*models.AdvanceRequest)
	assertEvents(t, req5.Events, models.EventUnfreezeRequest)
	assertSteps(t, req5.Steps, "unfreeze_sender")
	assert.Equal(t, saga.StateCompensatingSender, req5.UpdateSagaStateRequest.SagaState)

	// Cancelled
	req6 := calls[5].(*models.AdvanceRequest)
	assertEvents(t, req6.Events, models.EventTransactionCancelled)
	assert.Empty(t, req6.Steps)
	assert.Equal(t, saga.StateCancelled, req6.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCancelled, *req6.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_OnUsTransfer_ReverseFails_Blocks(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeOnUsTransfer)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Times(4)

	// 1. INIT → SENDER_FREEZING
	orch.ApplyWithFirstStep(context.Background(), tx, nil, saga.Apply{Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateSenderFreezing

	// 2. SENDER_FREEZING → CREDITING_RECIPIENT
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateCreditingRecipient

	// 3. CREDITING_RECIPIENT → SENDER_CAPTURING
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateSenderCapturing

	// 4. SENDER_CAPTURING → COMPENSATING_RECIPIENT (capture fails → reverse)
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeFailed})
	tx.SagaState = saga.StateCompensatingRecipient

	// 5. COMPENSATING_RECIPIENT → BLOCKED (reverse fails)
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeFailed})

	require.Len(t, calls, 5)

	// Init
	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventFreezeRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "freeze_sender")

	// Credit
	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventCreditRequest)
	assertSteps(t, req2.Steps, "credit_recipient")

	// Capture
	req3 := calls[2].(*models.AdvanceRequest)
	assertEvents(t, req3.Events, models.EventCaptureRequest)
	assertSteps(t, req3.Steps, "capture_sender")

	// Reverse (compensation)
	req4 := calls[3].(*models.AdvanceRequest)
	assertEvents(t, req4.Events, models.EventDebitRequest)
	assertSteps(t, req4.Steps, "reversing_recipient")
	assert.Equal(t, saga.StateCompensatingRecipient, req4.UpdateSagaStateRequest.SagaState)

	// Blocked
	req5 := calls[4].(*models.AdvanceRequest)
	assertEvents(t, req5.Events, models.EventBlockRequest, models.EventBlockRequest, models.EventTransactionBlocked)
	assertSteps(t, req5.Steps, "block_sender", "block_recipient")
	assert.Equal(t, saga.StateBlocked, req5.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusBlocked, *req5.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_OnUsTransfer_UnfreezeFails_Blocks(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeOnUsTransfer)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Times(3)

	// 1. INIT → SENDER_FREEZING
	orch.ApplyWithFirstStep(context.Background(), tx, nil, saga.Apply{Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateSenderFreezing

	// 2. SENDER_FREEZING → CREDITING_RECIPIENT
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateCreditingRecipient

	// 3. CREDITING_RECIPIENT → COMPENSATING_SENDER (credit fails → unfreeze)
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeFailed})
	tx.SagaState = saga.StateCompensatingSender

	// 4. COMPENSATING_SENDER → BLOCKED (unfreeze fails)
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeFailed})

	require.Len(t, calls, 4)

	// Init
	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventFreezeRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "freeze_sender")

	// Credit
	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventCreditRequest)
	assertSteps(t, req2.Steps, "credit_recipient")

	// Unfreeze (compensation)
	req3 := calls[2].(*models.AdvanceRequest)
	assertEvents(t, req3.Events, models.EventUnfreezeRequest)
	assertSteps(t, req3.Steps, "unfreeze_sender")
	assert.Equal(t, saga.StateCompensatingSender, req3.UpdateSagaStateRequest.SagaState)

	// Blocked
	req4 := calls[3].(*models.AdvanceRequest)
	assertEvents(t, req4.Events, models.EventBlockRequest, models.EventBlockRequest, models.EventTransactionBlocked)
	assertSteps(t, req4.Steps, "block_sender", "block_recipient")
	assert.Equal(t, saga.StateBlocked, req4.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusBlocked, *req4.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_OutboundRemittance_HappyPath(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeOutboundRemittance)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Times(3)

	// INIT → SENDER_FREEZING
	orch.ApplyWithFirstStep(context.Background(), tx, nil, saga.Apply{Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateSenderFreezing

	// SENDER_FREEZING → EXTERNAL_PAYMENT
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateExternalPayment

	// EXTERNAL_PAYMENT → SENDER_CAPTURING
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateSenderCapturing

	// SENDER_CAPTURING → COMPLETED
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})

	require.Len(t, calls, 4)

	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventFreezeRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "freeze_sender")

	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventPaymentRequest)
	assertSteps(t, req2.Steps, "outbound_payment")
	assert.Equal(t, saga.StateExternalPayment, req2.UpdateSagaStateRequest.SagaState)

	req3 := calls[2].(*models.AdvanceRequest)
	assertEvents(t, req3.Events, models.EventCaptureRequest)
	assertSteps(t, req3.Steps, "capture_sender")
	assert.Equal(t, saga.StateSenderCapturing, req3.UpdateSagaStateRequest.SagaState)

	req4 := calls[3].(*models.AdvanceRequest)
	assertEvents(t, req4.Events, models.EventTransactionCompleted)
	assert.Empty(t, req4.Steps)
	assert.Equal(t, saga.StateCompleted, req4.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCompleted, *req4.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_InboundRemittance_HappyPath(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeInboundRemittance)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	// INIT → CREDITING_RECIPIENT
	orch.ApplyWithFirstStep(context.Background(), tx, nil, saga.Apply{Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateCreditingRecipient

	// CREDITING_RECIPIENT → COMPLETED
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})

	require.Len(t, calls, 2)

	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventCreditRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "credit_recipient")

	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventTransactionCompleted)
	assert.Empty(t, req2.Steps)
	assert.Equal(t, saga.StateCompleted, req2.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCompleted, *req2.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_TopUpRequest_HappyPath(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeTopUpRequest)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	// INIT → REQUESTING_TOP_UP
	orch.ApplyWithFirstStep(context.Background(), tx, nil, saga.Apply{Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateRequestingTopUp

	// REQUESTING_TOP_UP → COMPLETED
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})

	require.Len(t, calls, 2)

	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventPaymentRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "topup_request")

	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventTransactionCompleted)
	assert.Empty(t, req2.Steps)
	assert.Equal(t, saga.StateCompleted, req2.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCompleted, *req2.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_RefundInternal_HappyPath(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeRefund)
	parentID := uuid.New()
	tx.ParentTransactionID = &parentID
	parent := newParentTransaction(models.TransactionTypeOnUsTransfer)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Times(2)

	// INIT → CREDITING_SENDER
	orch.ApplyWithFirstStep(context.Background(), tx, parent, saga.Apply{Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateCreditingSender

	// CREDITING_SENDER → DEBITING_RECIPIENT
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateDebitingRecipient

	// DEBITING_RECIPIENT → COMPLETED
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})

	require.Len(t, calls, 3)

	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventCreditRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "credit_sender")

	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventDebitRequest)
	assertSteps(t, req2.Steps, "debit_recipient")
	assert.Equal(t, saga.StateDebitingRecipient, req2.UpdateSagaStateRequest.SagaState)

	req3 := calls[2].(*models.AdvanceRequest)
	assertEvents(t, req3.Events, models.EventTransactionCompleted)
	assert.Empty(t, req3.Steps)
	assert.Equal(t, saga.StateCompleted, req3.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCompleted, *req3.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_RefundOutbound_HappyPath(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeRefund)
	parentID := uuid.New()
	tx.ParentTransactionID = &parentID
	parent := newParentTransaction(models.TransactionTypeOutboundRemittance)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	// INIT → CREDITING_SENDER
	orch.ApplyWithFirstStep(context.Background(), tx, parent, saga.Apply{Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateCreditingSender

	// CREDITING_SENDER → COMPLETED
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})

	require.Len(t, calls, 2)

	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventCreditRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "credit_sender")

	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventTransactionCompleted)
	assert.Empty(t, req2.Steps)
	assert.Equal(t, saga.StateCompleted, req2.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCompleted, *req2.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_RefundInbound_HappyPath(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeRefund)
	parentID := uuid.New()
	tx.ParentTransactionID = &parentID
	parent := newParentTransaction(models.TransactionTypeInboundRemittance)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Times(2)

	// INIT → DEBITING_RECIPIENT
	orch.ApplyWithFirstStep(context.Background(), tx, parent, saga.Apply{Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateDebitingRecipient

	// DEBITING_RECIPIENT → EXTERNAL_PAYMENT
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateExternalPayment

	// EXTERNAL_PAYMENT → COMPLETED
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})

	require.Len(t, calls, 3)

	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventDebitRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "debit_recipient")

	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventPaymentRequest)
	assertSteps(t, req2.Steps, "refund_payment")
	assert.Equal(t, saga.StateExternalPayment, req2.UpdateSagaStateRequest.SagaState)

	req3 := calls[2].(*models.AdvanceRequest)
	assertEvents(t, req3.Events, models.EventTransactionCompleted)
	assert.Empty(t, req3.Steps)
	assert.Equal(t, saga.StateCompleted, req3.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCompleted, *req3.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_ChargebackInternal_HappyPath(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeChargeback)
	parentID := uuid.New()
	tx.ParentTransactionID = &parentID
	parent := newParentTransaction(models.TransactionTypeOnUsTransfer)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Times(2)

	// INIT → DEBITING_RECIPIENT
	orch.ApplyWithFirstStep(context.Background(), tx, parent, saga.Apply{Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateDebitingRecipient

	// DEBITING_RECIPIENT → CREDITING_SENDER
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateCreditingSender

	// CREDITING_SENDER → COMPLETED
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})

	require.Len(t, calls, 3)

	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventDebitRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "debit_recipient")

	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventCreditRequest)
	assertSteps(t, req2.Steps, "credit_sender")
	assert.Equal(t, saga.StateCreditingSender, req2.UpdateSagaStateRequest.SagaState)

	req3 := calls[2].(*models.AdvanceRequest)
	assertEvents(t, req3.Events, models.EventTransactionCompleted)
	assert.Empty(t, req3.Steps)
	assert.Equal(t, saga.StateCompleted, req3.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCompleted, *req3.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_ChargebackOutbound_HappyPath(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeChargeback)
	parentID := uuid.New()
	tx.ParentTransactionID = &parentID
	parent := newParentTransaction(models.TransactionTypeOutboundRemittance)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	// INIT → CREDITING_SENDER
	orch.ApplyWithFirstStep(context.Background(), tx, parent, saga.Apply{Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateCreditingSender

	// CREDITING_SENDER → COMPLETED
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})

	require.Len(t, calls, 2)

	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventCreditRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "credit_sender")

	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventTransactionCompleted)
	assert.Empty(t, req2.Steps)
	assert.Equal(t, saga.StateCompleted, req2.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCompleted, *req2.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_ChargebackInbound_HappyPath(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeChargeback)
	parentID := uuid.New()
	tx.ParentTransactionID = &parentID
	parent := newParentTransaction(models.TransactionTypeInboundRemittance)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Times(2)

	// INIT → DEBITING_RECIPIENT
	orch.ApplyWithFirstStep(context.Background(), tx, parent, saga.Apply{Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateDebitingRecipient

	// DEBITING_RECIPIENT → EXTERNAL_PAYMENT
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateExternalPayment

	// EXTERNAL_PAYMENT → COMPLETED
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})

	require.Len(t, calls, 3)

	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventDebitRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "debit_recipient")

	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventPaymentRequest)
	assertSteps(t, req2.Steps, "chargeback_payment")
	assert.Equal(t, saga.StateExternalPayment, req2.UpdateSagaStateRequest.SagaState)

	req3 := calls[2].(*models.AdvanceRequest)
	assertEvents(t, req3.Events, models.EventTransactionCompleted)
	assert.Empty(t, req3.Steps)
	assert.Equal(t, saga.StateCompleted, req3.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCompleted, *req3.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_RefundOnUsFX_HappyPath(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeRefund)
	parentID := uuid.New()
	tx.ParentTransactionID = &parentID
	parent := newParentTransaction(models.TransactionTypeOnUsFXTransfer)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Times(4)

	// 1. INIT → DEBITING_RECIPIENT
	orch.ApplyWithFirstStep(context.Background(), tx, parent, saga.Apply{Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateDebitingRecipient

	fxDealID := uuid.New()
	tx.FXDealID = &fxDealID

	// 2. DEBITING_RECIPIENT → ACTIVATING_FX
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateActivatingFX

	// 3. ACTIVATING_FX → CREDITING_SENDER
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateCreditingSender

	// 4. CREDITING_SENDER → COMMITTING_FX
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateCommittingFX

	// 5. COMMITTING_FX → COMPLETED
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})

	require.Len(t, calls, 5)

	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventDebitRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "debit_recipient")
	assert.Equal(t, saga.StateDebitingRecipient, req1.UpdateSagaStateRequest.SagaState)

	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventFXActivateRequest)
	assertSteps(t, req2.Steps, "activate_fx")
	assert.Equal(t, saga.StateActivatingFX, req2.UpdateSagaStateRequest.SagaState)

	req3 := calls[2].(*models.AdvanceRequest)
	assertEvents(t, req3.Events, models.EventCreditRequest)
	assertSteps(t, req3.Steps, "credit_sender")
	assert.Equal(t, saga.StateCreditingSender, req3.UpdateSagaStateRequest.SagaState)

	req4 := calls[3].(*models.AdvanceRequest)
	assertEvents(t, req4.Events, models.EventFXCommitRequest)
	assertSteps(t, req4.Steps, "commit_fx")
	assert.Equal(t, saga.StateCommittingFX, req4.UpdateSagaStateRequest.SagaState)

	req5 := calls[4].(*models.AdvanceRequest)
	assertEvents(t, req5.Events, models.EventTransactionCompleted)
	assert.Empty(t, req5.Steps)
	assert.Equal(t, saga.StateCompleted, req5.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCompleted, *req5.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_ChargebackOnUsFX_HappyPath(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeChargeback)
	parentID := uuid.New()
	tx.ParentTransactionID = &parentID
	parent := newParentTransaction(models.TransactionTypeOnUsFXTransfer)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Times(4)

	// 1. INIT → DEBITING_RECIPIENT
	orch.ApplyWithFirstStep(context.Background(), tx, parent, saga.Apply{Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateDebitingRecipient

	fxDealID := uuid.New()
	tx.FXDealID = &fxDealID

	// 2. DEBITING_RECIPIENT → ACTIVATING_FX
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateActivatingFX

	// 3. ACTIVATING_FX → CREDITING_SENDER
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateCreditingSender

	// 4. CREDITING_SENDER → COMMITTING_FX
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateCommittingFX

	// 5. COMMITTING_FX → COMPLETED
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})

	require.Len(t, calls, 5)

	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventDebitRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "debit_recipient")
	assert.Equal(t, saga.StateDebitingRecipient, req1.UpdateSagaStateRequest.SagaState)

	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventFXActivateRequest)
	assertSteps(t, req2.Steps, "activate_fx")
	assert.Equal(t, saga.StateActivatingFX, req2.UpdateSagaStateRequest.SagaState)

	req3 := calls[2].(*models.AdvanceRequest)
	assertEvents(t, req3.Events, models.EventCreditRequest)
	assertSteps(t, req3.Steps, "credit_sender")
	assert.Equal(t, saga.StateCreditingSender, req3.UpdateSagaStateRequest.SagaState)

	req4 := calls[3].(*models.AdvanceRequest)
	assertEvents(t, req4.Events, models.EventFXCommitRequest)
	assertSteps(t, req4.Steps, "commit_fx")
	assert.Equal(t, saga.StateCommittingFX, req4.UpdateSagaStateRequest.SagaState)

	req5 := calls[4].(*models.AdvanceRequest)
	assertEvents(t, req5.Events, models.EventTransactionCompleted)
	assert.Empty(t, req5.Steps)
	assert.Equal(t, saga.StateCompleted, req5.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCompleted, *req5.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}

func TestOrchestrator_AdjustmentDebit_HappyPath(t *testing.T) {
	mockSM := new(MockSagaManager)
	defs := saga.Register()
	orch := saga.NewOrchestrator(mockSM, defs)

	tx := newTestTransaction(models.TransactionTypeAdjustmentDebit)

	var calls []any

	mockSM.On("AdvanceWithFirstStep", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	mockSM.On("Advance", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		calls = append(calls, args.Get(1))
	}).Once()

	// INIT → DEBITING_RECIPIENT
	orch.ApplyWithFirstStep(context.Background(), tx, nil, saga.Apply{Outcome: saga.OutcomeSuccess})
	tx.SagaState = saga.StateDebitingRecipient

	// DEBITING_RECIPIENT → COMPLETED
	tx.UpdatedAt = time.Now()
	orch.Apply(context.Background(), tx, saga.Apply{TransactionID: tx.ID, Outcome: saga.OutcomeSuccess})

	require.Len(t, calls, 2)

	req1 := calls[0].(*models.AdvanceWithFirstStepRequest)
	assertEvents(t, req1.Events, models.EventDebitRequest, models.EventTransactionCreated)
	assertSteps(t, req1.Steps, "debit_recipient")

	req2 := calls[1].(*models.AdvanceRequest)
	assertEvents(t, req2.Events, models.EventTransactionCompleted)
	assert.Empty(t, req2.Steps)
	assert.Equal(t, saga.StateCompleted, req2.UpdateSagaStateRequest.SagaState)
	assert.Equal(t, models.TransactionStatusCompleted, *req2.UpdateSagaStateRequest.Status)

	mockSM.AssertExpectations(t)
}
