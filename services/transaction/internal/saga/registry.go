package saga

import "transaction-service/internal/models"

const (
	StateInit                  = "INIT"
	StateSenderFreezing        = "SENDER_FREEZING"
	StateRecipientDepositing   = "RECIPIENT_DEPOSITING"
	StateSenderCapturing       = "SENDER_CAPTURING"
	StateCompensatingRecipient = "COMPENSATING_RECIPIENT"
	StateCompensatingSender    = "COMPENSATING_SENDER"
	StateCompleted             = "COMPLETED"
	StateCancelled             = "CANCELLED"
	StateBlocked               = "BLOCKED"
)

const (
	StepFreezeSender       = "freeze_sender"
	StepDepositRecipient   = "deposit_recipient"
	StepCaptureSender      = "capture_sender"
	StepReversingRecipient = "reversing_recipient"
	StepUnfreezeSender     = "unfreeze_sender"
	StepBlockSender        = "block_sender"
	StepBlockRecipient     = "block_recipient"
)

func buildOnUsTransfer() Definition {
	transaction := map[string]map[int]Transition{}

	transaction[StateSenderFreezing] = map[int]Transition{
		OutcomeSuccess: {
			From: StateSenderFreezing, NewState: StateRecipientDepositing,
			Commands: []Command{depositRecipientCmd()},
		},
		OutcomeFailed: {
			From: StateSenderFreezing, NewState: StateCancelled,
			NewStatus: models.TransactionStatusCancelled,
			Commands:  []Command{transactionCancelledCmd("sender freeze rejected")},
		},
	}

	transaction[StateRecipientDepositing] = map[int]Transition{
		OutcomeSuccess: {
			From: StateRecipientDepositing, NewState: StateSenderCapturing,
			Commands: []Command{captureSenderCmd()},
		},
		OutcomeFailed: {
			From: StateRecipientDepositing, NewState: StateCompensatingSender,
			Commands: []Command{unfreezeSenderCmd()},
		},
	}

	transaction[StateSenderCapturing] = map[int]Transition{
		OutcomeSuccess: {
			From: StateSenderCapturing, NewState: StateCompleted,
			NewStatus: models.TransactionStatusCompleted,
			Commands:  []Command{transactionCompletedCmd()},
		},
		OutcomeFailed: {
			From: StateSenderCapturing, NewState: StateCompensatingRecipient,
			Commands: []Command{reversingRecipientCmd()},
		},
	}

	transaction[StateCompensatingRecipient] = map[int]Transition{
		OutcomeSuccess: {
			From: StateCompensatingRecipient, NewState: StateCompensatingSender,
			Commands: []Command{unfreezeSenderCmd()},
		},
		OutcomeFailed: {
			From: StateCompensatingRecipient, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands: []Command{blockSenderCmd(), blockRecipientCmd(), transactionBlockedCmd()},
		},
	}

	transaction[StateCompensatingSender] = map[int]Transition{
		OutcomeSuccess: {
			From: StateCompensatingSender, NewState: StateCancelled,
			NewStatus: models.TransactionStatusCancelled,
			Commands:  []Command{transactionCancelledCmd("compensated")},
		},
		OutcomeFailed: {
			From: StateCompensatingSender, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands: []Command{blockSenderCmd(), blockRecipientCmd(), transactionBlockedCmd()},
		},
	}

	return Definition{
		SagaType: models.SagaOnUsTransfer, InitState: StateInit, Transitions: transaction,
	}
}
