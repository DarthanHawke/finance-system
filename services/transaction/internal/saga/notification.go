package saga

import (
	"fmt"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"

	"github.com/google/uuid"
)

func AnalyticCreatePayload(transaction *models.Transaction) (any, error) {
	sender, ok := partyIdentifiers(transaction, models.PartyRoleSender)
	if !ok {
		return nil, &apperr.WrappedError{
			Op:  "saga.AnalyticPayload",
			Err: fmt.Errorf("party %q not found", models.PartyRoleSender),
		}
	}
	recipient, ok := partyIdentifiers(transaction, models.PartyRoleRecipient)
	if !ok {
		return nil, &apperr.WrappedError{
			Op:  "saga.AnalyticPayload",
			Err: fmt.Errorf("party %q not found", models.PartyRoleRecipient),
		}
	}
	var parentID string
	if transaction.ParentTransactionID != uuid.Nil {
		parentID = transaction.ParentTransactionID.String()
	}
	return models.TransactionCreatedPayload{
		ID:                  transaction.ID.String(),
		ParentTransactionID: parentID,
		Type:                transaction.Type,
		Status:              transaction.Status,
		Amount:              transaction.Amount,
		Currency:            transaction.Currency,
		Sender:              sender,
		Recipient:           recipient,
		Description:         transaction.Description,
		CreatedAt:           transaction.CreatedAt,
	}, nil
}

func AnalyticPayload(transaction *models.Transaction, reason string) (any, error) {
	return models.TransactionUpdatePayload{
		ID:       transaction.ID.String(),
		Status:   transaction.Status,
		Reason:   reason,
		UpdateAt: transaction.UpdatedAt,
	}, nil
}

func transactionCreatedNotif() Notification {
	return Notification{
		EventType: models.EventTransactionCreated,
		StepName:  StepTransactionCreated,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return AnalyticCreatePayload(transaction)
		},
	}
}

func transactionCompletedNotif() Notification {
	return Notification{
		EventType: models.EventTransactionCompleted,
		StepName:  StepTransactionCompleted,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return AnalyticPayload(transaction, models.ReasonCompleted)
		},
	}
}

func transactionCancelledNotif(reason string) Notification {
	return Notification{
		EventType: models.EventTransactionCancelled,
		StepName:  StepTransactionCancelled,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return AnalyticPayload(transaction, reason)
		},
	}
}

func transactionBlockedNotif(reason string) Notification {
	return Notification{
		EventType: models.EventTransactionBlocked,
		StepName:  StepTransactionBlocked,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return AnalyticPayload(transaction, reason)
		},
	}
}
