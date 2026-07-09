// Пакет kafka предоставляет реализацию Consumer/Producer для Kafka
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/lib/metrics"
	"transaction-service/internal/lib/tracing"
	"transaction-service/internal/models"
	"transaction-service/internal/repository/redis"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Consumer реализует Kafka consumer для обработки событий
type Consumer struct {
	readers      map[string]*kafka.Reader
	handler      EventHandler
	deduplicator *redis.Deduplicator
	logger       *slog.Logger
	tracer       trace.Tracer
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
	HandleTransactionCreate(ctx context.Context, event *models.Event) error
	HandleResponse(ctx context.Context, event *models.Event) error
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
		tracer:       otel.Tracer("kafka-consumer"),
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
				metrics.ConsumerFetchErrors.Inc()
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

	log := tracing.WithTraceContext(ctx, c.logger.With(
		"op", op,
		"topic", topic,
		"worker_id", workerID,
	))

	for message := range messages {
		err := c.processMessage(ctx, topic, message)
		if err != nil {
			log.Warn("message processing failed, will retry",
				"error", err,
				"partition", message.Partition,
				"offset", message.Offset,
			)
			metrics.ConsumerRetryableErrors.Inc()
			continue
		}

		// Коммитим офсет
		if err := reader.CommitMessages(ctx, message); err != nil {
			log.Error("failed to commit message",
				"error", err,
				"partition", message.Partition,
				"offset", message.Offset,
			)
			metrics.ConsumerCommitErrors.Inc()
		}
	}

	log.Debug("worker stopped")
}

// processMessage обрабатывает отдельное сообщение
func (c *Consumer) processMessage(ctx context.Context, topic string, message kafka.Message) error {
	const op = "consumer.processMessage"
	start := time.Now()
	defer func() {
		metrics.ConsumerMessageDuration.Observe(time.Since(start).Seconds())
	}()

	log := tracing.WithTraceContext(ctx, c.logger.With(
		"op", op,
		"topic", topic,
		"partition", message.Partition,
		"offset", message.Offset,
		"key", string(message.Key),
	))

	carrier := propagation.MapCarrier{}
	for _, h := range message.Headers {
		carrier[h.Key] = string(h.Value)
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	ctx, span := c.tracer.Start(ctx, "kafka.consume",
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", topic),
			attribute.String("messaging.kafka.message.key", string(message.Key)),
			attribute.Int("messaging.kafka.partition", message.Partition),
			attribute.Int64("messaging.kafka.offset", message.Offset),
		),
	)
	defer span.End()

	// Парсим сообщение в Event
	var event models.Event
	if err := json.Unmarshal(message.Value, &event); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "unmarshal failed")
		log.Error("failed to unmarshal event",
			"error", err,
		)
		metrics.ConsumerMessagesProcessed.WithLabelValues("corrupted").Inc()
		return nil
	}

	span.SetAttributes(
		attribute.String("event_id", event.ID.String()),
		attribute.String("event_type", event.Type),
		attribute.String("transaction_id", event.TransactionID.String()),
	)

	log.Debug("processing event",
		"event_id", event.ID.String(),
		"event_type", event.Type,
		"transaction_id", event.TransactionID.String(),
	)

	isDuplicate, err := c.deduplicator.IsDuplicate(ctx, event.ID.String())
	if err != nil {
		metrics.ConsumerDedupErrors.Inc()
		c.logger.Warn("dedup check failed", "error", err)
	}
	if isDuplicate {
		span.SetStatus(codes.Ok, "duplicate skipped")
		metrics.ConsumerMessagesProcessed.WithLabelValues("duplicate").Inc()
		return nil
	}

	// Обрабатываем событие в зависимости от типа
	if err := c.routeEvent(ctx, &event); err != nil {
		retryable, terminal := apperr.Classify(err)

		if terminal {
			span.RecordError(err)
			span.SetStatus(codes.Error, "terminal error")
			metrics.ConsumerMessagesProcessed.WithLabelValues("terminal_error").Inc()
			log.Error("terminal error - committing offset, manual intervention required",
				"event_type", event.Type,
				"event_id", event.ID.String(),
				"error", err,
			)
			return nil
		}

		if !retryable {
			span.RecordError(err)
			span.SetStatus(codes.Error, "non-retryable error")
			metrics.ConsumerMessagesProcessed.WithLabelValues("business_error").Inc()
			log.Warn("non-retryable error - committing offset",
				"event_type", event.Type,
				"event_id", event.ID.String(),
				"error", err,
			)
			return nil
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "retryable error")
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	span.SetStatus(codes.Ok, "processed")
	log.Debug("event processed successfully")
	metrics.ConsumerMessagesProcessed.WithLabelValues("success").Inc()
	return nil
}

// routeEvent направляет событие соответствующему обработчику
func (c *Consumer) routeEvent(ctx context.Context, event *models.Event) error {
	switch event.Type {
	case models.EventTransactionCreate:
		return c.handler.HandleTransactionCreate(ctx, event)
	case models.EventBlockResponse:
		return c.handler.HandleBlockAccountResponse(ctx, event)
	case models.EventFreezeResponse,
		models.EventCaptureResponse,
		models.EventCreditResponse,
		models.EventDebitResponse,
		models.EventUnfreezeResponse,
		models.EventPaymentResponse,
		models.EventCommitResponse,
		models.EventRollbackResponse,
		models.EventFXActivateResponse,
		models.EventFXCommitResponse,
		models.EventFXCancelResponse:
		return c.handler.HandleResponse(ctx, event)
	default:
		return apperr.ErrUnknownEventType
	}
}
