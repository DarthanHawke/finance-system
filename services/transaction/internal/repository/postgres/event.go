// Пакет postgres реализовывает работу с базами данных
package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// EventRepository структура релизующая методы для работы с событиями
type EventRepository struct {
	db     *Database
	logger *zap.Logger
}

func NewEventRepository(db *Database, logger *zap.Logger) *EventRepository {
	return &EventRepository{
		db:     db,
		logger: logger.With(zap.String("component", "transaction_service")),
	}
}

// CreateEvent создает событие
func (r *EventRepository) CreateEvent(ctx context.Context, req *models.CreateEventRequest) error {
	const op = "repository.event.CreateEvent"

	const query = `
		INSERT INTO events (id, transaction_id, partition_key, type, status, source, created_at, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		req.ID,
		req.TransactionID,
		req.PartitionKey,
		req.Type,
		req.Status,
		req.Source,
		req.CreatedAt,
		req.Payload,
	)

	if err != nil {
		if strings.Contains(err.Error(), "23505") {
			return apperr.ErrEventIDNotUnique
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetPendingEvents возвращает не завершненные события
func (r *EventRepository) GetPendingEvents(ctx context.Context, req *models.GetEventRequest) (models.GetEventResponse, error) {
	const op = "repository.event.GetPendingEvents"

	// Апдейтим processed_at, чтоб дргуие потоки не взяли эти же значения
	// и возвращам события
	const query = `
		UPDATE events 
		SET processed_at = $1
		WHERE id IN (
			SELECT id 
			FROM events 
			WHERE status = $2 
			AND (processed_at IS NULL OR processed_at < $3)
			ORDER BY created_at ASC 
			LIMIT $4
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, transaction_id, partition_key, type, status, source, 
			created_at, processed_at, payload
	`

	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)

	var events []models.Event
	err := r.db.SelectContext(ctx, &events, query,
		time.Now(),
		models.EventStatusPending,
		fiveMinutesAgo,
		req.Limit,
	)
	if err != nil {
		return models.GetEventResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	resp := models.GetEventResponse{Events: events}

	return resp, nil
}

// UpdateEventStatus обновляет статус события
func (r *EventRepository) UpdateEventStatus(ctx context.Context, req *models.UpdateEventStatusRequest) error {
	const op = "repository.event.UpdateEventStatus"

	const query = `
		UPDATE events 
		SET status = $1, 
			processed_at = $2
		WHERE id = $3
	`

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, query,
			req.Status,
			time.Now(),
			req.ID,
		)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		if rowsAffected == 0 {
			return fmt.Errorf("%s: expected more zero row affected, got %d", op, rowsAffected)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
