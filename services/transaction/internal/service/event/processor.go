// service
package service

import (
	"context"
	"fmt"
	"time"
	"transaction-service/internal/models"

	"go.uber.org/zap"
)

type OutboxProcessor struct {
	outboxManager OutboxManager
	producer      Producer
	logger        *zap.Logger
	batchSize     int
	handlePeriod  time.Duration
}

type Config struct {
	BatchSize    int
	handlePeriod time.Duration
}

type Producer interface {
	Produce(ctx context.Context, topic string, key string, event any) error
	ProduceDLQ(ctx context.Context, key string, event any, err error) error
}

type OutboxManager interface {
	GetPendingEvents(ctx context.Context, req *models.GetEventRequest) (models.GetEventResponse, error)
	UpdateEventStatus(ctx context.Context, req *models.UpdateEventStatusRequest) error
}

func NewOutboxProcessor(
	outboxManager OutboxManager,
	producer Producer,
	logger *zap.Logger,
	config Config,
) *OutboxProcessor {
	return &OutboxProcessor{
		outboxManager: outboxManager,
		producer:      producer,
		logger:        logger.With(zap.String("component", "outbox_processor")),
		batchSize:     config.BatchSize,
		handlePeriod:  config.handlePeriod,
	}
}

// Start запускает обработчик outbox
func (p *OutboxProcessor) StartProcessEvents(ctx context.Context) {
	const op = "outbox.processor.StartProcessEvents"

	logger := p.logger.With(zap.String("op", op))

	logger.Info("starting outbox processor")

	ticker := time.NewTicker(p.handlePeriod)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("outbox processor context cancelled")
				return
			case <-ticker.C:
				p.processPendingEvents(ctx)
			}
		}
	}()
}

// processPendingEvents обрабатывает pending события
func (p *OutboxProcessor) processPendingEvents(ctx context.Context) {
	const op = "outbox.processor.processPendingEvents"

	logger := p.logger.With(zap.String("op", op))

	events, err := p.outboxManager.GetPendingEvents(ctx, &models.GetEventRequest{Limit: p.batchSize})
	if err != nil {
		logger.Error("failed to get pending events", zap.Error(err))
		return
	}

	if len(events.Events) == 0 {
		return
	}

	logger.Info("processing pending events", zap.Int("count", len(events.Events)))

	for _, event := range events.Events {
		if err := p.processEvent(ctx, event); err != nil {
			logger.Error("failed to process event",
				zap.String("event_id", event.ID.String()),
				zap.Error(err),
			)
		}
	}
}

// processEvent обрабатывает отдельное событие
func (p *OutboxProcessor) processEvent(ctx context.Context, event models.Event) error {
	const op = "outbox.processor.processEvent"

	logger := p.logger.With(
		zap.String("op", op),
		zap.String("event_id", event.ID.String()),
	)

	// Ключ для партиционирования - id для гарантии порядка
	messageKey := fmt.Sprintf("%s:%s", event.Type, event.ID)
	topic := models.TopicMap[event.Type]

	if err := p.producer.Produce(ctx, topic, messageKey, event); err != nil {
		if time.Since(event.CreatedAt) > 8*time.Hour {
			if err := p.producer.ProduceDLQ(ctx, messageKey, event, err); err != nil {
				p.logger.Warn("failed to publish event to DLQ, will retry",
					zap.Error(err),
				)
				return fmt.Errorf("%s: %w", op, err)
			}
		} else {
			p.logger.Warn("failed to publish event, will retry",
				zap.Error(err),
			)
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	if err := p.outboxManager.UpdateEventStatus(
		ctx,
		&models.UpdateEventStatusRequest{
			ID:     event.ID,
			Status: models.EventStatusCompleted,
		},
	); err != nil {
		return fmt.Errorf("%s: failed to update event status: %w", op, err)
	}

	logger.Debug("event published successfully",
		zap.String("topic", topic),
	)

	return nil
}
