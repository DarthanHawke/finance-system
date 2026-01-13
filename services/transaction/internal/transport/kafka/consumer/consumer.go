// Пакет kafka предоставляет реализацию консьюмера для Kafka
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"transaction-service/internal/models"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Consumer реализует Kafka consumer для обработки событий
type Consumer struct {
	readers map[string]*kafka.Reader
	handler EventHandler
	logger  *zap.Logger
	wg      sync.WaitGroup
}

// ConsumerConfig содержит конфигурацию для Kafka Consumer
type ConsumerConfig struct {
	Brokers          []string
	GroupID          string
	MinBytes         int
	MaxBytes         int
	MaxWait          time.Duration
	CommitInterval   time.Duration
	SessionTimeout   time.Duration
	RebalanceTimeout time.Duration
	StartOffset      int64
	Concurrency      int
}

// EventHandler определяет интерфейс для обработки событий
type EventHandler interface {
	HandleTransactionResponse(ctx context.Context, event *models.Event) error
	HandleFreezeResponse(ctx context.Context, event *models.Event) error
	HandleReserveResponse(ctx context.Context, event *models.Event) error
	HandleExternalResponse(ctx context.Context, event *models.Event) error
	HandleWithdrawResponse(ctx context.Context, event *models.Event) error
	HandleDepositResponse(ctx context.Context, event *models.Event) error
	HandleUnfreezeResponse(ctx context.Context, event *models.Event) error
	HandleUnreserveResponse(ctx context.Context, event *models.Event) error
	HandleExternalRollbackResponse(ctx context.Context, event *models.Event) error
	HandleRefundResponse(ctx context.Context, event *models.Event) error
	HandleBlockAccountResponse(ctx context.Context, event *models.Event) error
}

// NewConsumer создает новый экземпляр Kafka Consumer
func NewConsumer(
	topics []string,
	config ConsumerConfig,
	handler EventHandler,
	logger *zap.Logger,
) *Consumer {
	readers := make(map[string]*kafka.Reader)

	for _, topic := range topics {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:           config.Brokers,
			GroupID:           config.GroupID,
			Topic:             topic,
			MinBytes:          config.MinBytes,
			MaxBytes:          config.MaxBytes,
			MaxWait:           config.MaxWait,
			CommitInterval:    config.CommitInterval,
			SessionTimeout:    config.SessionTimeout,
			RebalanceTimeout:  config.RebalanceTimeout,
			StartOffset:       config.StartOffset,
			HeartbeatInterval: time.Second * 3,
			RetentionTime:     time.Hour * 24,
			ReadBackoffMin:    time.Millisecond * 100,
			ReadBackoffMax:    time.Second * 1,
		})

		readers[topic] = reader
	}

	return &Consumer{
		readers: readers,
		handler: handler,
		logger:  logger.With(zap.String("component", "kafka_consumer")),
	}
}

// Start запускает консьюмеры для всех топиков
func (c *Consumer) Start(ctx context.Context) {
	const op = "kafka.consumer.Start"

	logger := c.logger.With(
		zap.String("op", op),
	)

	logger.Info("starting kafka consumers")

	for topic, reader := range c.readers {
		c.wg.Add(1)
		go c.consume(ctx, topic, reader)
	}

	logger.Info("all kafka consumers started")
}

// Stop останавливает все консьюмеры
func (c *Consumer) Stop() error {
	const op = "kafka.consumer.Stop"

	logger := c.logger.With(
		zap.String("op", op),
	)

	logger.Info("stopping kafka consumers")

	// Закрываем все readers
	var lastErr error
	for topic, reader := range c.readers {
		if err := reader.Close(); err != nil {
			c.logger.Error("failed to close reader",
				zap.String("topic", topic),
				zap.Error(err),
			)
			lastErr = err
		}
	}

	// Ждем завершения всех горутин
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	// Таймаут для graceful shutdown
	select {
	case <-done:
		c.logger.Info("all consumers stopped gracefully")
	case <-time.After(30 * time.Second):
		c.logger.Warn("forced shutdown after timeout")
	}

	if lastErr != nil {
		return fmt.Errorf("%s: %w", op, lastErr)
	}

	return nil
}

