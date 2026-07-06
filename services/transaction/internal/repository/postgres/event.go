// Пакет postgres реализовывает работу с PostgreSQL
package postgres

import (
	"context"
	"errors"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

// EventRepository реализует методы для работы с событиями
type EventRepository struct {
	db *Database
}

// NewEventRepository создает обертку над бд, реализуюзую методы для работы с таблицей events
func NewEventRepository(db *Database) *EventRepository {
	return &EventRepository{
		db: db,
	}
}

// InsertEvent создает событие
func (r *EventRepository) InsertEvent(
	ctx context.Context,
	tx *sqlx.Tx,
	req *models.InsertEventRequest,
) error {
	const op = "event.InsertEvent"

	const queryEvent = `
		INSERT INTO events (id, transaction_id, partition_key, event_type, 
		step_name, event_status, source, trace_id, span_id, created_at, payload)
		VALUES (:id, :transaction_id, :partition_key, :event_type, 
		:step_name, :event_status, :source, :trace_id, :span_id, :created_at, :payload)
		ON CONFLICT (transaction_id, event_type, step_name) DO NOTHING
	`

	_, err := tx.NamedExecContext(ctx, queryEvent, req)

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

// GetPendingEvents возвращает незавершненные события
func (r *EventRepository) GetPendingEvents(ctx context.Context, req *models.GetEventRequest) (models.GetEventResponse, error) {
	const op = "event.GetPendingEvents"

	const query = `
		UPDATE events 
		SET processed_at = $1
		WHERE id IN (
			SELECT id 
			FROM events 
			WHERE event_status = $2 
			AND (processed_at IS NULL OR processed_at < $3)
			ORDER BY created_at ASC 
			LIMIT $4
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, transaction_id, partition_key, event_type, event_status, source, 
			trace_id, span_id, created_at, processed_at, payload
	`

	var events []models.Event
	err := r.db.SelectContext(ctx, &events, query,
		req.ProcessedAt,
		models.EventStatusPending,
		req.FiveMinutesAgo,
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
		SET event_status = $1, 
			processed_at = $2
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, req.Status, req.ProcessedAt, req.ID)
	if err != nil {
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	if rowsAffected == 0 {
		return apperr.ErrEventNotFound
	}

	return nil
}
