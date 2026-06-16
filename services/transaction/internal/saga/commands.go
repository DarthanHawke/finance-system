package saga

import (
	"fmt"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"
)

func partyIdentifiers(transaction *models.Transaction, role string) (map[string]string, bool) {
	for _, p := range transaction.Parties {
		if p.PartyRole == role {
			return p.Identifiers, true
		}
	}
	return nil, false
}

func accountPayload(transaction *models.Transaction, role string, stepName string) (any, string, error) {
	identifiers, ok := partyIdentifiers(transaction, role)
	if !ok {
		return nil, "", &apperr.WrappedError{
			Op:  "saga.accountPayload",
			Err: fmt.Errorf("party %q not found", role),
		}
	}
	code := identifiers["ACCOUNT_CODE"]
	return models.AccountCommandPayload{
		TransactionID: transaction.ID.String(),
		StepName:      stepName,
		AccountCode:   code,
		Amount:        transaction.Amount,
		Currency:      transaction.Currency,
	}, code, nil
}

func freezeSenderCmd() Command {
	return Command{
		EventType: models.EventFreezeRequest,
		StepName:  StepFreezeSender,
		StepKind:  models.StepKindAction,
		BuildPayload: func(transaction *models.Transaction) (any, string, error) {
			return accountPayload(transaction, models.PartyRoleSender, StepFreezeSender)
		},
	}
}

func depositRecipientCmd() Command {
	return Command{
		EventType: models.EventDepositRequest,
		StepName:  StepDepositRecipient,
		StepKind:  models.StepKindAction,
		BuildPayload: func(transaction *models.Transaction) (any, string, error) {
			return accountPayload(transaction, models.PartyRoleRecipient, StepDepositRecipient)
		},
	}
}

func captureSenderCmd() Command {
	return Command{
		EventType: models.EventCaptureRequest,
		StepName:  StepCaptureSender,
		StepKind:  models.StepKindAction,
		BuildPayload: func(transaction *models.Transaction) (any, string, error) {
			return accountPayload(transaction, models.PartyRoleSender, StepCaptureSender)
		},
	}
}

func reversingRecipientCmd() Command {
	return Command{
		EventType: models.EventDebitRequest,
		StepName:  StepReversingRecipient,
		StepKind:  models.StepKindCompensation,
		BuildPayload: func(transaction *models.Transaction) (any, string, error) {
			return accountPayload(transaction, models.PartyRoleRecipient, StepReversingRecipient)
		},
	}
}

func unfreezeSenderCmd() Command {
	return Command{
		EventType: models.EventUnfreezeRequest,
		StepName:  StepUnfreezeSender,
		StepKind:  models.StepKindCompensation,
		BuildPayload: func(transaction *models.Transaction) (any, string, error) {
			return accountPayload(transaction, models.PartyRoleSender, StepUnfreezeSender)
		},
	}
}

func blockSenderCmd() Command {
	return Command{
		EventType: models.EventBlockRequest,
		StepName:  StepBlockSender,
		StepKind:  models.StepKindCompensation,
		BuildPayload: func(transaction *models.Transaction) (any, string, error) {
			return accountPayload(transaction, models.PartyRoleSender, StepBlockSender)
		},
	}
}

func blockRecipientCmd() Command {
	return Command{
		EventType: models.EventBlockRequest,
		StepName:  StepBlockRecipient,
		StepKind:  models.StepKindCompensation,
		BuildPayload: func(transaction *models.Transaction) (any, string, error) {
			return accountPayload(transaction, models.PartyRoleRecipient, StepBlockRecipient)
		},
	}
}

func transactionCompletedCmd() Command {
	return Command{
		EventType: models.EventTransactionCompleted,
		StepName:  "transaction_completed",
		StepKind:  models.StepKindAction,
		BuildPayload: func(transaction *models.Transaction) (any, string, error) {
			return models.TransactionCompletedPayload{CompletedAt: transaction.UpdatedAt}, transaction.ID.String(), nil
		},
	}
}

func transactionCancelledCmd(reason string) Command {
	return Command{
		EventType: models.EventTransactionCancelled,
		StepName:  "transaction_cancelled",
		StepKind:  models.StepKindAction,
		BuildPayload: func(transaction *models.Transaction) (any, string, error) {
			return models.TransactionCancelledPayload{Reason: reason, CancelledAt: transaction.UpdatedAt}, transaction.ID.String(), nil
		},
	}
}

func transactionBlockedCmd() Command {
	return Command{
		EventType: models.EventTransactionBlocked,
		StepName:  "transaction_blocked",
		StepKind:  models.StepKindAction,
		BuildPayload: func(transaction *models.Transaction) (any, string, error) {
			return models.TransactionCancelledPayload{
				Reason:      "blocked, manual intervention required",
				CancelledAt: transaction.UpdatedAt,
			}, transaction.ID.String(), nil
		},
	}
}
