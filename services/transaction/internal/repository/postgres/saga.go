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

// SagaRepository реализует методы для работы с сагой
type SagaRepository struct {
	db *Database
}

// NewSagaRepository создает обертку над бд, реализуюзую методы для работы с таблицей saga
func NewSagaRepository(db *Database) *SagaRepository {
	return &SagaRepository{
		db: db,
	}
}

// InsertSagaStep создает шаг саги
func (r *SagaRepository) InsertSagaStep(
	ctx context.Context,
	tx *sqlx.Tx,
	req *models.InsertSagaStepRequest,
) error {
	const op = "saga.CreateSagaStep"

	const querySagaStep = `
		INSERT INTO saga_steps (id, transaction_id, event_id, step_name, step_kind, saga_status, created_at, updated_at)
		VALUES (:id, :transaction_id, :event_id, :step_name, :step_kind, :saga_status, :created_at, :updated_at)
		ON CONFLICT (transaction_id, step_name, step_kind) DO NOTHING
	`

	_, err := tx.NamedExecContext(ctx, querySagaStep, req)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return apperr.ErrSagaStepAlreadyExists
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return nil
}
