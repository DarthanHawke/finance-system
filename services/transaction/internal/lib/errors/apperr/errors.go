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
	// ErrTransactionIDNotUnique - ID платежа уже занят
	ErrTransactionIDNotUnique = &AppError{Code: "TRANSACTION_ID_NOT_UNIQUE", Message: "transaction code not unique"}
	// ErrTransactionNotFound - платеж не найден
	ErrTransactionNotFound = &AppError{Code: "TRANSACTION_NOT_FOUND", Message: "Transaction not found"}
)

// Ошибки событий
var (
	// ErrEventIDNotUnique - ID события уже занят
	ErrEventIDNotUnique = &AppError{Code: "EVENT_ID_NOT_UNIQUE", Message: "event code not unique"}
	// ErrEventUnsuccess - событе вернуло успешно вернуло сообщение с отрицательным результатом
	ErrEventUnsuccess = &AppError{Code: "EVENT_UNSUCCESS", Message: "event unsuccess"}
)
