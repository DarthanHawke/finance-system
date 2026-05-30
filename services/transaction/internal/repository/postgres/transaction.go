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
	db    *Database
	cache RedisCacheManager
}

// RedisCacheManager интерфейс с методами для кэша
type RedisCacheManager interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any) error
	Delete(ctx context.Context, key string) error
	DeleteByPrefix(ctx context.Context, prefix string) error
}

func NewTransactionRepository(db *Database, cache RedisCacheManager) *TransactionRepository {
	return &TransactionRepository{
		db:    db,
		cache: cache,
	}
}

// CreateTransaction создаёт новый платёж
func (r *TransactionRepository) CreateTransaction(
	ctx context.Context,
	req *models.CreateTransactionRequest,
	event *models.CreateEventRequest,
) error {
	const op = "transaction.CreateTransaction"

	const queryTransactions = `
		INSERT INTO transactions (id, type, amount, currency, 
				sender_type, sender_account_code, sender_phone, sender_card_number,
				recipient_type, recipient_account_code, recipient_phone, recipient_card_number,
				processing_code, stan, authorization_code, description, status)
		VALUES (:id, :type, :amount, :currency, 
				:sender_type, :sender_account_code, :sender_phone, :sender_card_number,
				:recipient_type, :recipient_account_code, :recipient_phone, :recipient_card_number,
				:processing_code, :stan, :authorization_code, :description, :status)
	`

	const queryEvents = `
		INSERT INTO events (id, transaction_id, partition_key, type, status, source, payload)
		VALUES (:id, :transaction_id, :partition_key, :type, :status, :source, :payload)
	`

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.NamedExecContext(ctx, queryTransactions, req)

		if err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return apperr.ErrTransactionIDNotUnique
			}
			return err
		}

		_, err = tx.NamedExecContext(ctx, queryEvents, event)

		if err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return apperr.ErrEventIDNotUnique
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

	go r.tryInvalidateAccountTransactionsCache(ctx, op, req.SenderAccountCode)
	go r.tryInvalidateAccountTransactionsCache(ctx, op, req.RecipientAccountCode)

	return nil
}

// GetTransaction возвращает полную информацию о платеже
func (r *TransactionRepository) GetTransaction(
	ctx context.Context,
	req *models.GetTransactionRequest,
) (models.GetTransactionResponse, error) {
	const op = "transaction.GetTransaction"

	query := `SELECT id, type, amount, currency, 
				sender_type, sender_account_code, sender_phone, sender_card_number,
				recipient_type, recipient_account_code, recipient_phone, recipient_card_number,
				description, status, created_at, updated_at
		FROM transactions 
		WHERE id = $1`

	var resp models.GetTransactionResponse

	err := r.db.GetContext(ctx, &resp, query, req.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.GetTransactionResponse{}, apperr.ErrTransactionNotFound
		}
		return models.GetTransactionResponse{}, &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	return resp, nil
}

// GetTransactions возвращает все платежи связанные со счетом
func (r *TransactionRepository) GetTransactions(
	ctx context.Context,
	req *models.GetTransactionsRequest,
) (models.GetTransactionsResponse, error) {
	const op = "transaction.GetTransactions"

	cacheKey := fmt.Sprintf("account_transactions:%s:limit:%d:offset:%d",
		req.AccountCode, req.Limit, req.Offset)
	var resp models.GetTransactionsResponse
	var transactions []models.Transaction

	cachedData, err := r.cache.Get(ctx, cacheKey)
	if err == nil && cachedData != "" {
		if err := json.Unmarshal([]byte(cachedData), &resp); err == nil {
			return resp, nil
		}
	}

	query := `SELECT id, type, amount, currency, 
				sender_type, sender_account_code, sender_phone, sender_card_number,
				recipient_type, recipient_account_code, recipient_phone, recipient_card_number,
				description, status, created_at, updated_at
		FROM transactions 
		WHERE sender_account_code = $1 or recipient_account_code = $1
		LIMIT $2 OFFSET $3`

	err = r.db.SelectContext(ctx, &transactions, query, req.AccountCode, req.Limit, req.Offset)
	if err != nil {
		return models.GetTransactionsResponse{}, &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	if len(transactions) == 0 {
		return models.GetTransactionsResponse{}, apperr.ErrTransactionNotFound
	}

	resp = models.GetTransactionsResponse{Transactions: transactions}

	data, err := json.Marshal(resp)
	if err != nil {
		return models.GetTransactionsResponse{}, &apperr.WrappedError{
			Op:  op,
			Err: fmt.Errorf("marshal response: %w", err),
		}
	}
	if err := r.cache.Set(ctx, cacheKey, string(data)); err != nil {
	}

	return resp, nil
}

// UpdateTransactionStatus обновляет статус платежа
func (r *TransactionRepository) UpdateTransactionStatus(ctx context.Context, req *models.UpdateTransactionStatusRequest) error {
	const op = "transaction.UpdateTransactionStatus"

	const query = `UPDATE transactions 
					SET status = $1, updated_at = $2
					WHERE id = $3
	`

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, query, req.Status, time.Now(), req.ID)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return apperr.ErrTransactionNotFound
		}

		return nil
	})
	if err != nil {
		return &apperr.WrappedError{
			Op:  op,
			Err: err,
		}
	}

	go r.tryInvalidateAccountTransactionsCache(ctx, op, req.SenderAccountCode)
	go r.tryInvalidateAccountTransactionsCache(ctx, op, req.RecipientAccountCode)

	return nil
}

const cacheInvalidationTimeout = 5 * time.Second

// invalidateAccountTransactionsCache инвалидирует кэш по номеру счёта
func (r *TransactionRepository) tryInvalidateAccountTransactionsCache(ctx context.Context, op string, account string) {
	defer func() {
		if p := recover(); p != nil {
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, cacheInvalidationTimeout)
	defer cancel()

	cacheKey := fmt.Sprintf("account_transactions:%s", account)
	if err := r.cache.DeleteByPrefix(ctx, cacheKey); err != nil {
	}
}
