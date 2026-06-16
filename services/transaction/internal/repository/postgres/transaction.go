// Пакет postgres реализовывает работу с базами данных
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

// TransactionRepository структура релизующая методы для работы со платежами
type TransactionRepository struct {
	db *Database
}

func NewTransactionRepository(db *Database) *TransactionRepository {
	return &TransactionRepository{
		db: db,
	}
}

// CreateTransaction создаёт новый платёж
func (r *TransactionRepository) CreateTransaction(
	ctx context.Context,
	req *models.CreateTransactionRequest,
	event *models.CreateEventRequest,
	step *models.CreateSagaStepRequest,
) error {
	const op = "transaction.CreateTransaction"

	const queryTransactions = `
		INSERT INTO transactions (id, idempotency_key, parent_transaction_id, type, status, amount, currency, description)
		VALUES (:id, :idempotency_key, :parent_transaction_id, :type, :status, :amount, :currency, :description)
	`

	const queryParties = `
		INSERT INTO transaction_parties (transaction_id, role, party_type, identifiers)
		VALUES (:transaction_id, :role, :party_type, :identifiers)
	`

	const queryEvents = `
		INSERT INTO events (id, transaction_id, partition_key, type, status, source, trace_id, span_id, payload)
		VALUES (:id, :transaction_id, :partition_key, :type, :status, :source, :trace_id, :span_id, :payload)
	`

	const querySagaStep = `
		INSERT INTO saga_steps (id, transaction_id, event_id, step_name, step_kind, status)
		VALUES (:id, :transaction_id, :event_id, :step_name, :step_kind, :status)
	`

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.NamedExecContext(ctx, queryTransactions, req)

		if err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return apperr.ErrTransactionIDNotUnique
			}
			return err
		}

		for _, party := range req.Parties {
			_, err = tx.NamedExecContext(ctx, queryParties, party)
			if err != nil {
				if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
					return apperr.ErrTransactionPartiesAlreadyExist
				}
				return err
			}
		}

		_, err = tx.NamedExecContext(ctx, queryEvents, event)
		if err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return apperr.ErrEventIDNotUnique
			}
			return err
		}

		_, err = tx.NamedExecContext(ctx, querySagaStep, step)
		if err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return apperr.ErrSagaStepAlreadyExists
			}
			return err
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

// GetTransaction возвращает полную информацию о платеже
func (r *TransactionRepository) GetTransaction(
	ctx context.Context,
	req *models.GetTransactionRequest,
) (models.GetTransactionResponse, error) {
	const op = "transaction.GetTransaction"

	query := `
        SELECT t.id, t.idempotency_key, t.parent_transaction_id, 
			t.type::text, t.status::text, t.amount::float8, t.currency,
            t.initiator::text, t.description, t.created_at, t.updated_at,
            COALESCE(
                jsonb_agg(
                    jsonb_build_object(
                        'role', tp.role::text,
                        'party_type', tp.party_type::text,
                        'identifiers', tp.identifiers
                    )
                ) FILTER (WHERE tp.transaction_id IS NOT NULL),
                '[]'::jsonb
            ) as parties
        FROM transactions t
        LEFT JOIN transaction_parties tp ON t.id = tp.transaction_id
        WHERE t.id = $1
        GROUP BY t.id
    `

	var row struct {
		models.Transaction
		PartiesJSON json.RawMessage `db:"parties"`
	}

	err := r.db.GetContext(ctx, &row, query, req.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.GetTransactionResponse{}, apperr.ErrTransactionNotFound
		}
		return models.GetTransactionResponse{}, &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	if err := json.Unmarshal(row.PartiesJSON, &row.Transaction.Parties); err != nil {
		return models.GetTransactionResponse{}, &apperr.WrappedError{
			Op:  op,
			Err: fmt.Errorf("unmarshal payload: %w", err),
		}
	}

	return models.GetTransactionResponse{
		Transaction: &row.Transaction,
	}, nil
}

// UpdateTransactionStatus обновляет статус платежа
func (r *TransactionRepository) UpdateTransactionStatus(ctx context.Context, req *models.UpdateTransactionStatusRequest) error {
	const op = "transaction.UpdateTransactionStatus"

	const query = `UPDATE transactions 
					SET status = $1, updated_at = $2
					WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, req.Status, time.Now(), req.ID)
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
