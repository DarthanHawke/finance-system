// Пакет apperr содержит ошибки
package apperr

import (
	"errors"
	"fmt"
)

// Error — доменная ошибка
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// WrappedError — ошибка с контекстом операции.
type WrappedError struct {
	Op  string
	Err error
}

func (e *WrappedError) Error() string {
	if e.Err == nil {
		return e.Op
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *WrappedError) Unwrap() error {
	return e.Err
}

// TerminalError — ошибка, требующая ручного вмешательства.
type TerminalError struct {
	Op      string
	Context string
	Err     error
}

func (e *TerminalError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("TERMINAL %s: %s", e.Op, e.Context)
	}
	return fmt.Sprintf("TERMINAL %s: %s: %v", e.Op, e.Context, e.Err)
}

func (e *TerminalError) Unwrap() error {
	return e.Err
}

// Предопределенные ошибки
// Ошибки счёта, баланса, суммы:
var (
	// ErrTransactionIDNotUnique - ID платежа уже занят
	ErrTransactionIDNotUnique = &Error{
		Code:    "TRANSACTION_ID_NOT_UNIQUE",
		Message: "transaction code not unique",
	}
	// ErrTransactionPartiesAlreadyExist - ID платежа уже занят
	ErrTransactionPartiesAlreadyExist = &Error{
		Code:    "TRANSACTION_PARTIES_ALREADY_EXIST",
		Message: "parties for this transaction already exist",
	}
	// ErrTransactionNotFound - платеж не найден
	ErrTransactionNotFound = &Error{
		Code:    "TRANSACTION_NOT_FOUND",
		Message: "Transaction not found",
	}
	// ErrUnknownTransactionType - не известный тип операций
	ErrUnknownTransactionType = &Error{
		Code:    "UNKNOWN_TRANSACTION_TYPE",
		Message: "unknown transaction type",
	}
)

// Ошибки событий
var (
	// ErrEventIDNotUnique - ID события уже занят
	ErrEventIDNotUnique = &Error{
		Code:    "EVENT_ID_NOT_UNIQUE",
		Message: "event code not unique",
	}
	// ErrEventUnsuccess - событе вернуло успешно вернуло сообщение с отрицательным результатом
	ErrEventUnsuccess = &Error{
		Code:    "EVENT_UNSUCCESS",
		Message: "event unsuccess",
	}
	// ErrEventNotFound — событие не найдено
	ErrEventNotFound = &Error{
		Code:    "EVENT_NOT_FOUND",
		Message: "event not found",
	}
	// ErrUnknownEventType — неизвестный тип события
	ErrUnknownEventType = &Error{
		Code:    "UNKNOWN_EVENT_TYPE",
		Message: "unknown event type",
	}
)

// Classify определяет, является ли ошибка retryable или terminal.
func Classify(err error) (retryable bool, terminal bool) {
	if err == nil {
		return false, false
	}

	// TerminalError — не ретраим, требуется ручное вмешательство
	if _, ok := errors.AsType[*TerminalError](err); ok {
		return false, true
	}

	// Доменная ошибка — не ретраим (невалидные данные, бизнес-отказ)
	if _, ok := errors.AsType[*Error](err); ok {
		return false, false
	}

	// Инфраструктурная ошибка — ретраим
	return true, false
}
