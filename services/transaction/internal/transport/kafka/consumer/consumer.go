// Пакет kafka предоставляет реализацию консьюмера для Kafka
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"
	"transaction-service/internal/repository/redis"

	"github.com/segmentio/kafka-go"
)

// Consumer реализует Kafka consumer для обработки событий
type Consumer struct {
	readers      map[string]*kafka.Reader
	handler      EventHandler
	deduplicator *redis.Deduplicator
	logger       *slog.Logger
	concurrency  int
	wg           sync.WaitGroup
}

// Config содержит конфигурацию для Kafka Consumer
type Config struct {
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
	config *Config,
	handler EventHandler,
	deduplicator *redis.Deduplicator,
	logger *slog.Logger,
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
		readers:      readers,
		handler:      handler,
		deduplicator: deduplicator,
		logger:       logger.With("component", "kafka_consumer"),
		concurrency:  config.Concurrency,
	}
}

// Run запускает консьюмеры для всех топиков
func (c *Consumer) Run(ctx context.Context) {
	const op = "consumer.Run"

	log := c.logger.With("op", op)
	log.Info("starting kafka consumers",
		"concurrency_per_topic", c.concurrency,
	)

	for topic, reader := range c.readers {
		c.wg.Add(1)
		go c.consume(ctx, topic, reader)
	}

	log.Info("all kafka consumers started")
}

// Stop останавливает все консьюмеры
func (c *Consumer) Stop() error {
	const op = "consumer.Stop"

	log := c.logger.With(
		"op", op,
	)

	log.Info("stopping kafka consumers")

	// Закрываем все readers
	var lastErr error
	for topic, reader := range c.readers {
		if err := reader.Close(); err != nil {
			log.Error("failed to close reader",
				"topic", topic,
				"error", err,
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
		log.Info("all consumers stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Warn("forced shutdown after timeout")
	}

	if lastErr != nil {
		return fmt.Errorf("%s: %w", op, lastErr)
	}

	return nil
}

// consume обрабатывает сообщения из конкретного топика
func (c *Consumer) consume(ctx context.Context, topic string, reader *kafka.Reader) {
	const op = "consumer.consume"

	defer c.wg.Done()

	log := c.logger.With(
		"op", op,
		"topic", topic,
	)

	log.Info("starting to consume topic")

	messages := make(chan kafka.Message, c.concurrency*2)

	// Запускаем пул воркеров для этого топика
	var workersWg sync.WaitGroup
	for i := 0; i < c.concurrency; i++ {
		workersWg.Add(1)
		go c.worker(ctx, topic, reader, messages, i, &workersWg)
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("shutting down consumer, waiting for workers")
			close(messages) // воркеры дообработают оставшиеся и выйдут
			workersWg.Wait()
			log.Info("consumer stopped")
			return
		default:
			message, err := reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					close(messages)
					workersWg.Wait()
					log.Info("consumer stopped")
					return
				}
				log.Error("failed to fetch message", "error", err)
				// TODO: metrics - consumer_fetch_error_total
				continue
			}

			select {
			case <-ctx.Done():
				close(messages)
				workersWg.Wait()
				return
			case messages <- message:
			}
		}
	}
}

// worker обрабатывает сообщения из канала
func (c *Consumer) worker(
	ctx context.Context,
	topic string,
	reader *kafka.Reader,
	messages <-chan kafka.Message,
	workerID int,
	wg *sync.WaitGroup,
) {
	const op = "consumer.worker"

	defer wg.Done()

	log := c.logger.With(
		"op", op,
		"topic", topic,
		"worker_id", workerID,
	)

	for message := range messages {
		err := c.processMessage(ctx, topic, message)
		if err != nil {
			log.Warn("message processing failed, will retry",
				"error", err,
				"partition", message.Partition,
				"offset", message.Offset,
			)
			// TODO: metrics - consumer_retryable_error_total
			continue
		}

		// Коммитим офсет
		if err := reader.CommitMessages(ctx, message); err != nil {
			log.Error("failed to commit message",
				"error", err,
				"partition", message.Partition,
				"offset", message.Offset,
			)
			// TODO: metrics - consumer_commit_error_total
		}
	}

	log.Debug("worker stopped")
}

// processMessage обрабатывает отдельное сообщение
func (c *Consumer) processMessage(ctx context.Context, topic string, message kafka.Message) error {
	const op = "consumer.processMessage"

	log := c.logger.With(
		"op", op,
		"topic", topic,
		"partition", message.Partition,
		"offset", message.Offset,
		"key", string(message.Key),
	)

	// Парсим сообщение в Event
	var event models.Event
	if err := json.Unmarshal(message.Value, &event); err != nil {
		log.Error("failed to unmarshal event",
			"error", err,
		)
		// TODO: metrics - consumer_corrupted_message_total
		return nil
	}

	log.Debug("processing event",
		"event_id", event.ID.String(),
		"event_type", event.Type,
		"transaction_id", event.TransactionID.String(),
	)

	isDuplicate, err := c.deduplicator.IsDuplicate(ctx, event.ID.String())
	if err != nil {
		// TODO: metrics - consumer_dedup_error_total
		c.logger.Warn("dedup check failed", "error", err)
	}
	if isDuplicate {
		return nil
	}

	// Обрабатываем событие в зависимости от типа
	if err := c.routeEvent(ctx, &event); err != nil {
		retryable, terminal := apperr.Classify(err)

		if terminal {
			log.Error("terminal error - committing offset, manual intervention required",
				"event_type", event.Type,
				"event_id", event.ID.String(),
				"error", err,
			)
			// TODO: metrics - consumer_terminal_error_total
			return nil
		}

		if !retryable {
			log.Warn("non-retryable error - committing offset",
				"event_type", event.Type,
				"event_id", event.ID.String(),
				"error", err,
			)
			// TODO: metrics - consumer_business_error_total
			return nil
		}

		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	log.Debug("event processed successfully")

	return nil
}

// routeEvent направляет событие соответствующему обработчику
func (c *Consumer) routeEvent(ctx context.Context, event *models.Event) error {
	switch event.Type {
	case models.EventTransactionRequest:
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
		return apperr.ErrUnknownEventType
	}
}
