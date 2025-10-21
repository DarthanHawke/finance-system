// Пакет service предоставляет бизнес-логику для управления платежами
package service

import (
	"context"
)

// TransactionService реализует бизнес-логику работы с платежами
type TransactionService struct {
	producer Producer
}

// NewTransactionService создает новый экземпляр TransactionService
func NewTransactionService(
	producer Producer,
) *TransactionService {
	return &TransactionService{
		producer: producer,
	}
}

type Producer interface {
	ProduceMessage(ctx context.Context, topic string, key string, value any) error
}

// HandleDepositeRequest
func (s *TransactionService) HandleDepositeRequest(ctx context.Context) error {
	const op = "service.transaction.HandleDepositeRequest"

	// TODO Отпраляет запрос на пополнение

	return nil
}
