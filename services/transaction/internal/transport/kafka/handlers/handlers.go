package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"
	"transaction-service/internal/saga"

	"github.com/google/uuid"
)

type Handlers struct {
	orchestratorManager OrchestratorManager
	transactionManager  TransactionManager
}

type OrchestratorManager interface {
	Definition(sagaType string) (saga.Definition, bool)
	ResolveSagaType(txnType string, parentType *string) (string, error)
	Apply(ctx context.Context, apply saga.Apply) error
}

type TransactionManager interface {
	CreateTransaction(
		ctx context.Context,
		req *models.CreateTransactionRequest,
		event *models.CreateEventRequest,
		step *models.CreateSagaStepRequest,
	) error
	GetTransaction(ctx context.Context, req *models.GetTransactionRequest) (models.GetTransactionResponse, error)
}

func New(orchestrator *saga.Orchestrator) *Handlers {
	return &Handlers{orchestratorManager: orchestrator}
}

func outcomeFromResponse(event *models.Event) (int, error) {
	var payload models.ResponsePayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return 0, &apperr.TerminalError{
			Op:      "handler.outcome",
			Context: "bad response payload",
			Err:     err,
		}
	}
	if payload.Success {
		return saga.OutcomeSuccess, nil
	}
	return saga.OutcomeFailed, nil
}

func (h *Handlers) HandleTransactionCreate(ctx context.Context, event *models.Event) error {
	const op = "handlers.HandleTransactionCreate"

	var p models.TransactionPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return &apperr.WrappedError{Op: op, Err: err}
	}

	now := time.Now()
	tx := &models.Transaction{
		ID:             uuid.New(),
		IdempotencyKey: p.IdempotencyKey,
		Type:           p.Type,
		Status:         models.TransactionStatusProcessing,
		Amount:         p.Amount,
		Currency:       p.Currency,
		Description:    p.Description,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	pTx := &models.Transaction{}
	if p.ParentTransactionID != "" {
		parentID, err := uuid.Parse(p.ParentTransactionID)
		if err != nil {
			return &apperr.WrappedError{Op: op, Err: apperr.ErrInvalidParentTransactionID}
		}
		tx.ParentTransactionID = parentID
		parentTransaction, err := h.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{
			ID: parentID,
		})
		pTx = parentTransaction.Transaction

	}

	sagaType, err := h.orchestratorManager.ResolveSagaType(tx.Type, &pTx.Type)
	if err != nil {
		return &apperr.WrappedError{Op: op, Err: err}
	}

	def, ok := h.orchestratorManager.Definition(sagaType)
	if !ok {
		return &apperr.TerminalError{Op: op, Context: "no saga definition for " + sagaType}
	}

	tx.Parties = []models.TransactionParty{
		{
			PartyRole:   models.PartyRoleSender,
			PartyType:   p.SenderType(),
			Identifiers: cleanMap(p.SenderFields()),
		},
		{
			PartyRole:   models.PartyRoleRecipient,
			PartyType:   p.RecipientType(),
			Identifiers: cleanMap(p.RecipientFields()),
		},
	}

	startTransitions, ok := def.Transitions[def.InitState]
	if !ok {
		return &apperr.TerminalError{Op: op, Context: "no init transitions for " + sagaType}
	}
	startTr, ok := startTransitions[saga.OutcomeSuccess]
	if !ok || len(startTr.Commands) == 0 {
		return &apperr.TerminalError{Op: op, Context: "no start command for " + sagaType}
	}

	tx.SagaState = startTr.NewState

	cmd := startTr.Commands[0]
	payload, err := cmd.BuildPayload(tx)
	if err != nil {
		return &apperr.WrappedError{Op: op, Err: err}
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return &apperr.WrappedError{
			Op:  op,
			Err: fmt.Errorf("marshal payload: %w", err),
		}
	}

	firstEvent := &models.Event{
		ID:            uuid.New(),
		TransactionID: tx.ID,
		PartitionKey:  tx.ID.String(),
		Type:          cmd.EventType,
		Status:        models.EventStatusPending,
		Source:        models.Source,
		TraceID:       event.TraceID,
		SpanID:        event.SpanID,
		Payload:       payloadBytes,
	}

	firstStep := &models.CreateSagaStepRequest{
		ID:            uuid.New(),
		TransactionID: tx.ID,
		EventID:       firstEvent.ID,
		StepName:      cmd.StepName,
		StepKind:      cmd.StepKind,
		Status:        models.StepStatusStarted,
	}

	return h.transactionManager.CreateTransaction(
		ctx,
		&models.CreateTransactionRequest{Transaction: tx},
		&models.CreateEventRequest{Event: firstEvent},
		firstStep,
	)
}

