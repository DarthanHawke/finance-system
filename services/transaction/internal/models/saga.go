// Пакет models содержит различные dto'шки, модельки, константы
package models

import (
	"time"

	"github.com/google/uuid"
)

// статусы шагов саги
const (
	StepStatusStarted   string = "STARTED"
	StepStatusCompleted string = "COMPLETED"
	StepStatusFailed    string = "FAILED"
)

// тип шагов стаги
const (
	StepKindAction       string = "ACTION"
	StepKindCompensation string = "COMPENSATION"
)

// SagaStep структура шага саги
type SagaStep struct {
	ID            uuid.UUID `db:"id" json:"id" validate:"required"`
	TransactionID uuid.UUID `db:"transaction_id" json:"transaction_id" validate:"required"`
	EventID       uuid.UUID `db:"event_id" json:"event_id" validate:"required"`
	StepName      string    `db:"step_name" json:"step_name" validate:"required"`
	StepKind      string    `db:"step_kind" json:"step_kind" validate:"required"`
	Status        string    `db:"saga_status" json:"saga_status" validate:"required"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

// InsertSagaStepRequest обертка для создания шага саги
type InsertSagaStepRequest struct {
	*SagaStep
}

// AdvanceRequest обертка с данными для продвижения саги и создания событий в одной транзакции
type AdvanceRequest struct {
	Events []InsertEventRequest
	Steps  []InsertSagaStepRequest
	*UpdateSagaStateRequest
}

// AdvanceWithFirstStepRequest обертка с данными для продвижения саги и создания бизнес-транзакции и событий в одной транзакции
type AdvanceWithFirstStepRequest struct {
	InsertTransactionRequest
	InsertPartiesRequest
	Events []InsertEventRequest
	Steps  []InsertSagaStepRequest
	*UpdateSagaStateRequest
}
