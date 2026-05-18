// Пакет models содержит различные dto'шки, модельки, константы
package models

/* legacy code
import (
	"time"

	"github.com/google/uuid"
)

// статусы счета:
const (
	Active  string = "active"
	Blocked string = "blocked"
	Closed  string = "closed"
)

// Account - дтошка счёта
type Account struct {
	Code         string    `db:"code" json:"сode"`
	Name         string    `db:"name" json:"name"`
	UserID       uuid.UUID `db:"user_id" json:"user_id"`
	Currency     string    `db:"currency" json:"currency"`
	Balance      float64   `db:"balance" json:"balance"`
	BlockedFunds float64   `db:"blocked_funds" json:"blocked_funds"`
	Status       string    `db:"status" json:"status" validate:"required,oneof=active blocked closed"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type CreateAccountRequest struct {
	Code     string    `db:"code" json:"сode" validate:"required"`
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

type FundsRequest struct {
	UserID uuid.UUID `db:"user_id" json:"user_id" validate:"required"`
	Code   string    `db:"code" json:"сode" validate:"required"`
	Amount float64   `db:"amount" json:"amount" validate:"required,gt=0"`
}

type UpdateStatusRequest struct {
	UserID uuid.UUID `db:"user_id" json:"user_id" validate:"required"`
	Code   string    `db:"code" json:"сode" validate:"required"`
	Status string    `db:"status" json:"status" validate:"required,oneof=active blocked closed"`
}
*/
