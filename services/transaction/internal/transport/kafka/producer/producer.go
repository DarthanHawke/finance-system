// kafka предоставляет реализацию Producer для Kafka
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
	"transaction-service/internal/lib/errors/apperr"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer      *kafka.Writer
	dlqWriter   *kafka.Writer
	retryConfig *RetryConfig
	logger      *slog.Logger
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
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(config.Brokers...),
		Balancer:               &kafka.Hash{},
		BatchSize:              config.BatchSize,
		BatchTimeout:           config.BatchTimeout,
		RequiredAcks:           kafka.RequiredAcks(config.RequiredAcks),
		MaxAttempts:            config.MaxAttempts,
		WriteTimeout:           config.WriteTimeout,
		AllowAutoTopicCreation: true,
		Compression:            kafka.Snappy,
	}

	dlqWriter := &kafka.Writer{
		Addr:                   kafka.TCP(dlqConfig.Brokers...),
		Topic:                  dlqConfig.Topic,
		Balancer:               &kafka.Hash{},
		BatchSize:              dlqConfig.BatchSize,
		BatchTimeout:           dlqConfig.BatchTimeout,
		MaxAttempts:            dlqConfig.MaxAttempts,
		AllowAutoTopicCreation: true,
		Compression:            kafka.Snappy,
	}

	return &Producer{
		writer:      writer,
		dlqWriter:   dlqWriter,
		retryConfig: retryConfig,
		logger:      logger.With("component", "kafka_producer"),
	}
}

// Produce публикует событие в Kafka
func (p *Producer) Produce(ctx context.Context, topic, key string, event any) error {
	const op = "producer.Produce"

	log := p.logger.With(
		"op", op,
		"topic", topic,
		"key", key,
	)

	eventBytes, err := json.Marshal(event)
	if err != nil {
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

	waitDuration := p.retryConfig.InitialWait

	for attempt := 1; attempt <= p.retryConfig.MaxAttempts; attempt++ {
		err = p.writer.WriteMessages(ctx, message)
		if err == nil {
			// TODO: metrics - kafka_produce_success_total
			return nil
		}

		// Если это последняя попытка, выходим
		if attempt == p.retryConfig.MaxAttempts {
			break
		}

		// Ждем перед следующей попыткой
		select {
		case <-ctx.Done():
			return &apperr.WrappedError{
				Op:  op,
				Err: ctx.Err(),
			}
		case <-time.After(waitDuration):
			// Увеличиваем время ожидания для следующей попытки
			waitDuration = min(time.Duration(float64(waitDuration)*p.retryConfig.Multiplier), p.retryConfig.MaxWait)
		}
	}

	log.Error("failed to publish message after all retry attempts",
		"op", op,
		"topic", topic,
		"key", key,
		"error", err,
		"max_attempts", p.retryConfig.MaxAttempts,
	)
	// TODO: metrics - kafka_produce_error_total

	return &apperr.WrappedError{
		Op:  op,
		Err: fmt.Errorf("failed after %d attempts: %w", p.retryConfig.MaxAttempts, err),
	}
}

// ProduceDLQ отправляет сообщение в DLQ
func (p *Producer) ProduceDLQ(ctx context.Context, key string, event any, eventError error) error {
	const op = "producer.ProduceDLQ"

	log := p.logger.With(
		"op", op,
		"key", key,
	)

	eventBytes, err := json.Marshal(event)
	if err != nil {
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
		log.Error("failed to publish message to DLQ",
			"op", op,
			"key", key,
			"error", err,
		)
		// TODO: metrics - kafka_dlq_error_total
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	log.Warn("event published to DLQ",
		"op", op,
		"key", key,
		"error", eventError.Error(),
	)
	// TODO: metrics - kafka_dlq_sent_total

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
