// Пакет repository реализовывает работу с базами данных
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// TransactionRepository структура релизующая методы для работы со платежами
type TransactionRepository struct {
	db     *Database
	cache  RedisCacheManager
	logger *zap.Logger
}

// RedisCacheManager интерфейс с методами для кэша
type RedisCacheManager interface {
	Get(ctx context.Context, key string) (string, error)
	SetWithTTL(ctx context.Context, key string, value any) error
	Delete(ctx context.Context, key string) error
	DeleteByPrefix(ctx context.Context, prefix string) error
}

func NewTransactionRepository(db *Database, cache RedisCacheManager, logger *zap.Logger) *TransactionRepository {
	return &TransactionRepository{
		db:     db,
		cache:  cache,
		logger: logger.With(zap.String("component", "transaction_service")),
	}
}

// CreateTransaction создаёт новый платёж
func (r *TransactionRepository) CreateTransaction(
	ctx context.Context,
	req *models.CreateTransactionRequest,
	event *models.CreateEventRequest,
) error {
	const op = "repository.transaction.CreateTransaction"

	const queryTransactions = `
		INSERT INTO transactions (id, type, amount, currency, 
				sender_type, sender_account_code, sender_phone, sender_card_number,
				recipient_type, recipient_account_code, recipient_phone, recipient_card_number,
				processing_code, stan, authorization_code, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`

	const queryEvents = `
		INSERT INTO events (id, transaction_id, type, status, source, created_at, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, queryTransactions,
			req.ID,
			req.Type,
			req.Amount,
			req.Currency,
			req.SenderType,
			req.SenderAccountCode,
			req.SenderPhone,
			req.SenderCardNumber,
			req.RecipientType,
			req.RecipientAccountCode,
			req.RecipientPhone,
			req.RecipientAccountCode,
			req.ProcessingCode,
			req.Stan,
			req.AuthorizationCode,
			req.Description,
			req.Status,
			time.Now(),
			time.Now(),
		)

		if err != nil {
			if strings.Contains(err.Error(), "23505") {
				return apperr.ErrTransactionIDNotUnique
			}
			return fmt.Errorf("%s: %w", op, err)
		}

		_, err = tx.ExecContext(ctx, queryEvents,
			event.ID,
			event.TransactionID,
			event.Type,
			event.Status,
			event.Source,
			event.CreatedAt,
			event.Payload,
		)

		if err != nil {
			if strings.Contains(err.Error(), "23505") {
				return apperr.ErrEventIDNotUnique
			}
			return fmt.Errorf("%s: %w", op, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
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
	const op = "repository.transaction.GetTransaction"

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
		return models.GetTransactionResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	return resp, nil
}

// GetTransactions возвращает все платежи связанные со счетом
func (r *TransactionRepository) GetTransactions(
	ctx context.Context,
	req *models.GetTransactionsRequest,
) (models.GetTransactionsResponse, error) {
	const op = "repository.transaction.GetTransactions"

	cacheKey := fmt.Sprintf("account_transactions:%s:limit:%d:offset:%d",
		req.AccountCode, req.Limit, req.Offset)
	var resp models.GetTransactionsResponse
	var transactions []models.Transaction

	cachedData, err := r.cache.Get(ctx, cacheKey)
	if err == nil && cachedData != "" {
		if err := json.Unmarshal([]byte(cachedData), &resp); err == nil {
			r.logger.Debug("cache hit")
			return resp, nil
		}
		r.logger.Warn("failed to unmarshal cached data",
			zap.String("op", op),
			zap.Error(err),
		)
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
		if errors.Is(err, sql.ErrNoRows) {
			return models.GetTransactionsResponse{}, apperr.ErrTransactionNotFound
		}
		return models.GetTransactionsResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	resp = models.GetTransactionsResponse{Transactions: transactions}

	data, err := json.Marshal(resp)
	if err != nil {
		r.logger.Warn("failed to matshal json data",
			zap.String("op", op),
			zap.Error(err),
		)
		return resp, nil
	}
	if err := r.cache.SetWithTTL(ctx, cacheKey, string(data)); err != nil {
		r.logger.Warn("failed to set cache key with data",
			zap.String("op", op),
			zap.String("cachekey", cacheKey),
			zap.Error(err),
		)
	}

	return resp, nil
}

// UpdateTransactionStatus обновляет статус платежа
func (r *TransactionRepository) UpdateTransactionStatus(ctx context.Context, req *models.UpdateTransactionStatusRequest) error {
	const op = "repository.transaction.UpdateTransactionStatus"

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
		if rowsAffected != 1 {
			return fmt.Errorf("%s: expected 1 row affected, got %d", op, rowsAffected)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	go r.tryInvalidateAccountTransactionsCache(ctx, op, req.SenderAccountCode)
	go r.tryInvalidateAccountTransactionsCache(ctx, op, req.RecipientAccountCode)

	return nil
}

const cacheInvalidationTimeout = 5 * time.Second

// invalidateAccountTransactionsCache инвалидирует кэш по номеру счёта, и логирует ошибку в случае неудачи
func (r *TransactionRepository) tryInvalidateAccountTransactionsCache(ctx context.Context, op string, account string) {
	ctx, cancel := context.WithTimeout(ctx, cacheInvalidationTimeout)
	defer cancel()

	cacheKey := fmt.Sprintf("account_transactions:%s", account)
	if err := r.cache.DeleteByPrefix(ctx, cacheKey); err != nil {
		r.logger.Warn("failed to delete cache key",
			zap.String("op", op),
			zap.String("cachekey", cacheKey),
			zap.Error(err),
		)
	}
}
