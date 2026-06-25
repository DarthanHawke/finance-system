package handler

import (
	"context"
	"encoding/json"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"
	"transaction-service/internal/saga"
)

type Handlers struct {
	orchestratorManager OrchestratorManager
}

type OrchestratorManager interface {
	Apply(ctx context.Context, apply saga.Apply) error
	ApplyWithFirstStep(ctx context.Context, transaction *models.TransactionPayload, apply saga.Apply) error
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

	var transactionPayload models.TransactionPayload
	if err := json.Unmarshal(event.Payload, &transactionPayload); err != nil {
		return &apperr.WrappedError{Op: op, Err: err}
	}

	return h.orchestratorManager.ApplyWithFirstStep(ctx, &transactionPayload, saga.Apply{
		Outcome: saga.OutcomeSuccess,
		TraceID: event.TraceID,
		SpanID:  event.SpanID,
	})
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
