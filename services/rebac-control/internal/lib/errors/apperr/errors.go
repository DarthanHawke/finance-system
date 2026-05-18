// Пакет apperr реализует пользовательский тип для обёртки над предопределёнными ошибками
package apperr

/* legacy code
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
	// ErrAccountNotFound - счет не найден
	ErrAccountNotFound = &AppError{Code: "ACCOUNT_NOT_FOUND", Message: "Account not found"}

	// ErrInsufficientFunds - недостаточно средств
	ErrInsufficientFunds = &AppError{Code: "INSUFFICIENT_FUNDS", Message: "Insufficient funds"}

	// ErrInvalidAmount - неккоректная сумма
	ErrInvalidAmount = &AppError{Code: "INVALID_AMOUNT", Message: "Invalid amount"}
)
*/
