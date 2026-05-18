package models

/* legacy code
import (
	"time"

	"github.com/google/uuid"
)

const (
	SystemRelationType  string = "system"
	ClientRelationType  string = "client"
	AdminRelationType   string = "admin"
	SupportRelationType string = "support"
)

const (
	FinanceSystemEntity string = "financeSystem"
	TransactionEntity   string = "transaction"
	AccountEntity       string = "account"
	CurrencyEntity      string = "currency"
	UserEntity          string = "user"
	RelationEntyty      string = "relation"
	PermissionEntity    string = "permission"
)

const (
	TransactionCreate       string = "transaction_create"
	TransactionRead         string = "transaction_read"
	TransactionReadAll      string = "transaction_read_all"
	TransactionCancel       string = "transaction_cancel"
	TransactionConvert      string = "pyment_convert"
	TransactionUpdateStatus string = "transaction_update_status"
	UserRead                string = "user_read"
	UserReadAll             string = "user_read_all"
	UserUpdate              string = "user_update"
	AccountCreate           string = "account_create"
	AccountRead             string = "account_read"
	CurrencyRead            string = "currency_read"
	CurrencyUpdate          string = "currency_update"
	SessionTerminate        string = "session_terminate"
	SessionManage           string = "session_manage"
	RelationManage          string = "relation_manage"
	RelationRead            string = "relation_read"
	PermissionManage        string = "permission_manage"
	PermissionRead          string = "permission_read"
)

// Entity представляет сущность в системе ReBAC
type Entity struct {
	ID        uuid.UUID `db:"id"`
	Type      string    `db:"type"`
	CreatedAt time.Time `db:"created_at"`
}

// Relation представляет отношение между сущностями
type Relation struct {
	ID           uuid.UUID `db:"id"`
	SourceID     uuid.UUID `db:"source_id"`
	TargetID     uuid.UUID `db:"target_id"`
	RelationType string    `db:"relation_type"`
	CreatedAt    time.Time `db:"created_at"`
}

// Permission представляет разрешение в системе
type Permission struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
}

// PermissionAssignment представляет назначение разрешения для типа отношения
type PermissionAssignment struct {
	RelationType string    `db:"relation_type"`
	PermissionID uuid.UUID `db:"permission_id"`
	CreatedAt    time.Time `db:"created_at"`
}
*/
