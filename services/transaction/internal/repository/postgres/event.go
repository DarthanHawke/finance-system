// Пакет postgres реализовывает работу с базами данных
package postgres

import (
	"context"
	"errors"
	"time"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

// EventRepository структура релизующая методы для работы с событиями
type EventRepository struct {
	db *Database
}

func NewEventRepository(db *Database) *EventRepository {
	return &EventRepository{
		db: db,
	}
}

// CreateEvent создает событие
func (r *EventRepository) CreateEvent(ctx context.Context, req *models.CreateEventRequest) error {
	const op = "event.CreateEvent"

	const query = `
		INSERT INTO events (id, transaction_id, partition_key, type, status, source, payload)
		VALUES (:id, :transaction_id, :partition_key, :type, :status, :source, :payload)
	`

	_, err := r.db.NamedExecContext(ctx, query, req)

	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return apperr.ErrEventIDNotUnique
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return nil
}

// GetPendingEvents возвращает не завершненные события
func (r *EventRepository) GetPendingEvents(ctx context.Context, req *models.GetEventRequest) (models.GetEventResponse, error) {
	const op = "event.GetPendingEvents"

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
		return models.GetEventResponse{}, &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}
	if len(events) == 0 {
		return models.GetEventResponse{}, nil
	}

	resp := models.GetEventResponse{Events: events}

	return resp, nil
}

// UpdateEventStatus обновляет статус события
func (r *EventRepository) UpdateEventStatus(ctx context.Context, req *models.UpdateEventStatusRequest) error {
	const op = "event.UpdateEventStatus"

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
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			return apperr.ErrEventNotFound
		}

		return nil
	})
	if err != nil {
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return nil
}
