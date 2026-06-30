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

type Notification struct {
	EventType    string
	StepName     string
	BuildPayload func(transaction *models.Transaction) (any, error)
}

type Transition struct {
	From          string
	NewState      string
	NewStatus     string
	Commands      []Command
	Notifications []Notification
	Terminal      bool
}

type Definition struct {
	SagaType    string
	InitState   string
	Transitions map[string]map[int]Transition
}

func partyIdentifiers(transaction *models.Transaction, role string) (map[string]string, bool) {
	for _, p := range transaction.Parties {
		if p.PartyRole == role {
			return p.Identifiers, true
		}
	}
	return nil, false
}
