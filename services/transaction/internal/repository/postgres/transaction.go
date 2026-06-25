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

// InsertTransaction создаёт новый платёж
func (r *TransactionRepository) InsertTransaction(
	ctx context.Context,
	tx *sqlx.Tx,
	req *models.InsertTransactionRequest,
) error {
	const op = "transaction.InsertTransaction"

	const queryTransaction = `
		INSERT INTO transactions (id, idempotency_key, parent_transaction_id, type, status, amount, currency, description)
		VALUES (:id, :idempotency_key, :parent_transaction_id, :type, :status, :amount, :currency, :description)
	`

	_, err := tx.NamedExecContext(ctx, queryTransaction, req)

	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return apperr.ErrTransactionIDNotUnique
		}
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return nil
}

// InsertParties вставляет Parties в транзакцию
func (r *TransactionRepository) InsertParties(
	ctx context.Context,
	tx *sqlx.Tx,
	req *models.InsertPartiesRequest,
) error {
	const op = "transaction.InsertParties"

	const queryParties = `
		INSERT INTO transaction_parties (transaction_id, party_role, party_type, identifiers)
		VALUES (:transaction_id, :party_role, :party_type, :identifiers)
	`
	var err error
	for _, party := range req.Parties {
		_, err = tx.NamedExecContext(ctx, queryParties, party)
		if err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return apperr.ErrTransactionPartiesAlreadyExist
			}
			return &apperr.WrappedError{
				Op:  op,
				Err: err,
			}
		}
	}

	return nil
}

// UpdateSagaState обновляет состояние саги и статус транзакции
func (r *TransactionRepository) UpdateSagaState(
	ctx context.Context,
	tx *sqlx.Tx,
	req *models.UpdateSagaStateRequest,
) error {
	const op = "saga.UpdateSagaState"

	querySagaState := `
		UPDATE transactions
		SET saga_state = $1,
			status     = COALESCE($2, status),
			updated_at = $3
		WHERE id = $4
		AND saga_state = $5
	`

	result, err := tx.ExecContext(ctx, querySagaState, req.SagaState, req.Status, time.Now(), req.TransactionID, req.ExpectedFrom)
	if err != nil {
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
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
		if current == req.SagaState {
			return apperr.ErrSagaAlreadyAdvanced
		}
		return &apperr.WrappedError{
			Op: op,
			Err: fmt.Errorf("%w: expected %q got %q",
				apperr.ErrInvalidSagaState, req.ExpectedFrom, current),
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
            t.description, t.created_at, t.updated_at,
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
