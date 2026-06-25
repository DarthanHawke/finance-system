package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"

	"github.com/google/uuid"
)

type TransactionManager interface {
	GetTransaction(ctx context.Context, req *models.GetTransactionRequest) (models.GetTransactionResponse, error)
}

type SagaManager interface {
	Advance(ctx context.Context, req *models.AdvanceRequest) error
	AdvanceWithFirstStep(ctx context.Context, req *models.AdvanceWithFirstStepRequest) error
}

type Orchestrator struct {
	transactionManager TransactionManager
	sagaManager        SagaManager
	definition         map[string]Definition
}

func NewOrchestrator(
	transactionManager TransactionManager,
	sagaManager SagaManager,
	definition map[string]Definition,
) *Orchestrator {
	return &Orchestrator{
		transactionManager: transactionManager,
		sagaManager:        sagaManager,
		definition:         definition,
	}
}

func (o *Orchestrator) Apply(ctx context.Context, apply Apply) error {
	const op = "saga.Apply"

	resp, err := o.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{ID: apply.TransactionID})
	if err != nil {
		return err
	}
	transaction := resp.Transaction

	def, ok := o.definition[transaction.SagaType]
	if !ok {
		return &apperr.TerminalError{Op: op, Context: "unknown saga type: " + transaction.SagaType}
	}

	byOutcome, ok := def.Transitions[transaction.SagaState]
	if !ok {
		return nil
	}

	transition, ok := byOutcome[apply.Outcome]
	if !ok {
		return nil
	}

	var events []models.InsertEventRequest
	var steps []models.InsertSagaStepRequest

	for _, cmd := range transition.Commands {
		payload, err := cmd.BuildPayload(transaction)
		if err != nil {
			return err
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		event := models.InsertEventRequest{
			Event: &models.Event{
				ID:            uuid.New(),
				TransactionID: transaction.ID,
				StepName:      cmd.StepName,
				PartitionKey:  transaction.ID.String(),
				Type:          cmd.EventType,
				Status:        models.EventStatusPending,
				Source:        models.Source,
				TraceID:       apply.TraceID,
				SpanID:        apply.SpanID,
				Payload:       payloadBytes,
			},
		}
		events = append(events, event)

		step := models.InsertSagaStepRequest{
			SagaStep: &models.SagaStep{
				ID:            uuid.New(),
				TransactionID: transaction.ID,
				EventID:       event.Event.ID,
				StepName:      cmd.StepName,
				StepKind:      cmd.StepKind,
				Status:        models.StepStatusStarted,
			},
		}
		steps = append(steps, step)
	}

	updateState := &models.UpdateSagaStateRequest{
		TransactionID: transaction.ID,
		ExpectedFrom:  transition.From,
		SagaState:     transition.NewState,
		Status:        &transition.NewStatus,
	}

	advanceRequest := &models.AdvanceRequest{
		Events:                 events,
		Steps:                  steps,
		UpdateSagaStateRequest: updateState,
	}

	if err := o.sagaManager.Advance(ctx, advanceRequest); err != nil {
		return err
	}

	return nil
}

