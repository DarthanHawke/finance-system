// Пакет repository реализовывает работу с базами данных
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"payment-service/internal/lib/errors/apperr"
	"payment-service/internal/models"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// PaymentRepository структура релизующая методы для работы со платежами
type PaymentRepository struct {
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

func NewPaymentRepository(db *Database, cache RedisCacheManager, logger *zap.Logger) *PaymentRepository {
	return &PaymentRepository{
		db:     db,
		cache:  cache,
		logger: logger.With(zap.String("component", "payment_service")),
	}
}

// CreatePayment создаёт новый платёж
func (r *PaymentRepository) CreatePayment(
	ctx context.Context,
	req *models.CreatePaymentRequest,
	event *models.CreateEventRequest,
) error {
	const op = "repository.payment.CreatePayment"

	const queryPayments = `
		INSERT INTO payments (id, payment_type, direction, amount, currency, 
				sender_type, sender_account_code, sender_phone, sender_card_number,
				recipient_type, recipient_account_code, recipient_phone, recipient_card_number,
				processing_code, stan, authorization_code, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`

	const queryEvents = `
		INSERT INTO events (id, payment_id, type, status, source, created_at, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, queryPayments,
			req.ID,
			req.PaymentType,
			req.Direction,
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
				return apperr.ErrPaymentIDNotUnique
			}
			return fmt.Errorf("%s: %w", op, err)
		}

		_, err = tx.ExecContext(ctx, queryEvents,
			event.ID,
			event.PaymentID,
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

	go r.tryInvalidateAccountPaymentsCache(ctx, op, req.SenderAccountCode)
	go r.tryInvalidateAccountPaymentsCache(ctx, op, req.RecipientAccountCode)

	return nil
}

// GetPayment возвращает полную информацию о платеже
func (r *PaymentRepository) GetPayment(
	ctx context.Context,
	req *models.GetPaymentRequest,
) (models.GetPaymentResponse, error) {
	const op = "repository.payment.GetPayment"

	query := `SELECT id, payment_type, direction, amount, currency, 
				sender_type, sender_account_code, sender_phone, sender_card_number,
				recipient_type, recipient_account_code, recipient_phone, recipient_card_number,
				description, status, created_at, updated_at
		FROM payments 
		WHERE id = $1`

	var resp models.GetPaymentResponse

	err := r.db.GetContext(ctx, &resp, query, req.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.GetPaymentResponse{}, apperr.ErrPaymentNotFound
		}
		return models.GetPaymentResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	return resp, nil
}

// GetPayments возвращает все платежи связанные со счетом
func (r *PaymentRepository) GetPayments(
	ctx context.Context,
	req *models.GetPaymentsRequest,
) (models.GetPaymentsResponse, error) {
	const op = "repository.payment.GetPayments"

	cacheKey := fmt.Sprintf("account_payments:%s:limit:%d:offset:%d",
		req.AccountCode, req.Limit, req.Offset)
	var resp models.GetPaymentsResponse
	var payments []models.Payment

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

	query := `SELECT id, payment_type, direction, amount, currency, 
				sender_type, sender_account_code, sender_phone, sender_card_number,
				recipient_type, recipient_account_code, recipient_phone, recipient_card_number,
				description, status, created_at, updated_at
		FROM payments 
		WHERE sender_account_code = $1 or recipient_account_code = $1
		LIMIT $2 OFFSET $3`

	err = r.db.SelectContext(ctx, &payments, query, req.AccountCode, req.Limit, req.Offset)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.GetPaymentsResponse{}, apperr.ErrPaymentNotFound
		}
		return models.GetPaymentsResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	resp = models.GetPaymentsResponse{Payments: payments}

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

// UpdatePaymentStatus обновляет статус платежа
func (r *PaymentRepository) UpdatePaymentStatus(ctx context.Context, req *models.UpdatePaymentStatusRequest) error {
	const op = "repository.payment.UpdatePaymentStatus"

	const query = `UPDATE payments 
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

	go r.tryInvalidateAccountPaymentsCache(ctx, op, req.SenderAccountCode)
	go r.tryInvalidateAccountPaymentsCache(ctx, op, req.RecipientAccountCode)

	return nil
}

const cacheInvalidationTimeout = 5 * time.Second

// invalidateAccountPaymentsCache инвалидирует кэш по номеру счёта, и логирует ошибку в случае неудачи
func (r *PaymentRepository) tryInvalidateAccountPaymentsCache(ctx context.Context, op string, account string) {
	ctx, cancel := context.WithTimeout(ctx, cacheInvalidationTimeout)
	defer cancel()

	cacheKey := fmt.Sprintf("account_payments:%s", account)
	if err := r.cache.DeleteByPrefix(ctx, cacheKey); err != nil {
		r.logger.Warn("failed to delete cache key",
			zap.String("op", op),
			zap.String("cachekey", cacheKey),
			zap.Error(err),
		)
	}
}
