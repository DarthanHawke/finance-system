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

type Orchestrator struct {
	sagaManager SagaManager
	definition  map[string]Definition
}

type SagaManager interface {
	Advance(ctx context.Context, req *models.AdvanceRequest) error
	AdvanceWithFirstStep(ctx context.Context, req *models.AdvanceWithFirstStepRequest) error
}

func NewOrchestrator(
	sagaManager SagaManager,
	definition map[string]Definition,
) *Orchestrator {
	return &Orchestrator{
		sagaManager: sagaManager,
		definition:  definition,
	}
}

func (o *Orchestrator) Apply(ctx context.Context, transaction *models.Transaction, apply Apply) error {
	const op = "saga.Apply"

	now := time.Now()
	transaction.CreatedAt = now
	transaction.UpdatedAt = now

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

	events, steps, err := o.CreateEventAndSteps(transition, transaction, apply.TraceID, apply.SpanID, now)
	if err != nil {
		return err
	}

	updateState := &models.UpdateSagaStateRequest{
		TransactionID: transaction.ID,
		ExpectedFrom:  transition.From,
		SagaState:     transition.NewState,
		Status:        &transition.NewStatus,
		UpdatedAt:     now,
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

func (o *Orchestrator) ApplyWithFirstStep(
	ctx context.Context,
	transaction *models.Transaction,
	transactionParent *models.Transaction,
	apply Apply,
) error {
	const op = "saga.ApplyWithFirstStep"
	now := time.Now()
	var parentType *string
	if transactionParent != nil {
		parentType = &transactionParent.Type
	}

	sagaType, err := o.ResolveSagaType(transaction.Type, parentType)
	if err != nil {
		return &apperr.WrappedError{Op: op, Err: err}
	}
	transaction.CreatedAt = now
	transaction.UpdatedAt = now
	transaction.SagaType = sagaType

	_, ok := o.Definition(sagaType)
	if !ok {
		return &apperr.TerminalError{Op: op, Context: "no saga definition for " + sagaType}
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

	events, steps, err := o.CreateEventAndSteps(transition, transaction, apply.TraceID, apply.SpanID, now)
	if err != nil {
		return err
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
		UpdatedAt:     transaction.UpdatedAt,
	}

	advanceRequest := &models.AdvanceWithFirstStepRequest{
		InsertTransactionRequest: models.InsertTransactionRequest{Transaction: transaction},
		InsertPartiesRequest:     models.InsertPartiesRequest{Parties: transaction.Parties},
		Events:                   events,
		Steps:                    steps,
		UpdateSagaStateRequest:   updateState,
	}

	if err := o.sagaManager.AdvanceWithFirstStep(ctx, advanceRequest); err != nil {
		return err
	}

	return nil
}

func (o *Orchestrator) ResolveSagaType(transactionType string, parentType *string) (string, error) {
	const op = "saga.ResolveSagaType"

	switch transactionType {
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

func (o *Orchestrator) CreateEventAndSteps(
	transition Transition,
	transaction *models.Transaction,
	traceID string,
	spanID string,
	now time.Time,
) ([]models.InsertEventRequest,
	[]models.InsertSagaStepRequest,
	error,
) {
	var events []models.InsertEventRequest
	var steps []models.InsertSagaStepRequest

	for _, cmd := range transition.Commands {
		payload, err := cmd.BuildPayload(transaction)
		if err != nil {
			return nil, nil, err
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, err
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
				TraceID:       traceID,
				SpanID:        spanID,
				CreatedAt:     now,
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
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		}
		steps = append(steps, step)
	}

	for _, notif := range transition.Notifications {
		payload, err := notif.BuildPayload(transaction)
		if err != nil {
			return nil, nil, err
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, err
		}

		event := models.InsertEventRequest{
			Event: &models.Event{
				ID:            uuid.New(),
				TransactionID: transaction.ID,
				StepName:      notif.StepName,
				PartitionKey:  transaction.ID.String(),
				Type:          notif.EventType,
				Status:        models.EventStatusPending,
				Source:        models.Source,
				TraceID:       traceID,
				SpanID:        spanID,
				CreatedAt:     now,
				Payload:       payloadBytes,
			},
		}
		events = append(events, event)
	}

	return events, steps, nil
}

func (o *Orchestrator) Definition(sagaType string) (Definition, bool) {
	def, ok := o.definition[sagaType]
	return def, ok
}
