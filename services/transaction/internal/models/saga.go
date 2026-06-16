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
	ID            uuid.UUID `db:"id" json:"id"`
	TransactionID uuid.UUID `db:"transaction_id" json:"transaction_id"`
	EventID       uuid.UUID `db:"event_id" json:"event_id,omitempty"`
	StepName      string    `db:"step_name" json:"step_name"`
	StepKind      string    `db:"step_kind" json:"step_kind"`
	Status        string    `db:"status" json:"status"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

// CreateSagaStepRequest - запрос на создание шага саги
type CreateSagaStepRequest struct {
	ID            uuid.UUID `db:"id"`
	TransactionID uuid.UUID `db:"transaction_id"`
	EventID       uuid.UUID `db:"event_id"`
	StepName      string    `db:"step_name"`
	StepKind      string    `db:"step_kind"`
	Status        string    `db:"status"`
}

// GetSagaStepRequest - запрос на получение шага саги
type GetSagaStepRequest struct {
	ID            uuid.UUID `json:"id" validate:"required"`
	TransactionID uuid.UUID `json:"transaction_id" validate:"required"`
	StepName      string    `json:"step_name" validate:"required"`
	StepKind      string    `json:"step_kind" validate:"required"`
}

// GetSagaStepResponse - ответ с шагом саги
type GetSagaStepResponse struct {
	Step *SagaStep `json:"step"`
}

// GetSagaStepsByTransactionRequest - запрос на получение всех шагов саги по транзакции
type GetSagaStepsByTransactionRequest struct {
	TransactionID uuid.UUID `json:"transaction_id" validate:"required"`
}

// GetSagaStepsByTransactionResponse - ответ со всеми шагами саги по транзакции
type GetSagaStepsByTransactionResponse struct {
	Steps []SagaStep `json:"steps"`
}

// UpdateSagaStepStatusRequest - запрос на обновление статуса шага саги
type UpdateSagaStepStatusRequest struct {
	ID      uuid.UUID `db:"id" json:"id" validate:"required"`
	Status  string    `db:"status" json:"status" validate:"required,oneof=STARTED COMPLETED FAILED"`
	EventID uuid.UUID `db:"event_id" json:"event_id"`
}

// UpdateSagaStateRequest - запрос на обновление состояния саги
type UpdateSagaStateRequest struct {
	ID        uuid.UUID `db:"id" json:"id" validate:"required"`
	SagaState string    `db:"saga_state" json:"saga_state" validate:"required"`
	Status    string    `db:"status" json:"status,omitempty"`
}

// AdvanceRequest - запрос на атомарное продвижение саги
type AdvanceRequest struct {
	TransactionID uuid.UUID              `db:"transaction_id" json:"transaction_id" validate:"required"`
	ExpectedFrom  string                 `db:"expected_from" json:"expected_from" validate:"required"`
	NewState      string                 `db:"new_state" json:"new_state" validate:"required"`
	NewStatus     *string                `db:"new_status" json:"new_status,omitempty"`
	Step          *CreateSagaStepRequest `json:"step,omitempty"`
	Event         *CreateEventRequest    `json:"event,omitempty"`
	ExtraEvents   []*CreateEventRequest  `json:"extra_events,omitempty"`
}
