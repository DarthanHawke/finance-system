// Пакет service предоставляет бизнес-логику для управления платежами
package service

import (
	"context"
)

// PaymentService реализует бизнес-логику работы с платежами
type PaymentService struct {
	producer Producer
}

// NewPaymentService создает новый экземпляр PaymentService
func NewPaymentService(
	producer Producer,
) *PaymentService {
	return &PaymentService{
		producer: producer,
	}
}

type Producer interface {
	ProduceMessage(ctx context.Context, topic string, key string, value any) error
}

// HandleDepositeRequest
func (s *PaymentService) HandleDepositeRequest(ctx context.Context) error {
	const op = "service.payment.HandleDepositeRequest"

	// TODO Отпраляет запрос на пополнение

	return nil
}
