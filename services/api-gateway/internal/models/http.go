package models

import "time"

// SuccessResponse для успешных операций без данных
type SuccessResponse struct {
	Message string `json:"message"`
}

// CreateTransactionResponse ответ на создание транзакции
type CreateTransactionResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

// TransactionStatusResponse ответ со статусом транзакции
type TransactionStatusResponse struct {
	TransactionID string     `json:"transaction_id"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	Error         string     `json:"error,omitempty"`
}

// ErrorResponse стандартный ответ с ошибкой
type ErrorResponse struct {
	Error string `json:"error"`
}