// consume обрабатывает сообщения из конкретного топика
func (c *Consumer) consume(ctx context.Context, topic string, reader *kafka.Reader) {
	const op = "kafka.consumer.consume"

	defer c.wg.Done()

	logger := c.logger.With(
		zap.String("op", op),
		zap.String("topic", topic),
	)

	logger.Info("starting to consume topic")

	for {
		select {
		case <-ctx.Done():
			logger.Info("shutdown signal received")
			return
		default:
			message, err := reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					logger.Info("context cancelled")
					return
				}
				logger.Error("failed to fetch message", zap.Error(err))
				continue
			}

			if err := c.processMessage(ctx, topic, message); err != nil {
				logger.Error("failed to process message",
					zap.Error(err),
					zap.Int("partition", message.Partition),
					zap.Int64("offset", message.Offset),
				)
				continue
			}

			// Коммитим офсет
			if err := reader.CommitMessages(ctx, message); err != nil {
				logger.Error("failed to commit message",
					zap.Error(err),
					zap.Int("partition", message.Partition),
					zap.Int64("offset", message.Offset),
				)
			}
		}
	}
}

// processMessage обрабатывает отдельное сообщение
func (c *Consumer) processMessage(ctx context.Context, topic string, message kafka.Message) error {
	const op = "kafka.consumer.processMessage"

	logger := c.logger.With(
		zap.String("op", op),
		zap.String("topic", topic),
		zap.Int("partition", message.Partition),
		zap.Int64("offset", message.Offset),
		zap.String("key", string(message.Key)),
	)

	// Парсим сообщение в Event
	var event models.Event
	if err := json.Unmarshal(message.Value, &event); err != nil {
		logger.Error("failed to unmarshal event", zap.Error(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	logger.Debug("processing event",
		zap.String("event_id", event.ID.String()),
		zap.String("event_type", event.Type),
		zap.String("transaction_id", event.TransactionID.String()),
	)

	// Обрабатываем событие в зависимости от типа
	if err := c.routeEvent(ctx, &event); err != nil {
		logger.Error("failed to handle event",
			zap.String("event_type", event.Type),
			zap.Error(err),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	logger.Debug("event processed successfully")

	return nil
}

// routeEvent направляет событие соответствующему обработчику
func (c *Consumer) routeEvent(ctx context.Context, event *models.Event) error {
	switch event.Type {
	case models.EventTransactionResponse:
		return c.handler.HandleTransactionResponse(ctx, event)
	case models.EventFreezeResponse:
		return c.handler.HandleFreezeResponse(ctx, event)
	case models.EventReserveResponse:
		return c.handler.HandleReserveResponse(ctx, event)
	case models.EventExternalResponse:
		return c.handler.HandleExternalResponse(ctx, event)
	case models.EventWithdrawResponse:
		return c.handler.HandleWithdrawResponse(ctx, event)
	case models.EventDepositResponse:
		return c.handler.HandleDepositResponse(ctx, event)
	case models.EventUnfreezeResponse:
		return c.handler.HandleUnfreezeResponse(ctx, event)
	case models.EventUnreserveResponse:
		return c.handler.HandleUnreserveResponse(ctx, event)
	case models.EventExternalRollback:
		return c.handler.HandleExternalRollbackResponse(ctx, event)
	case models.EventRefundResponse:
		return c.handler.HandleRefundResponse(ctx, event)
	case models.EventBlockResponse:
		return c.handler.HandleBlockAccountResponse(ctx, event)
	default:
		return fmt.Errorf("unknown event type: %s", event.Type)
	}
}
