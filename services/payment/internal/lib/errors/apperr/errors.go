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
	// ErrPaymentIDNotUnique - номер счета уже занят
	ErrPaymentIDNotUnique = &AppError{Code: "PAYMENT_ID_NOT_UNIQUE", Message: "payment code not unique"}
	// ErrPaymentNotFound - счет не найден
	ErrPaymentNotFound = &AppError{Code: "PAYMENT_NOT_FOUND", Message: "Payment not found"}
)

// Ошибки событий
var (
	// ErrEventIDNotUnique - номер счета уже занят
	ErrEventIDNotUnique = &AppError{Code: "EVENT_ID_NOT_UNIQUE", Message: "event code not unique"}
)
