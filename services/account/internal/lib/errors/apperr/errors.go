// Пакет apperr реализует пользовательский тип для обёртки над предопределёнными ошибками
package apperr

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

// Предопределенные ошибки
// Ошибки счёта, баланса, суммы:
var (
	// ErrAccountCodeNotUnique - номер счета уже занят
	ErrAccountCodeNotUnique = &AppError{Code: "ACCOUNT_CODE_NOT_UNIQUE", Message: "account code not unique"}

	// ErrAccountNotFound - счет не найден
	ErrAccountNotFound = &AppError{Code: "ACCOUNT_NOT_FOUND", Message: "account not found"}

	// ErrAccountUpdateFailed - невозможно обновить счет(счет не найден, недостаточно средств, счет не активен)
	ErrAccountUpdateFailed = &AppError{Code: "ACCOUNT_UPDATE_FAILED", Message: "account update failed"}

	// ErrInvalidAmount - неккоректная сумма
	ErrInvalidAmount = &AppError{Code: "INVALID_AMOUNT", Message: "invalid amount"}
)

// Ошибки событий
var (
	// ErrEventIDNotUnique - номер счета уже занят
	ErrEventIDNotUnique = &AppError{Code: "EVENT_ID_NOT_UNIQUE", Message: "event code not unique"}
)
