package saga

import (
	"context"
	"encoding/json"
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
		payload, partitionKey, err := cmd.BuildPayload(transaction)
		if err != nil {
			return err
		}
		ev := o.buildEvent(models.BuildEventRequest{
			Transaction:  transaction,
			StepName:     cmd.StepName,
			PartitionKey: partitionKey,
			EventType:    cmd.EventType,
			TraceID:      apply.TraceID,
			SpanID:       apply.SpanID,
			Payload:      payload,
		})
		events = append(events, ev)
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
