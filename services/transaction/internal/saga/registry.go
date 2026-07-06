package saga

import (
	"transaction-service/internal/models"
)

const (
	StateInit                  = "INIT"
	StateSenderFreezing        = "SENDER_FREEZING"
	StateExternalPayment       = "EXTERNAL_PAYMENT"
	StateCreditingRecipient    = "CREDITING_RECIPIENT"
	StateCreditingSender       = "CREDITING_SENDER"
	StateDebitingRecipient     = "DEBITING_RECIPIENT"
	StateSenderCapturing       = "SENDER_CAPTURING"
	StateRequestingTopUp       = "REQUESTING_TOP_UP"
	StateCompensatingRecipient = "COMPENSATING_RECIPIENT"
	StateCompensatingSender    = "COMPENSATING_SENDER"
	StateCompensatingExternal  = "COMPENSATING_EXTERNAL"
	StateCompleted             = "COMPLETED"
	StateCancelled             = "CANCELLED"
	StateBlocked               = "BLOCKED"
)

const (
	StepFreezeSender    = "freeze_sender"
	StepCreditSender    = "credit_sender"
	StepCreditRecipient = "credit_recipient"
	StepCaptureSender   = "capture_sender"
	StepDebitRecipient  = "debit_recipient"

	StepTopupRequest      = "topup_request"
	StepOutboundPayment   = "outbound_payment"
	StepInboundPayment    = "inbound_payment"
	StepRefundPayment     = "refund_payment"
	StepRollbackPayment   = "rollback_payment"
	StepChargebackPayment = "chargeback_payment"

	StepReversingRecipient = "reversing_recipient"
	StepUnfreezeSender     = "unfreeze_sender"
	StepBlockSender        = "block_sender"
	StepBlockRecipient     = "block_recipient"

	StepTransactionCreated   = "transaction_created"
	StepTransactionCompleted = "transaction_completed"
	StepTransactionCancelled = "transaction_cancelled"
	StepTransactionBlocked   = "transaction_blocked"
)

func Register() map[string]Definition {
	return map[string]Definition{
		models.SagaOnUsTransfer:       buildOnUsTransfer(),
		models.SagaOutboundRemittance: buildOutboundRemittance(),
		models.SagaInboundRemittance:  buildInboundRemittance(),
		models.SagaTopUpRequest:       buildTopUpRequest(),
		models.SagaRefundInternal:     buildRefundInternal(),
		models.SagaRefundOutbound:     buildRefundOutbound(),
		models.SagaRefundInbound:      buildRefundInbound(),
		models.SagaChargebackInternal: buildChargebackInternal(),
		models.SagaChargebackOutbound: buildChargebackOutbound(),
		models.SagaChargebackInbound:  buildChargebackInbound(),
		models.SagaAdjustmentCredit:   buildAdjustmentCredit(),
		models.SagaAdjustmentDebit:    buildAdjustmentDebit(),
	}
}

