package saga

import (
	"transaction-service/internal/models"

	"github.com/google/uuid"
)

const (
	OutcomeSuccess int = iota
	OutcomeFailed
)

type Apply struct {
	TransactionID uuid.UUID
	Outcome       int
	TraceID       string
	SpanID        string
}

type Command struct {
	EventType    string
	StepName     string
	StepKind     string
	BuildPayload func(transaction *models.Transaction) (any, error)
}

type Transition struct {
	From      string
	NewState  string
	NewStatus string
	Commands  []Command
	Terminal  bool
}

type Definition struct {
	SagaType    string
	InitState   string
	Transitions map[string]map[int]Transition
}
