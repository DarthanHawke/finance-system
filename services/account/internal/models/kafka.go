package models

import (
	"time"
)

// Типы событий для Kafka
const (
	EventFreezeRequest    = "freeze.request"
	EventFreezeComplete   = "freeze.complete"
	EventFreezeFail       = "freeze.fail"
	EventUnfreezeRequest  = "unfreeze.request"
	EventUnfreezeComplete = "unfreeze.complete"
	EventUnfreezeFail     = "unfreeze.fail"

	EventReserveRequest    = "reserve.request"
	EventReserveComplete   = "reserve.complete"
	EventReserveFail       = "reserve.fail"
	EventUnreserveRequest  = "unreserve.request"
	EventUnreserveComplete = "unreserve.complete"
	EventUnreserveFail     = "unreserve.fail"

	EventDepositRequest  = "deposit.request"
	EventDepositComplete = "deposit.complete"
	EventDepositFail     = "deposit.fail"

	EventWithdrawRequest  = "withdraw.request"
	EventWithdrawComplete = "withdraw.complete"
	EventWithdrawFail     = "withdraw.fail"
)

type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	PaymentID string    `json:"payment_id"`
	Source    string    `json:"source"`
}

type BalanceRequestEvent struct {
	Event
	AccountCode string  `json:"code"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
}

type BalanceResponseEvent struct {
	Event
	Success bool   `json:"success"`
	Reason  string `json:"reason,omitempty"`
}
