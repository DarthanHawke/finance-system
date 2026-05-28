// kafka
package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"
)

type OutboxProcessor struct {
	outboxManager OutboxManager
	producer      Producer
	logger        *slog.Logger
	batchSize     int
	handlePeriod  time.Duration
	concurrency   int
	eventsCh      chan models.Event
	wg            sync.WaitGroup
}

type Config struct {
	BatchSize    int
	HandlePeriod time.Duration
	Concurrency  int
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
	logger *slog.Logger,
	config *Config,
) *OutboxProcessor {
	return &OutboxProcessor{
		outboxManager: outboxManager,
		producer:      producer,
		logger:        logger.With("component", "outbox_processor"),
		batchSize:     config.BatchSize,
		handlePeriod:  config.HandlePeriod,
		concurrency:   config.Concurrency,
		eventsCh:      make(chan models.Event, config.Concurrency*2),
	}
}

// Start запускает обработчик outbox
func (p *OutboxProcessor) Run(ctx context.Context) {
	const op = "processor.Run"

	log := p.logger.With("op", op)
	log.Info("starting outbox processor",
		"concurrency", p.concurrency,
	)

	for i := 0; i < p.concurrency; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	ticker := time.NewTicker(p.handlePeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("outbox processor stopping, waiting for workers")
			close(p.eventsCh)
			p.wg.Wait()
			log.Info("outbox processor context cancelled")
			return
		case <-ticker.C:
			p.fetchAndDispatch(ctx)
		}
	}
}

// fetchAndDispatch получает события из БД и отправляет в канал воркерам
func (p *OutboxProcessor) fetchAndDispatch(ctx context.Context) {
	const op = "processor.fetchAndDispatch"

	log := p.logger.With("op", op)

	events, err := p.outboxManager.GetPendingEvents(ctx, &models.GetEventRequest{Limit: p.batchSize})
	if err != nil {
		log.Error("failed to get pending events", "error", err)
		// TODO: metrics - outbox_get_events_error_total (alert: >0 за 5 мин → critical)
		return
	}

	if len(events.Events) == 0 {
		return
	}

	log.Debug("dispatching events to workers",
		"count", len(events.Events),
	)

	for _, event := range events.Events {
		select {
		case <-ctx.Done():
			return
		case p.eventsCh <- event:
		}
	}
}

// worker обрабатывает события из канала
func (p *OutboxProcessor) worker(ctx context.Context, workerID int) {
	defer p.wg.Done()

	log := p.logger.With(
		"op", "processor.worker",
		"worker_id", workerID,
	)

	for event := range p.eventsCh {
		if err := p.processEvent(ctx, event); err != nil {
			// TODO: metrics - outbox_process_event_error_total
		}
	}

	log.Debug("worker stopped")
}

// processEvent обрабатывает отдельное событие
func (p *OutboxProcessor) processEvent(ctx context.Context, event models.Event) error {
	const op = "processor.processEvent"

	log := p.logger.With(
		"op", "processor.worker",
	)

	// Ключ для партиционирования - хэш номера счета для гарантии порядка сообщения внутри одного счета
	messageKey := event.PartitionKey
	topic := models.TopicMap[event.Type]

	err := p.producer.Produce(ctx, topic, messageKey, event)
	if err != nil {
		retryable, isTerminal := apperr.Classify(err)

		if isTerminal {
			// TODO: metrics - outbox_terminal_error_total
			log.Error("TERMINAL ERROR - sending to DLQ immediately",
				"op", op,
				"event_id", event.ID.String(),
				"error", err,
			)
			if err := p.producer.ProduceDLQ(ctx, messageKey, event, err); err != nil {
				return &apperr.WrappedError{
					Op:  op,
					Err: err,
				}
			}
			// TODO: metrics - outbox_dlq_sent_total
			if err := p.outboxManager.UpdateEventStatus(
				ctx,
				&models.UpdateEventStatusRequest{
					ID:     event.ID,
					Status: models.EventStatusCompleted,
				},
			); err != nil {
				return &apperr.WrappedError{
					Op:  op,
					Err: fmt.Errorf("update event status after DLQ: %w", err),
				}
			}
			return nil
		}

		if !retryable {
			// TODO: metrics - outbox_business_error_total
			log.Warn("non-retryable error, sending to DLQ",
				"op", op,
				"event_id", event.ID.String(),
				"error", err,
			)
			if err := p.producer.ProduceDLQ(ctx, messageKey, event, err); err != nil {
				return &apperr.WrappedError{
					Op:  op,
					Err: err,
				}
			}
			// TODO: metrics - outbox_dlq_sent_total
			if err := p.outboxManager.UpdateEventStatus(
				ctx,
				&models.UpdateEventStatusRequest{
					ID:     event.ID,
					Status: models.EventStatusCompleted,
				},
			); err != nil {
				return &apperr.WrappedError{
					Op:  op,
					Err: fmt.Errorf("update event status after DLQ: %w", err),
				}
			}
			return nil
		}

		if time.Since(event.CreatedAt) > 8*time.Hour {
			// TODO: metrics - outbox_expired_total
			log.Warn("event expired, sending to DLQ",
				"op", op,
				"event_id", event.ID.String(),
				"age", time.Since(event.CreatedAt),
			)
			if err := p.producer.ProduceDLQ(ctx, messageKey, event, err); err != nil {
				return &apperr.WrappedError{
					Op:  op,
					Err: err,
				}
			}
			// TODO: metrics - outbox_dlq_sent_total
			if err := p.outboxManager.UpdateEventStatus(
				ctx,
				&models.UpdateEventStatusRequest{
					ID:     event.ID,
					Status: models.EventStatusCompleted,
				},
			); err != nil {
				return &apperr.WrappedError{
					Op:  op,
					Err: fmt.Errorf("update event status after DLQ: %w", err),
				}
			}
			return nil
		}

		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	if err := p.outboxManager.UpdateEventStatus(
		ctx,
		&models.UpdateEventStatusRequest{
			ID:     event.ID,
			Status: models.EventStatusCompleted,
		},
	); err != nil {
		return &apperr.WrappedError{
			Op:  op,
			Err: fmt.Errorf("update event status: %w", err),
		}
	}

	return nil
}
