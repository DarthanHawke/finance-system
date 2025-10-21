// Пакет models содержит различные dto'шки, модельки, константы
package models

import (
	"time"

	"github.com/google/uuid"
)

const Source = "account-service"

// статусы счета:
const (
	// Active активный сччет
	Active string = "active"
	// Blocked заблокированный счет
	Blocked string = "blocked"
	// Closed закрытый счет
	Closed string = "closed"
)

// Account - дтошка счёта
type Account struct {
	Code           string    `db:"code" json:"сode"`
	Name           string    `db:"name" json:"name"`
	UserID         uuid.UUID `db:"user_id" json:"user_id"`
	Currency       string    `db:"currency" json:"currency"`
	Balance        float64   `db:"balance" json:"balance"`
	FrozenBalance  float64   `db:"frozen_balance" json:"frozen_balance"`
	ReserveBalance float64   `db:"reserve_balance" json:"reserve_balance"`
	Status         string    `db:"status" json:"status" validate:"required,oneof=active blocked closed"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

type CreateAccountRequest struct {
	Code     string    `db:"code" json:"сode"`
	Name     string    `db:"name" json:"name" validate:"required,min=1,max=100"`
	UserID   uuid.UUID `db:"user_id" json:"user_id" validate:"required"`
	Currency string    `db:"currency" json:"currency" validate:"required,oneof=USD EUR RUB"`
}

type GetAccountRequest struct {
	Code string `db:"code" json:"сode" validate:"required"`
}

type GetAccountResponse struct {
	*Account
}

type GetAccountsRequest struct {
	UserID uuid.UUID `db:"user_id" json:"user_id" validate:"required"`
	Limit  int       `db:"limit" json:"limit" validate:"required"`
	Offset int       `db:"limit" json:"offset" validate:"required"`
}

type GetAccountsResponse struct {
	Accounts []Account `json:"accounts"`
}

type BalanceRequest struct {
	Code          string    `db:"code" json:"сode" validate:"required"`
	Amount        float64   `db:"amount" json:"amount" validate:"required,gt=0"`
	TransactionID uuid.UUID `json:"transaction_id,omitempty"`
}

type UpdateAccountRequest struct {
	Code string `db:"code" json:"сode" validate:"required"`
}
