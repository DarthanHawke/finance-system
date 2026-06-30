// kafka предоставляет реализацию Consumer/Producer для Kafka
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/lib/metrics"
	"transaction-service/internal/lib/tracing"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Producer реализует Kafka producer для обработки событий
type Producer struct {
	writer      *kafka.Writer
	dlqWriter   *kafka.Writer
	retryConfig *RetryConfig
	logger      *slog.Logger
	tracer      trace.Tracer
}

// Config содержит конфигурацию для Kafka Producer
type Config struct {
	Brokers          []string
	Topic            string
	BatchSize        int
	BatchTimeout     time.Duration
	RequiredAcks     int
	MaxAttempts      int
	WriteTimeout     time.Duration
	RebalanceTimeout time.Duration
}

// DLQConfig содержит конфигурацию для DLQ
type DLQConfig struct {
	Brokers      []string
	Topic        string
	BatchSize    int
	BatchTimeout time.Duration
	MaxAttempts  int
}

// RetryConfig содержит конфигурацию для retry
type RetryConfig struct {
	MaxAttempts int
	InitialWait time.Duration
	MaxWait     time.Duration
	Multiplier  float64
}

// NewProducer создает новый экземпляр Kafka Producer
func NewProducer(
	config *Config,
	dlqConfig *DLQConfig,
	retryConfig *RetryConfig,
	logger *slog.Logger,
) *Producer {
	return &Producer{
		writer:      createWriter(config.Brokers, "", config.BatchSize, config.BatchTimeout, config.RequiredAcks, config.MaxAttempts, config.WriteTimeout),
		dlqWriter:   createWriter(dlqConfig.Brokers, dlqConfig.Topic, dlqConfig.BatchSize, dlqConfig.BatchTimeout, 0, dlqConfig.MaxAttempts, 0),
		retryConfig: retryConfig,
		logger:      logger.With("component", "kafka_producer"),
		tracer:      otel.Tracer("kafka-producer"),
	}
}

// createWriter собирает kafka.Writer
func createWriter(brokers []string, topic string, batchSize int, batchTimeout time.Duration, requiredAcks, maxAttempts int, writeTimeout time.Duration) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		BatchSize:              batchSize,
		BatchTimeout:           batchTimeout,
		RequiredAcks:           kafka.RequiredAcks(requiredAcks),
		MaxAttempts:            maxAttempts,
		WriteTimeout:           writeTimeout,
		AllowAutoTopicCreation: false,
		Compression:            kafka.Snappy,
	}
}

// Produce публикует событие в Kafka
func (p *Producer) Produce(ctx context.Context, topic, key string, event any) error {
	const op = "producer.Produce"
	start := time.Now()
	defer func() {
		metrics.KafkaProduceDuration.Observe(time.Since(start).Seconds())
	}()

	log := tracing.WithTraceContext(ctx, p.logger.With(
		"op", op,
		"topic", topic,
		"key", key,
	))

	ctx, span := p.tracer.Start(ctx, "kafka.produce",
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", topic),
			attribute.String("messaging.kafka.message.key", key),
		),
	)
	defer span.End()

	eventBytes, err := json.Marshal(event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "marshal failed")
		log.Error("failed to marshal event", "error", err)
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	message := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: eventBytes,
		Time:  time.Now(),
		Headers: []kafka.Header{
			{
				Key:   "source",
				Value: []byte("transaction-service"),
			},
			{
				Key:   "content-type",
				Value: []byte("application/json"),
			},
		},
	}

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	for k, v := range carrier {
		message.Headers = append(message.Headers, kafka.Header{Key: k, Value: []byte(v)})
	}

	waitDuration := p.retryConfig.InitialWait

	for attempt := 1; attempt <= p.retryConfig.MaxAttempts; attempt++ {
		err = p.writer.WriteMessages(ctx, message)
		if err == nil {
			span.SetStatus(codes.Ok, "published")
			metrics.KafkaProduceTotal.WithLabelValues("success").Inc()
			return nil
		}

		// Если это последняя попытка, выходим
		if attempt == p.retryConfig.MaxAttempts {
			break
		}

		// Ждем перед следующей попыткой
		select {
		case <-ctx.Done():
			span.RecordError(ctx.Err())
			span.SetStatus(codes.Error, "context cancelled")
			return &apperr.WrappedError{
				Op:  op,
				Err: ctx.Err(),
			}
		case <-time.After(waitDuration):
			// Увеличиваем время ожидания для следующей попытки
			waitDuration = min(time.Duration(float64(waitDuration)*p.retryConfig.Multiplier), p.retryConfig.MaxWait)
		}
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, "all retries exhausted")

	log.Error("failed to publish message after all retry attempts",
		"op", op,
		"topic", topic,
		"key", key,
		"error", err,
		"max_attempts", p.retryConfig.MaxAttempts,
	)
	metrics.KafkaProduceTotal.WithLabelValues("error").Inc()

	return &apperr.WrappedError{
		Op:  op,
		Err: fmt.Errorf("failed after %d attempts: %w", p.retryConfig.MaxAttempts, err),
	}
}

// ProduceDLQ отправляет сообщение в DLQ
func (p *Producer) ProduceDLQ(ctx context.Context, key string, event any, eventError error) error {
	const op = "producer.ProduceDLQ"

	log := tracing.WithTraceContext(ctx, p.logger.With(
		"op", op,
		"key", key,
	))

	ctx, span := p.tracer.Start(ctx, "kafka.produce_dlq",
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", "dlq"),
			attribute.String("messaging.kafka.message.key", key),
		),
	)
	defer span.End()

	eventBytes, err := json.Marshal(event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "marshal failed")
		log.Error("failed to marshal event for DLQ", "error", err)
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	dlqMessage := kafka.Message{
		Key:   []byte(key),
		Value: eventBytes,
		Time:  time.Now(),
		Headers: []kafka.Header{
			{
				Key:   "source",
				Value: []byte("transaction-service"),
			},
			{
				Key:   "content-type",
				Value: []byte("application/json"),
			},
			{
				Key:   "dlq-timestamp",
				Value: []byte(time.Now().Format(time.RFC3339)),
			},
			{
				Key:   "error",
				Value: []byte(eventError.Error()),
			},
		},
	}

	err = p.dlqWriter.WriteMessages(ctx, dlqMessage)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "dlq write failed")
		log.Error("failed to publish message to DLQ",
			"op", op,
			"key", key,
			"error", err,
		)
		metrics.KafkaDLQSent.WithLabelValues("error").Inc()
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	span.SetStatus(codes.Ok, "published to DLQ")
	log.Warn("event published to DLQ",
		"op", op,
		"key", key,
		"error", eventError.Error(),
	)
	metrics.KafkaDLQSent.WithLabelValues("success").Inc()

	return nil
}

// Close закрывает соединение с Kafka
func (p *Producer) Close() error {
	var errs []error

	if p.writer != nil {
		if err := p.writer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close main writer: %w", err))
		}
	}

	if p.dlqWriter != nil {
		if err := p.dlqWriter.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close DLQ writer: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing producer: %v", errs)
	}

	return nil
}
