// kafka предоставляет реализацию Producer для Kafka
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Producer struct {
	writer      *kafka.Writer
	dlqWriter   *kafka.Writer
	retryConfig *RetryConfig
	logger      *zap.Logger
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
	logger *zap.Logger,
) *Producer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(config.Brokers...),
		Topic:                  config.Topic,
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
		logger:      logger.With(zap.String("component", "kafka_producer")),
	}
}

// Produce публикует событие в Kafka
func (p *Producer) Produce(ctx context.Context, topic, key string, event any) error {
	const op = "kafka.producer.Produce"

	logger := p.logger.With(
		zap.String("op", op),
		zap.String("topic", topic),
		zap.String("key", key),
	)

	eventBytes, err := json.Marshal(event)
	if err != nil {
		logger.Error("failed to marshal event", zap.Error(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	message := kafka.Message{
		Key:   []byte(key),
		Value: eventBytes,
		Time:  time.Now(),
		Headers: []kafka.Header{
			{
				Key:   "source",
				Value: []byte("account-service"),
			},
			{
				Key:   "content-type",
				Value: []byte("application/json"),
			},
		},
	}

	var lastErr error
	waitDuration := p.retryConfig.InitialWait

	for attempt := 1; attempt <= p.retryConfig.MaxAttempts; attempt++ {
		err = p.writer.WriteMessages(ctx, message)
		if err == nil {
			logger.Debug("event published successfully",
				zap.Int("message_size", len(message.Value)),
			)
			return nil
		}

		lastErr = err
		logger.Warn("failed to publish message, will retry",
			zap.Error(err),
			zap.ByteString("message_key", message.Key),
			zap.Duration("wait_time", waitDuration),
		)

		// Если это последняя попытка, выходим
		if attempt == p.retryConfig.MaxAttempts {
			break
		}

		// Ждем перед следующей попыткой
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s: %w", op, ctx.Err())
		case <-time.After(waitDuration):
			// Увеличиваем время ожидания для следующей попытки
			waitDuration = min(time.Duration(float64(waitDuration)*p.retryConfig.Multiplier), p.retryConfig.MaxWait)
		}
	}

	logger.Error("failed to publish message after all retry attempts",
		zap.Error(lastErr),
		zap.Int("max_attempts", p.retryConfig.MaxAttempts),
		zap.ByteString("message_key", message.Key),
	)

	return fmt.Errorf("%s: failed after %d attempts: %w", op, p.retryConfig.MaxAttempts, lastErr)
}

// ProduceDLQ отправляет сообщение в DLQ
func (p *Producer) ProduceDLQ(ctx context.Context, key string, event any, eventError error) error {
	const op = "kafka.producer.ProduceDLQ"

	logger := p.logger.With(
		zap.String("op", op),
		zap.String("key", key),
	)

	eventBytes, err := json.Marshal(event)
	if err != nil {
		logger.Error("failed to marshal event for DLQ", zap.Error(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	dlqMessage := kafka.Message{
		Key:   []byte(key),
		Value: eventBytes,
		Time:  time.Now(),
		Headers: []kafka.Header{
			{
				Key:   "source",
				Value: []byte("account-service"),
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
		logger.Error("failed to publish message to DLQ",
			zap.Error(err),
			zap.ByteString("message_key", dlqMessage.Key),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	logger.Warn("event published to DLQ successfully",
		zap.Int("message_size", len(dlqMessage.Value)),
		zap.String("original_error", eventError.Error()),
	)

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