func buildOnUsTransfer() Definition {
	transaction := map[string]map[int]Transition{}

	transaction[StateInit] = map[int]Transition{
		OutcomeSuccess: {
			From: StateInit, NewState: StateSenderFreezing,
			Commands:      []Command{freezeSenderCmd()},
			Notifications: []Notification{transactionCreatedNotif()},
		},
	}

	transaction[StateSenderFreezing] = map[int]Transition{
		OutcomeSuccess: {
			From: StateSenderFreezing, NewState: StateCreditingRecipient,
			Commands: []Command{creditRecipientCmd()},
		},
		OutcomeFailed: {
			From: StateSenderFreezing, NewState: StateCancelled,
			NewStatus:     models.TransactionStatusCancelled,
			Notifications: []Notification{transactionCancelledNotif(models.ReasonSenderFreezeRejected)},
		},
	}

	transaction[StateCreditingRecipient] = map[int]Transition{
		OutcomeSuccess: {
			From: StateCreditingRecipient, NewState: StateSenderCapturing,
			Commands: []Command{captureSenderCmd()},
		},
		OutcomeFailed: {
			From: StateCreditingRecipient, NewState: StateCompensatingSender,
			Commands: []Command{unfreezeSenderCmd()},
		},
	}

	transaction[StateSenderCapturing] = map[int]Transition{
		OutcomeSuccess: {
			From: StateSenderCapturing, NewState: StateCompleted,
			NewStatus:     models.TransactionStatusCompleted,
			Notifications: []Notification{transactionCompletedNotif()},
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
			Commands:      []Command{blockSenderCmd(), blockRecipientCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	transaction[StateCompensatingSender] = map[int]Transition{
		OutcomeSuccess: {
			From: StateCompensatingSender, NewState: StateCancelled,
			NewStatus:     models.TransactionStatusCancelled,
			Notifications: []Notification{transactionCancelledNotif(models.ReasonCompensated)},
		},
		OutcomeFailed: {
			From: StateCompensatingSender, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands:      []Command{blockSenderCmd(), blockRecipientCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	return Definition{
		SagaType: models.SagaOnUsTransfer, InitState: StateInit, Transitions: transaction,
	}
}

func buildOutboundRemittance() Definition {
	t := map[string]map[int]Transition{}

	t[StateInit] = map[int]Transition{
		OutcomeSuccess: {
			From: StateInit, NewState: StateSenderFreezing,
			Commands:      []Command{freezeSenderCmd()},
			Notifications: []Notification{transactionCreatedNotif()},
		},
	}

	t[StateSenderFreezing] = map[int]Transition{
		OutcomeSuccess: {
			From: StateSenderFreezing, NewState: StateExternalPayment,
			Commands: []Command{outboundPaymentCmd()},
		},
		OutcomeFailed: {
			From: StateSenderFreezing, NewState: StateCancelled,
			NewStatus:     models.TransactionStatusCancelled,
			Notifications: []Notification{transactionCancelledNotif(models.ReasonSenderFreezeRejected)},
		},
	}

	t[StateExternalPayment] = map[int]Transition{
		OutcomeSuccess: {
			From: StateExternalPayment, NewState: StateSenderCapturing,
			Commands: []Command{captureSenderCmd()},
		},
		OutcomeFailed: {
			From: StateExternalPayment, NewState: StateCompensatingSender,
			Commands: []Command{unfreezeSenderCmd()},
		},
	}

	t[StateSenderCapturing] = map[int]Transition{
		OutcomeSuccess: {
			From: StateSenderCapturing, NewState: StateCompleted,
			NewStatus:     models.TransactionStatusCompleted,
			Notifications: []Notification{transactionCompletedNotif()},
		},
		OutcomeFailed: {
			From: StateSenderCapturing, NewState: StateCompensatingExternal,
			Commands: []Command{rollbackPaymentCmd()},
		},
	}

	t[StateCompensatingExternal] = map[int]Transition{
		OutcomeSuccess: {
			From: StateCompensatingExternal, NewState: StateCompensatingSender,
			Commands: []Command{unfreezeSenderCmd()},
		},
		OutcomeFailed: {
			From: StateCompensatingExternal, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands:      []Command{blockSenderCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	t[StateCompensatingSender] = map[int]Transition{
		OutcomeSuccess: {
			From: StateCompensatingSender, NewState: StateCancelled,
			NewStatus:     models.TransactionStatusCancelled,
			Notifications: []Notification{transactionCancelledNotif(models.ReasonCompensated)},
		},
		OutcomeFailed: {
			From: StateCompensatingSender, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands:      []Command{blockSenderCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	return Definition{
		SagaType: models.SagaOutboundRemittance, InitState: StateInit, Transitions: t,
	}
}

func buildInboundRemittance() Definition {
	t := map[string]map[int]Transition{}

	t[StateInit] = map[int]Transition{
		OutcomeSuccess: {
			From: StateInit, NewState: StateCreditingRecipient,
			Commands:      []Command{creditRecipientCmd()},
			Notifications: []Notification{transactionCreatedNotif()},
		},
	}

	t[StateCreditingRecipient] = map[int]Transition{
		OutcomeSuccess: {
			From: StateCreditingRecipient, NewState: StateCompleted,
			NewStatus:     models.TransactionStatusCompleted,
			Notifications: []Notification{transactionCompletedNotif()},
		},
		OutcomeFailed: {
			From: StateCreditingRecipient, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands:      []Command{blockRecipientCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	return Definition{
		SagaType: models.SagaInboundRemittance, InitState: StateInit, Transitions: t,
	}
}

func buildTopUpRequest() Definition {
	t := map[string]map[int]Transition{}

	t[StateInit] = map[int]Transition{
		OutcomeSuccess: {
			From:          StateInit,
			NewState:      StateRequestingTopUp,
			Commands:      []Command{topUpRequestCmd()},
			Notifications: []Notification{transactionCreatedNotif()},
		},
	}

	t[StateRequestingTopUp] = map[int]Transition{
		OutcomeSuccess: {
			From:          StateRequestingTopUp,
			NewState:      StateCompleted,
			NewStatus:     models.TransactionStatusCompleted,
			Notifications: []Notification{transactionCompletedNotif()},
		},
		OutcomeFailed: {
			From:          StateRequestingTopUp,
			NewState:      StateCancelled,
			NewStatus:     models.TransactionStatusCancelled,
			Notifications: []Notification{transactionCancelledNotif(models.ReasonTopUpRejected)},
		},
	}

	return Definition{
		SagaType: models.SagaTopUpRequest, InitState: StateInit, Transitions: t,
	}
}

func buildRefundInternal() Definition {
	t := map[string]map[int]Transition{}

	t[StateInit] = map[int]Transition{
		OutcomeSuccess: {
			From: StateInit, NewState: StateCreditingSender,
			Commands:      []Command{creditSenderCmd()},
			Notifications: []Notification{transactionCreatedNotif()},
		},
	}

	t[StateCreditingSender] = map[int]Transition{
		OutcomeSuccess: {
			From: StateCreditingSender, NewState: StateDebitingRecipient,
			Commands: []Command{debitRecipientCmd()},
		},
		OutcomeFailed: {
			From: StateCreditingSender, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands:      []Command{blockSenderCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	t[StateDebitingRecipient] = map[int]Transition{
		OutcomeSuccess: {
			From: StateDebitingRecipient, NewState: StateCompleted,
			NewStatus:     models.TransactionStatusCompleted,
			Notifications: []Notification{transactionCompletedNotif()},
		},
		OutcomeFailed: {
			From: StateDebitingRecipient, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands:      []Command{blockRecipientCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	return Definition{
		SagaType: models.SagaRefundInternal, InitState: StateInit, Transitions: t,
	}
}

func buildRefundOutbound() Definition {
	t := map[string]map[int]Transition{}

	t[StateInit] = map[int]Transition{
		OutcomeSuccess: {
			From: StateInit, NewState: StateCreditingSender,
			Commands:      []Command{creditSenderCmd()},
			Notifications: []Notification{transactionCreatedNotif()},
		},
	}

	t[StateCreditingSender] = map[int]Transition{
		OutcomeSuccess: {
			From: StateCreditingSender, NewState: StateCompleted,
			NewStatus:     models.TransactionStatusCompleted,
			Notifications: []Notification{transactionCompletedNotif()},
		},
		OutcomeFailed: {
			From: StateCreditingSender, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands:      []Command{blockSenderCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	return Definition{
		SagaType: models.SagaRefundOutbound, InitState: StateInit, Transitions: t,
	}
}

func buildRefundInbound() Definition {
	t := map[string]map[int]Transition{}

	t[StateInit] = map[int]Transition{
		OutcomeSuccess: {
			From: StateInit, NewState: StateDebitingRecipient,
			Commands:      []Command{debitRecipientCmd()},
			Notifications: []Notification{transactionCreatedNotif()},
		},
	}

	t[StateDebitingRecipient] = map[int]Transition{
		OutcomeSuccess: {
			From: StateDebitingRecipient, NewState: StateExternalPayment,
			Commands: []Command{refundPaymentCmd()},
		},
		OutcomeFailed: {
			From: StateDebitingRecipient, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands:      []Command{blockRecipientCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	t[StateExternalPayment] = map[int]Transition{
		OutcomeSuccess: {
			From: StateExternalPayment, NewState: StateCompleted,
			NewStatus:     models.TransactionStatusCompleted,
			Notifications: []Notification{transactionCompletedNotif()},
		},
		OutcomeFailed: {
			From: StateExternalPayment, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands:      []Command{blockRecipientCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	return Definition{
		SagaType: models.SagaRefundInbound, InitState: StateInit, Transitions: t,
	}
}

func buildChargebackInternal() Definition {
	t := map[string]map[int]Transition{}

	t[StateInit] = map[int]Transition{
		OutcomeSuccess: {
			From: StateInit, NewState: StateDebitingRecipient,
			Commands:      []Command{debitRecipientCmd()},
			Notifications: []Notification{transactionCreatedNotif()},
		},
	}

	t[StateDebitingRecipient] = map[int]Transition{
		OutcomeSuccess: {
			From: StateDebitingRecipient, NewState: StateCreditingSender,
			Commands: []Command{creditSenderCmd()},
		},
		OutcomeFailed: {
			From: StateDebitingRecipient, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands:      []Command{blockRecipientCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	t[StateCreditingSender] = map[int]Transition{
		OutcomeSuccess: {
			From: StateCreditingSender, NewState: StateCompleted,
			NewStatus:     models.TransactionStatusCompleted,
			Notifications: []Notification{transactionCompletedNotif()},
		},
		OutcomeFailed: {
			From: StateCreditingSender, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands:      []Command{blockSenderCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	return Definition{
		SagaType: models.SagaChargebackInternal, InitState: StateInit, Transitions: t,
	}
}

func buildChargebackOutbound() Definition {
	t := map[string]map[int]Transition{}

	t[StateInit] = map[int]Transition{
		OutcomeSuccess: {
			From: StateInit, NewState: StateCreditingSender,
			Commands:      []Command{creditSenderCmd()},
			Notifications: []Notification{transactionCreatedNotif()},
		},
	}

	t[StateCreditingSender] = map[int]Transition{
		OutcomeSuccess: {
			From: StateCreditingSender, NewState: StateCompleted,
			NewStatus:     models.TransactionStatusCompleted,
			Notifications: []Notification{transactionCompletedNotif()},
		},
		OutcomeFailed: {
			From: StateCreditingSender, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands:      []Command{blockSenderCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	return Definition{
		SagaType: models.SagaChargebackOutbound, InitState: StateInit, Transitions: t,
	}
}

func buildChargebackInbound() Definition {
	t := map[string]map[int]Transition{}

	t[StateInit] = map[int]Transition{
		OutcomeSuccess: {
			From: StateInit, NewState: StateDebitingRecipient,
			Commands:      []Command{debitRecipientCmd()},
			Notifications: []Notification{transactionCreatedNotif()},
		},
	}

	t[StateDebitingRecipient] = map[int]Transition{
		OutcomeSuccess: {
			From: StateDebitingRecipient, NewState: StateExternalPayment,
			Commands: []Command{chargebackPaymentCmd()},
		},
		OutcomeFailed: {
			From: StateDebitingRecipient, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands:      []Command{blockRecipientCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	t[StateExternalPayment] = map[int]Transition{
		OutcomeSuccess: {
			From: StateExternalPayment, NewState: StateCompleted,
			NewStatus:     models.TransactionStatusCompleted,
			Notifications: []Notification{transactionCompletedNotif()},
		},
		OutcomeFailed: {
			From: StateExternalPayment, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands:      []Command{blockRecipientCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	return Definition{
		SagaType: models.SagaChargebackInbound, InitState: StateInit, Transitions: t,
	}
}

func buildAdjustmentCredit() Definition {
	t := map[string]map[int]Transition{}

	t[StateInit] = map[int]Transition{
		OutcomeSuccess: {
			From: StateInit, NewState: StateCreditingRecipient,
			Commands:      []Command{creditRecipientCmd()},
			Notifications: []Notification{transactionCreatedNotif()},
		},
	}

	t[StateCreditingRecipient] = map[int]Transition{
		OutcomeSuccess: {
			From: StateCreditingRecipient, NewState: StateCompleted,
			NewStatus:     models.TransactionStatusCompleted,
			Notifications: []Notification{transactionCompletedNotif()},
		},
		OutcomeFailed: {
			From: StateCreditingRecipient, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands:      []Command{blockRecipientCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	return Definition{
		SagaType: models.SagaAdjustmentCredit, InitState: StateInit, Transitions: t,
	}
}

func buildAdjustmentDebit() Definition {
	t := map[string]map[int]Transition{}

	t[StateInit] = map[int]Transition{
		OutcomeSuccess: {
			From: StateInit, NewState: StateDebitingRecipient,
			Commands:      []Command{debitRecipientCmd()},
			Notifications: []Notification{transactionCreatedNotif()},
		},
	}

	t[StateDebitingRecipient] = map[int]Transition{
		OutcomeSuccess: {
			From: StateDebitingRecipient, NewState: StateCompleted,
			NewStatus:     models.TransactionStatusCompleted,
			Notifications: []Notification{transactionCompletedNotif()},
		},
		OutcomeFailed: {
			From: StateDebitingRecipient, NewState: StateBlocked,
			NewStatus: models.TransactionStatusBlocked, Terminal: true,
			Commands:      []Command{blockRecipientCmd()},
			Notifications: []Notification{transactionBlockedNotif(models.ReasonTransactionCannotBeComplited)},
		},
	}

	return Definition{
		SagaType: models.SagaAdjustmentDebit, InitState: StateInit, Transitions: t,
	}
}
