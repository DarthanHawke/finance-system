package kafka

import (
	"account-service/internal/config"
	"account-service/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
	"go.uber.org/zap"
)

// Producer продюсер с гарантией доставки
type Producer struct {
	writer     *kafka.Writer
	dlqWriter  *kafka.Writer
	config     *config.KafkaConfig
	logger     *zap.Logger
	maxRetries int
}

// NewProducer создает новый продюсер
func NewProducer(config *config.KafkaConfig, logger *zap.Logger) *Producer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Balancer:     &kafka.LeastBytes{},
		MaxAttempts:  config.MaxRetries,
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		Compression:  compress.Snappy,
		RequiredAcks: kafka.RequireAll, // Гарантия доставки
		Async:        false,            // Синхронная отправка для гарантии
	}

	dlqW := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		MaxAttempts:  config.MaxRetries,
		RequiredAcks: kafka.RequireAll,
	}

	return &Producer{
		writer:     w,
		dlqWriter:  dlqW,
		config:     config,
		logger:     logger.With(zap.String("component", "kafka_producer")),
		maxRetries: config.MaxRetries,
	}
}

// SendMessage отправляет сообщение с гарантией доставки
func (p *Producer) SendMessage(ctx context.Context, topic string, messageType string, payload interface{}) error {
	const op = "kafka.producer.SendMessage"

	message := models.MessageWrapper{
		ID:         uuid.New().String(),
		Type:       messageType,
		Source:     "account-service",
		Timestamp:  time.Now(),
		Payload:    payload,
		RetryCount: 0,
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("%s: failed to marshal message: %w", op, err)
	}

	kafkaMessage := kafka.Message{
		Topic: topic,
		Key:   []byte(message.ID),
		Value: data,
		Headers: []kafka.Header{
			{Key: "message-id", Value: []byte(message.ID)},
			{Key: "message-type", Value: []byte(messageType)},
			{Key: "source", Value: []byte(message.Source)},
		},
		Time: time.Now(),
	}

	for attempt := 0; attempt < p.maxRetries; attempt++ {
		err = p.writer.WriteMessages(ctx, kafkaMessage)
		if err == nil {
			p.logger.Debug("message sent successfully",
				zap.String("topic", topic),
				zap.String("message_id", message.ID),
				zap.String("type", messageType),
			)
			return nil
		}

		p.logger.Warn("failed to send message, retrying",
			zap.String("op", op),
			zap.String("topic", topic),
			zap.String("message_id", message.ID),
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)

		if attempt < p.maxRetries-1 {
			time.Sleep(p.config.RetryInterval * time.Duration(attempt+1))
		}
	}

	// Если все попытки неудачны, отправляем в DLQ
	dlqErr := p.sendToDLQ(ctx, topic, message, err)
	if dlqErr != nil {
		p.logger.Error("failed to send message to DLQ",
			zap.String("op", op),
			zap.String("topic", topic),
			zap.String("message_id", message.ID),
			zap.Error(dlqErr),
		)
	}

	return fmt.Errorf("%s: failed to send message after %d attempts: %w", op, p.maxRetries, err)
}
