package saga

import (
	"fmt"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"
)

func accountPayload(transaction *models.Transaction, role string, stepName string) (any, error) {
	identifiers, ok := partyIdentifiers(transaction, role)
	if !ok {
		return nil, &apperr.WrappedError{
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
	}, nil
}

func switchPayload(transaction *models.Transaction, stepName string) (any, error) {
	senderIdentifiers, ok := partyIdentifiers(transaction, models.PartyRoleSender)
	if !ok {
		return nil, &apperr.WrappedError{
			Op:  "saga.switchPayload",
			Err: fmt.Errorf("party SENDER not found"),
		}
	}
	recipientIdentifiers, ok := partyIdentifiers(transaction, models.PartyRoleRecipient)
	if !ok {
		return nil, &apperr.WrappedError{
			Op:  "saga.switchPayload",
			Err: fmt.Errorf("party RECIPIENT not found"),
		}
	}
	return models.SwitchCommandPayload{
		TransactionID: transaction.ID.String(),
		StepName:      stepName,
		Amount:        transaction.Amount,
		Currency:      transaction.Currency,
		Sender:        senderIdentifiers,
		Recipient:     recipientIdentifiers,
		Description:   transaction.Description,
	}, nil
}

func freezeSenderCmd() Command {
	return Command{
		EventType: models.EventFreezeRequest,
		StepName:  StepFreezeSender,
		StepKind:  models.StepKindAction,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return accountPayload(transaction, models.PartyRoleSender, StepFreezeSender)
		},
	}
}

func creditRecipientCmd() Command {
	return Command{
		EventType: models.EventCreditRequest,
		StepName:  StepCreditRecipient,
		StepKind:  models.StepKindAction,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return accountPayload(transaction, models.PartyRoleRecipient, StepCreditRecipient)
		},
	}
}

func debitRecipientCmd() Command {
	return Command{
		EventType: models.EventDebitRequest,
		StepName:  StepDebitRecipient,
		StepKind:  models.StepKindAction,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return accountPayload(transaction, models.PartyRoleRecipient, StepDebitRecipient)
		},
	}
}

func creditSenderCmd() Command {
	return Command{
		EventType: models.EventCreditRequest,
		StepName:  StepCreditSender,
		StepKind:  models.StepKindAction,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return accountPayload(transaction, models.PartyRoleSender, StepCreditSender)
		},
	}
}

func captureSenderCmd() Command {
	return Command{
		EventType: models.EventCaptureRequest,
		StepName:  StepCaptureSender,
		StepKind:  models.StepKindAction,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return accountPayload(transaction, models.PartyRoleSender, StepCaptureSender)
		},
	}
}

func reversingRecipientCmd() Command {
	return Command{
		EventType: models.EventDebitRequest,
		StepName:  StepReversingRecipient,
		StepKind:  models.StepKindCompensation,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return accountPayload(transaction, models.PartyRoleRecipient, StepReversingRecipient)
		},
	}
}

func unfreezeSenderCmd() Command {
	return Command{
		EventType: models.EventUnfreezeRequest,
		StepName:  StepUnfreezeSender,
		StepKind:  models.StepKindCompensation,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return accountPayload(transaction, models.PartyRoleSender, StepUnfreezeSender)
		},
	}
}

func blockSenderCmd() Command {
	return Command{
		EventType: models.EventBlockRequest,
		StepName:  StepBlockSender,
		StepKind:  models.StepKindCompensation,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return accountPayload(transaction, models.PartyRoleSender, StepBlockSender)
		},
	}
}

func blockRecipientCmd() Command {
	return Command{
		EventType: models.EventBlockRequest,
		StepName:  StepBlockRecipient,
		StepKind:  models.StepKindCompensation,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return accountPayload(transaction, models.PartyRoleRecipient, StepBlockRecipient)
		},
	}
}

func outboundPaymentCmd() Command {
	return Command{
		EventType: models.EventPaymentRequest,
		StepName:  StepOutboundPayment,
		StepKind:  models.StepKindAction,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return switchPayload(transaction, StepOutboundPayment)
		},
	}
}

func inboundPaymentCmd() Command {
	return Command{
		EventType: models.EventPaymentRequest,
		StepName:  StepInboundPayment,
		StepKind:  models.StepKindAction,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return switchPayload(transaction, StepInboundPayment)
		},
	}
}

func rollbackPaymentCmd() Command {
	return Command{
		EventType: models.EventRollbackRequest,
		StepName:  StepRollbackPayment,
		StepKind:  models.StepKindCompensation,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return switchPayload(transaction, StepRollbackPayment)
		},
	}
}

func topUpRequestCmd() Command {
	return Command{
		EventType: models.EventPaymentRequest,
		StepName:  StepTopupRequest,
		StepKind:  models.StepKindAction,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return switchPayload(transaction, StepTopupRequest)
		},
	}
}

func refundPaymentCmd() Command {
	return Command{
		EventType: models.EventPaymentRequest,
		StepName:  StepRefundPayment,
		StepKind:  models.StepKindAction,
		BuildPayload: func(transaction *models.Transaction) (any, error) {
			return switchPayload(transaction, StepRefundPayment)
		},
	}
}

func chargebackPaymentCmd() Command {
	return Command{
		EventType: models.EventPaymentRequest,
		StepName:  StepChargebackPayment,
		StepKind:  models.StepKindAction,
		BuildPayload: func(tx *models.Transaction) (any, error) {
			return switchPayload(tx, StepChargebackPayment)
		},
	}
}
