package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"

	"github.com/google/uuid"
)

type TransactionManager interface {
	GetTransaction(ctx context.Context, req *models.GetTransactionRequest) (models.GetTransactionResponse, error)
}

type SagaManager interface {
	Advance(ctx context.Context, req *models.AdvanceRequest) error
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

	var events []*models.CreateEventRequest
	for _, cmd := range transition.Commands {
		payload, err := cmd.BuildPayload(transaction)
		if err != nil {
			return err
		}
		event := o.buildEvent(models.BuildEventRequest{
			Transaction:  transaction,
			StepName:     cmd.StepName,
			PartitionKey: transaction.ID.String(),
			EventType:    cmd.EventType,
			TraceID:      apply.TraceID,
			SpanID:       apply.SpanID,
			Payload:      payload,
		})
		events = append(events, event)
	}

	advanceRequest := &models.AdvanceRequest{
		TransactionID: transaction.ID,
		ExpectedFrom:  transition.From,
		NewState:      transition.NewState,
	}
	if transition.NewStatus != "" {
		advanceRequest.NewStatus = &transition.NewStatus
	}

	if len(events) > 0 {
		advanceRequest.Event = events[0]
		if len(events) > 1 {
			advanceRequest.ExtraEvents = events[1:]
		}
	}

	if len(transition.Commands) > 0 {
		advanceRequest.Step = &models.CreateSagaStepRequest{
			ID:            uuid.New(),
			TransactionID: transaction.ID,
			EventID:       events[0].ID,
			StepName:      transition.Commands[0].StepName,
			StepKind:      transition.Commands[0].StepKind,
			Status:        models.StepStatusStarted,
		}
	}

	if err := o.sagaManager.Advance(ctx, advanceRequest); err != nil {
		return err
	}

	return nil
}

func (o *Orchestrator) buildEvent(
	req models.BuildEventRequest,
) *models.CreateEventRequest {
	payloadBytes, _ := json.Marshal(req.Payload)
	return &models.CreateEventRequest{
		Event: &models.Event{
			ID:            uuid.New(),
			TransactionID: req.Transaction.ID,
			StepName:      req.StepName,
			PartitionKey:  req.PartitionKey,
			Type:          req.EventType,
			Status:        models.EventStatusPending,
			Source:        models.Source,
			TraceID:       req.TraceID,
			SpanID:        req.SpanID,
			Payload:       payloadBytes,
		},
	}
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

	case models.TransactionTypeAdjustment:
		return models.SagaAdjustmentCredit, nil

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