func (h *Handlers) HandleFreezeResponse(ctx context.Context, event *models.Event) error {
	outcome, err := outcomeFromResponse(event)
	if err != nil {
		return err
	}
	return h.orchestratorManager.Apply(ctx, saga.Apply{
		TransactionID: event.TransactionID,
		Outcome:       outcome,
		TraceID:       event.TraceID,
		SpanID:        event.SpanID,
	})
}

func (h *Handlers) HandleDepositResponse(ctx context.Context, event *models.Event) error {
	outcome, err := outcomeFromResponse(event)
	if err != nil {
		return err
	}
	return h.orchestratorManager.Apply(ctx, saga.Apply{
		TransactionID: event.TransactionID,
		Outcome:       outcome,
		TraceID:       event.TraceID,
		SpanID:        event.SpanID,
	})
}

func (h *Handlers) HandleCaptureResponse(ctx context.Context, event *models.Event) error {
	outcome, err := outcomeFromResponse(event)
	if err != nil {
		return err
	}
	return h.orchestratorManager.Apply(ctx, saga.Apply{
		TransactionID: event.TransactionID,
		Outcome:       outcome,
		TraceID:       event.TraceID,
		SpanID:        event.SpanID,
	})
}

func (h *Handlers) HandleCreditResponse(ctx context.Context, event *models.Event) error {
	outcome, err := outcomeFromResponse(event)
	if err != nil {
		return err
	}
	return h.orchestratorManager.Apply(ctx, saga.Apply{
		TransactionID: event.TransactionID,
		Outcome:       outcome,
		TraceID:       event.TraceID,
		SpanID:        event.SpanID,
	})
}

func (h *Handlers) HandleDebitResponse(ctx context.Context, event *models.Event) error {
	outcome, err := outcomeFromResponse(event)
	if err != nil {
		return err
	}
	return h.orchestratorManager.Apply(ctx, saga.Apply{
		TransactionID: event.TransactionID,
		Outcome:       outcome,
		TraceID:       event.TraceID,
		SpanID:        event.SpanID,
	})
}

func (h *Handlers) HandleUnfreezeResponse(ctx context.Context, event *models.Event) error {
	outcome, err := outcomeFromResponse(event)
	if err != nil {
		return err
	}
	return h.orchestratorManager.Apply(ctx, saga.Apply{
		TransactionID: event.TransactionID,
		Outcome:       outcome,
		TraceID:       event.TraceID,
		SpanID:        event.SpanID,
	})
}

func (h *Handlers) HandleExternalPaymentResponse(ctx context.Context, event *models.Event) error {
	outcome, err := outcomeFromResponse(event)
	if err != nil {
		return err
	}
	return h.orchestratorManager.Apply(ctx, saga.Apply{
		TransactionID: event.TransactionID,
		Outcome:       outcome,
		TraceID:       event.TraceID,
		SpanID:        event.SpanID,
	})
}

func (h *Handlers) HandleExternalCommitResponse(ctx context.Context, event *models.Event) error {
	outcome, err := outcomeFromResponse(event)
	if err != nil {
		return err
	}
	return h.orchestratorManager.Apply(ctx, saga.Apply{
		TransactionID: event.TransactionID,
		Outcome:       outcome,
		TraceID:       event.TraceID,
		SpanID:        event.SpanID,
	})
}

func (h *Handlers) HandleExternalRollbackResponse(ctx context.Context, event *models.Event) error {
	outcome, err := outcomeFromResponse(event)
	if err != nil {
		return err
	}
	return h.orchestratorManager.Apply(ctx, saga.Apply{
		TransactionID: event.TransactionID,
		Outcome:       outcome,
		TraceID:       event.TraceID,
		SpanID:        event.SpanID,
	})
}

func (h *Handlers) HandleBlockAccountResponse(ctx context.Context, event *models.Event) error {
	outcome, err := outcomeFromResponse(event)
	if err != nil {
		return err
	}
	if outcome == saga.OutcomeFailed {
		return &apperr.TerminalError{
			Op:      "handler.block",
			Context: "account block failed, tx=" + event.TransactionID.String(),
		}
	}
	return nil
}

func cleanMap(m map[string]string) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		if v != "" {
			result[k] = v
		}
	}
	return result
}