func (o *Orchestrator) ApplyWithFirstStep(ctx context.Context, transactionPayload *models.TransactionPayload, apply Apply) error {
	const op = "saga.ApplyWithFirstStep"

	now := time.Now()
	transaction := &models.Transaction{
		ID:             uuid.New(),
		IdempotencyKey: transactionPayload.IdempotencyKey,
		Type:           transactionPayload.Type,
		Status:         models.TransactionStatusProcessing,
		SagaState:      StateInit,
		Amount:         transactionPayload.Amount,
		Currency:       transactionPayload.Currency,
		Description:    transactionPayload.Description,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	var parentType *string
	if transactionPayload.ParentTransactionID != "" {
		parentID, err := uuid.Parse(transactionPayload.ParentTransactionID)
		if err != nil {
			return &apperr.WrappedError{Op: op, Err: apperr.ErrInvalidParentTransactionID}
		}
		transaction.ParentTransactionID = parentID

		parentResp, err := o.transactionManager.GetTransaction(ctx, &models.GetTransactionRequest{
			ID: parentID,
		})
		if err != nil {
			return err
		}
		parentTransaction := parentResp.Transaction

		// Проверяем, что родительская транзакция завершена
		if parentTransaction.Status != models.TransactionStatusCompleted {
			return &apperr.WrappedError{
				Op:  op,
				Err: fmt.Errorf("parent transaction %s is not completed", parentTransaction.ID),
			}
		}
		parentType = &parentTransaction.Type
	}

	sagaType, err := o.ResolveSagaType(transaction.Type, parentType)
	if err != nil {
		return &apperr.WrappedError{Op: op, Err: err}
	}
	transaction.SagaType = sagaType

	_, ok := o.Definition(sagaType)
	if !ok {
		return &apperr.TerminalError{Op: op, Context: "no saga definition for " + sagaType}
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

	def, ok := o.definition[transaction.SagaType]
	if !ok {
		return &apperr.TerminalError{Op: op, Context: "unknown saga type: " + transaction.SagaType}
	}

	startTransitions, ok := def.Transitions[def.InitState]
	if !ok {
		return &apperr.TerminalError{Op: op, Context: "no init transitions"}
	}
	transition, ok := startTransitions[OutcomeSuccess]
	if !ok || len(transition.Commands) == 0 {
		return &apperr.TerminalError{Op: op, Context: "no start command"}
	}

	cmd := transition.Commands[0]
	payload, err := cmd.BuildPayload(transaction)
	if err != nil {
		return err
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	event := models.InsertEventRequest{
		Event: &models.Event{
			ID:            uuid.New(),
			TransactionID: transaction.ID,
			StepName:      cmd.StepName,
			PartitionKey:  transaction.ID.String(),
			Type:          cmd.EventType,
			Status:        models.EventStatusPending,
			Source:        models.Source,
			TraceID:       apply.TraceID,
			SpanID:        apply.SpanID,
			Payload:       payloadBytes,
		},
	}
	step := models.InsertSagaStepRequest{
		SagaStep: &models.SagaStep{
			ID:            uuid.New(),
			TransactionID: transaction.ID,
			EventID:       event.Event.ID,
			StepName:      cmd.StepName,
			StepKind:      cmd.StepKind,
			Status:        models.StepStatusStarted,
		},
	}
	var newStatus *string
	if transition.NewStatus != "" {
		newStatus = &transition.NewStatus
	}
	updateState := &models.UpdateSagaStateRequest{
		TransactionID: transaction.ID,
		ExpectedFrom:  transition.From,
		SagaState:     transition.NewState,
		Status:        newStatus,
	}

	advanceRequest := &models.AdvanceWithFirstStepRequest{
		InsertTransactionRequest: models.InsertTransactionRequest{Transaction: transaction},
		InsertPartiesRequest:     models.InsertPartiesRequest{Parties: transaction.Parties},
		Events:                   []models.InsertEventRequest{event},
		Steps:                    []models.InsertSagaStepRequest{step},
		UpdateSagaStateRequest:   updateState,
	}

	if err := o.sagaManager.AdvanceWithFirstStep(ctx, advanceRequest); err != nil {
		return err
	}

	return nil
}

func (o *Orchestrator) ResolveSagaType(txnType string, parentType *string) (string, error) {
	const op = "saga.ResolveSagaType"

	switch txnType {
	case models.TransactionTypeOnUsTransfer,
		models.TransactionTypeDirectDebit:
		return models.SagaOnUsTransfer, nil

	case models.TransactionTypeOutboundRemittance,
		models.TransactionTypeOutboundDirectDebit:
		return models.SagaOutboundRemittance, nil

	case models.TransactionTypeInboundRemittance,
		models.TransactionTypeInboundDirectDebit:
		return models.SagaInboundRemittance, nil

	case models.TransactionTypeTopUpRequest:
		return models.SagaTopUpRequest, nil

	case models.TransactionTypeRefund:
		if parentType == nil {
			return "", &apperr.WrappedError{
				Op:  op,
				Err: apperr.ErrInvalidParentTransactionID,
			}
		}
		switch *parentType {
		case models.TransactionTypeOnUsTransfer,
			models.TransactionTypeDirectDebit:
			return models.SagaRefundInternal, nil
		case models.TransactionTypeOutboundRemittance,
			models.TransactionTypeOutboundDirectDebit:
			return models.SagaRefundOutbound, nil
		case models.TransactionTypeInboundRemittance,
			models.TransactionTypeInboundDirectDebit:
			return models.SagaRefundInbound, nil
		default:
			return "", &apperr.WrappedError{
				Op:  op,
				Err: fmt.Errorf("refund not supported for parent type: %s", *parentType),
			}
		}

	case models.TransactionTypeChargeback:
		if parentType == nil {
			return "", &apperr.WrappedError{
				Op:  op,
				Err: apperr.ErrInvalidParentTransactionID,
			}
		}
		switch *parentType {
		case models.TransactionTypeOnUsTransfer,
			models.TransactionTypeDirectDebit:
			return models.SagaChargebackInternal, nil
		case models.TransactionTypeOutboundRemittance,
			models.TransactionTypeOutboundDirectDebit:
			return models.SagaChargebackOutbound, nil
		case models.TransactionTypeInboundRemittance,
			models.TransactionTypeInboundDirectDebit:
			return models.SagaChargebackInbound, nil
		default:
			return "", &apperr.WrappedError{
				Op:  op,
				Err: fmt.Errorf("chargeback not supported for parent type: %s", *parentType),
			}
		}

	case models.TransactionTypeAdjustmentCredit:
		return models.SagaAdjustmentCredit, nil

	case models.TransactionTypeAdjustmentDebit:
		return models.SagaAdjustmentDebit, nil

	default:
		return "", &apperr.WrappedError{
			Op:  op,
			Err: apperr.ErrUnknownTransactionType,
		}
	}
}

func (o *Orchestrator) Definition(sagaType string) (Definition, bool) {
	def, ok := o.definition[sagaType]
	return def, ok
}
