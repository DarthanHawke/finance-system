// Пакет models содержит различные dto'шки, модельки, константы
package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	StepStatusStarted   string = "STARTED"
	StepStatusCompleted string = "COMPLETED"
	StepStatusFailed    string = "FAILED"
)

const (
	StepKindAction       string = "ACTION"
	StepKindCompensation string = "COMPENSATION"
)

type SagaStep struct {
	ID            uuid.UUID `db:"id" json:"id" validate:"required"`
	TransactionID uuid.UUID `db:"transaction_id" json:"transaction_id" validate:"required"`
	EventID       uuid.UUID `db:"event_id" json:"event_id" validate:"required"`
	StepName      string    `db:"step_name" json:"step_name" validate:"required"`
	StepKind      string    `db:"step_kind" json:"step_kind" validate:"required"`
	Status        string    `db:"status" json:"status" validate:"required"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

type InsertSagaStepRequest struct {
	*SagaStep
}

type AdvanceRequest struct {
	Events []InsertEventRequest
	Steps  []InsertSagaStepRequest
	*UpdateSagaStateRequest
}

type AdvanceWithFirstStepRequest struct {
	InsertTransactionRequest
	InsertPartiesRequest
	Events []InsertEventRequest
	Steps  []InsertSagaStepRequest
	*UpdateSagaStateRequest
}
