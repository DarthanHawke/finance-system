package service

/* legacy code
import (
	"context"
	"encoding/json"
	"log"

	"notification-service/internal/events"

	"github.com/streadway/amqp"
)

type NotificationService struct {
	listener *events.RabbitMQListener
	logger   *log.Logger
}

func NewNotificationService(listener *events.RabbitMQListener, logger *log.Logger) *NotificationService {
	return &NotificationService{
		listener: listener,
		logger:   logger,
	}
}

func (s *NotificationService) Start(ctx context.Context) error {
	msgs, err := s.listener.Consume()
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-msgs:
				s.processMessage(msg)
			}
		}
	}()

	return nil
}

func (s *NotificationService) processMessage(msg amqp.Delivery) {
	var payload struct {
		ID     string `json:"transaction_id"`
		Status string `json:"status"`
	}

	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		s.logger.Printf("Failed to unmarshal message: %v", err)
		return
	}

	s.logger.Printf("Transaction id: %v, status: %s", payload.ID, payload.Status)
}
*/
