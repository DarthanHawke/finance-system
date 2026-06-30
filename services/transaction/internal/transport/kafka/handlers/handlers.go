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
	Apply(ctx context.Context, transaction *models.Transaction, apply saga.Apply) error
	ApplyWithFirstStep(
		ctx context.Context,
		transaction *models.Transaction,
		transactionParent *models.Transaction,
		apply saga.Apply,
	) error
}

type TransactionManager interface {
	GetTransaction(ctx context.Context, req *models.GetTransactionRequest) (models.GetTransactionResponse, error)
}

func New(
	orchestrator *saga.Orchestrator,
	transactionManager TransactionManager,
) *Handlers {
	return &Handlers{
		orchestratorManager: orchestrator,
		transactionManager:  transactionManager,
	}
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

	transaction, err := h.createTransaciton(&transactionPayload)
	if err != nil {
		return err
	}

	var transactionParent *models.Transaction
	if transaction.ParentTransactionID != uuid.Nil {
		parentResp, err := h.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{
			ID: transaction.ParentTransactionID,
		})
		if err != nil {
			return err
		}
		transactionParent = parentResp.Transaction

		// Проверяем, что родительская транзакция завершена
		if transactionParent.Status != models.TransactionStatusCompleted {
			return &apperr.WrappedError{
				Op:  op,
				Err: fmt.Errorf("parent transaction %s is not completed", transactionParent.ID),
			}
		}
	}

	return h.orchestratorManager.ApplyWithFirstStep(
		ctx,
		transaction,
		transactionParent,
		saga.Apply{
			Outcome: saga.OutcomeSuccess,
			TraceID: event.TraceID,
			SpanID:  event.SpanID,
		},
	)
}

func (h *Handlers) HandleFreezeResponse(ctx context.Context, event *models.Event) error {
	outcome, err := outcomeFromResponse(event)
	if err != nil {
		return err
	}
	resp, err := h.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: event.TransactionID})
	if err != nil {
		return err
	}
	transaction := resp.Transaction
	transaction.UpdatedAt = time.Now()

	return h.orchestratorManager.Apply(ctx, transaction, saga.Apply{
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
	resp, err := h.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: event.TransactionID})
	if err != nil {
		return err
	}
	transaction := resp.Transaction
	transaction.UpdatedAt = time.Now()

	return h.orchestratorManager.Apply(ctx, transaction, saga.Apply{
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
	resp, err := h.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: event.TransactionID})
	if err != nil {
		return err
	}
	transaction := resp.Transaction
	transaction.UpdatedAt = time.Now()

	return h.orchestratorManager.Apply(ctx, transaction, saga.Apply{
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
	resp, err := h.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: event.TransactionID})
	if err != nil {
		return err
	}
	transaction := resp.Transaction
	transaction.UpdatedAt = time.Now()

	return h.orchestratorManager.Apply(ctx, transaction, saga.Apply{
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
	resp, err := h.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: event.TransactionID})
	if err != nil {
		return err
	}
	transaction := resp.Transaction
	transaction.UpdatedAt = time.Now()

	return h.orchestratorManager.Apply(ctx, transaction, saga.Apply{
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
	resp, err := h.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: event.TransactionID})
	if err != nil {
		return err
	}
	transaction := resp.Transaction
	transaction.UpdatedAt = time.Now()

	return h.orchestratorManager.Apply(ctx, transaction, saga.Apply{
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
	resp, err := h.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: event.TransactionID})
	if err != nil {
		return err
	}
	transaction := resp.Transaction
	transaction.UpdatedAt = time.Now()

	return h.orchestratorManager.Apply(ctx, transaction, saga.Apply{
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
	resp, err := h.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: event.TransactionID})
	if err != nil {
		return err
	}
	transaction := resp.Transaction
	transaction.UpdatedAt = time.Now()

	return h.orchestratorManager.Apply(ctx, transaction, saga.Apply{
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

func (h *Handlers) createTransaciton(
	transactionPayload *models.TransactionPayload,
) (*models.Transaction, error) {
	const op = "handle.createTransaciton"

	transaction := &models.Transaction{
		ID:             uuid.New(),
		IdempotencyKey: transactionPayload.IdempotencyKey,
		Type:           transactionPayload.Type,
		Status:         models.TransactionStatusProcessing,
		Amount:         transactionPayload.Amount,
		Currency:       transactionPayload.Currency,
		Description:    transactionPayload.Description,
	}

	transaction.ParentTransactionID = uuid.Nil
	if transactionPayload.ParentTransactionID != "" {
		parentID, err := uuid.Parse(transactionPayload.ParentTransactionID)
		if err != nil {
			return nil, &apperr.WrappedError{Op: op, Err: apperr.ErrInvalidParentTransactionID}
		}
		transaction.ParentTransactionID = parentID
	}

	transaction.Parties = []models.TransactionParty{
		{
			PartyRole:   models.PartyRoleSender,
			PartyType:   transactionPayload.SenderType(),
			Identifiers: models.CleanMap(transactionPayload.SenderFields()),
		},
		{
			PartyRole:   models.PartyRoleRecipient,
			PartyType:   transactionPayload.RecipientType(),
			Identifiers: models.CleanMap(transactionPayload.RecipientFields()),
		},
	}
	return transaction, nil
}
