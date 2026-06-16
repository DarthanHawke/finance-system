// Пакет postgres реализовывает работу с базами данных
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

// SagaRepository структура релизующая методы для работы с событиями
type SagaRepository struct {
	db *Database
}

func NewSagaRepository(db *Database) *SagaRepository {
	return &SagaRepository{
		db: db,
	}
}

// CreateSagaStep создает шаг саги
func (r *SagaRepository) CreateSagaStep(ctx context.Context, req *models.CreateSagaStepRequest) error {
	const op = "saga.CreateSagaStep"

	const query = `
		INSERT INTO saga_steps (id, transaction_id, event_id, step_name, step_kind, status)
		VALUES (:id, :transaction_id, :event_id, :step_name, :step_kind, :status)
	`

	_, err := r.db.NamedExecContext(ctx, query, req)
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

// GetSagaStep получает шаг саги по ID транзакции, имени шага и типу
func (r *SagaRepository) GetSagaStep(ctx context.Context, req *models.GetSagaStepRequest) (models.GetSagaStepResponse, error) {
	const op = "saga.GetSagaStep"

	const query = `
		SELECT id, transaction_id, event_id, step_name, step_kind, status, created_at, updated_at
		FROM saga_steps
		WHERE transaction_id = $1 AND step_name = $2 AND step_kind = $3
	`

	var step models.SagaStep
	err := r.db.GetContext(ctx, &step, query, req.TransactionID, req.StepName, req.StepKind)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.GetSagaStepResponse{}, apperr.ErrSagaStepNotFound
		}
		return models.GetSagaStepResponse{}, &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return models.GetSagaStepResponse{Step: &step}, nil
}

// GetSagaStepsByTransaction получает все шаги саги для транзакции
func (r *SagaRepository) GetSagaStepsByTransaction(ctx context.Context, req *models.GetSagaStepsByTransactionRequest) (models.GetSagaStepsByTransactionResponse, error) {
	const op = "saga.GetSagaStepsByTransaction"

	const query = `
		SELECT id, transaction_id, event_id, step_name, step_kind, status, created_at, updated_at
		FROM saga_steps
		WHERE transaction_id = $1
		ORDER BY created_at ASC
	`

	var steps []models.SagaStep
	err := r.db.SelectContext(ctx, &steps, query, req.TransactionID)
	if err != nil {
		return models.GetSagaStepsByTransactionResponse{}, &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return models.GetSagaStepsByTransactionResponse{Steps: steps}, nil
}

// UpdateSagaStepStatus обновляет статус шага саги
func (r *SagaRepository) UpdateSagaStepStatus(ctx context.Context, req *models.UpdateSagaStepStatusRequest) error {
	const op = "saga.UpdateSagaStepStatus"

	query := `
		UPDATE saga_steps 
		SET status = :status, 
		    event_id = COALESCE(:event_id, event_id),
		    updated_at = :updated_at
		WHERE id = :id
	`

	params := map[string]interface{}{
		"id":         req.ID,
		"status":     req.Status,
		"event_id":   req.EventID,
		"updated_at": time.Now(),
	}

	result, err := r.db.NamedExecContext(ctx, query, params)
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
		return apperr.ErrSagaStepNotFound
	}

	return nil
}

// UpdateSagaState обновляет состояние саги и опционально статус транзакции
func (r *SagaRepository) UpdateSagaState(ctx context.Context, req *models.UpdateSagaStateRequest) error {
	const op = "saga.UpdateSagaState"

	query := `UPDATE transactions 
			   SET saga_state = $1, 
			       status = COALESCE($2, status),
			       updated_at = $3
			   WHERE id = $4`

	result, err := r.db.ExecContext(ctx, query, req.SagaState, req.Status, time.Now(), req.ID)
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
		return apperr.ErrTransactionNotFound
	}

	return nil
}

// Advance выполняет атомарное продвижение саги: обновляет состояние, создает шаг и событие
func (r *SagaRepository) Advance(ctx context.Context, req *models.AdvanceRequest) error {
	const op = "saga.Advance"

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		const updateQuery = `
			UPDATE transactions
			SET saga_state = :new_state,
			    status     = COALESCE(:new_status, status),
			    updated_at = :updated_at
			WHERE id = :transaction_id
			  AND saga_state = :expected_from`

		params := map[string]interface{}{
			"new_state":      req.NewState,
			"new_status":     req.NewStatus,
			"updated_at":     time.Now(),
			"transaction_id": req.TransactionID,
			"expected_from":  req.ExpectedFrom,
		}

		result, err := tx.NamedExecContext(ctx, updateQuery, params)
		if err != nil {
			return fmt.Errorf("update saga state: %w", err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			var current string
			errSel := tx.GetContext(ctx, &current,
				`SELECT saga_state FROM transactions WHERE id = $1`, req.TransactionID)
			if errors.Is(errSel, sql.ErrNoRows) {
				return apperr.ErrTransactionNotFound
			}
			if errSel != nil {
				return fmt.Errorf("read current state: %w", errSel)
			}
			if current == req.NewState {
				return apperr.ErrSagaAlreadyAdvanced
			}
			return &apperr.WrappedError{
				Op: op,
				Err: fmt.Errorf("%w: expected %q got %q",
					apperr.ErrInvalidSagaState, req.ExpectedFrom, current),
			}
		}

		if req.Step != nil {
			const stepQuery = `
				INSERT INTO saga_steps (id, transaction_id, event_id, step_name, step_kind, status)
				VALUES (:id, :transaction_id, :event_id, :step_name, :step_kind, :status)
				ON CONFLICT (transaction_id, step_name, step_kind) DO NOTHING`
			if _, err = tx.NamedExecContext(ctx, stepQuery, req.Step); err != nil {
				return fmt.Errorf("create saga step: %w", err)
			}
		}

		events := []*models.CreateEventRequest{}
		if req.Event != nil {
			events = append(events, req.Event)
		}
		events = append(events, req.ExtraEvents...)

		for _, ev := range events {
			if ev == nil {
				continue
			}
			const eventQuery = `
				INSERT INTO events (id, transaction_id, partition_key, type, step_name, status, source, trace_id, span_id, payload)
				VALUES (:id, :transaction_id, :partition_key, :type, :step_name, :status, :source, :trace_id, :span_id, :payload)
				ON CONFLICT (transaction_id, type, step_name) DO NOTHING`
			if _, err = tx.NamedExecContext(ctx, eventQuery, ev); err != nil {
				return fmt.Errorf("create event: %w", err)
			}
		}

		return nil
	})

	if errors.Is(err, apperr.ErrSagaAlreadyAdvanced) {
		return nil
	}
	if err != nil {
		return &apperr.WrappedError{Op: op, Err: err}
	}
	return nil
}
