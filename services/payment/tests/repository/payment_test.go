// Пакет repository_test - тесты для пакета repository
package repository_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"payment-service/internal/lib/errors/apperr"
	"payment-service/internal/models"
	"payment-service/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestPaymentRepository содержит все зависимости для тестов
type TestPaymentRepository struct {
	Repo      *repository.PaymentRepository
	DB        *repository.Database
	MockDB    sqlmock.Sqlmock
	MockCache *MockRedisCache
}

// new создает TestAccountRepository
func new(t *testing.T) *TestPaymentRepository {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	database := &repository.Database{DB: sqlxDB}

	mockCache := &MockRedisCache{}
	logger := zap.NewNop()

	repo := repository.NewPaymentRepository(database, mockCache, logger)

	t.Cleanup(func() {
		db.Close()
	})

	return &TestPaymentRepository{
		Repo:      repo,
		DB:        database,
		MockDB:    mock,
		MockCache: mockCache,
	}
}

// MockRedisCache реализует RedisCacheManager для тестов
type MockRedisCache struct {
	GetFunc            func(ctx context.Context, key string) (string, error)
	SetWithTTLFunc     func(ctx context.Context, key string, value any) error
	DeleteFunc         func(ctx context.Context, key string) error
	DeleteByPrefixFunc func(ctx context.Context, prefix string) error
}

func (m *MockRedisCache) Get(ctx context.Context, key string) (string, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key)
	}
	return "", nil
}

func (m *MockRedisCache) SetWithTTL(ctx context.Context, key string, value any) error {
	if m.SetWithTTLFunc != nil {
		return m.SetWithTTLFunc(ctx, key, value)
	}
	return nil
}

func (m *MockRedisCache) Delete(ctx context.Context, key string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, key)
	}
	return nil
}

func (m *MockRedisCache) DeleteByPrefix(ctx context.Context, prefix string) error {
	if m.DeleteByPrefixFunc != nil {
		return m.DeleteByPrefixFunc(ctx, prefix)
	}
	return nil
}

// generateTestUUID - генерим uuid
func generateTestUUID(t *testing.T) uuid.UUID {
	id, err := uuid.NewUUID()
	require.NoError(t, err)
	return id
}

func TestPaymentRepository_CreatePayment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		paymentID := generateTestUUID(t)
		now := time.Now()

		r.MockDB.ExpectExec("INSERT INTO payments").
			WithArgs(
				paymentID,
				"transfer",
				"outgoing",
				1500.0,
				"RUB",
				"internal",
				"RUB80311173817",
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				"internal",
				"RUB123456789",
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				"123456",
				"789012",
				sqlmock.AnyArg(),
				"Dep to casino",
				"pending",
				now,
				now,
			).WillReturnResult(sqlmock.NewResult(1, 1))

		r.MockCache.DeleteFunc = func(ctx context.Context, key string) error {
			assert.Equal(t, fmt.Sprintf("account_payments:%s", "RUB80311173817"), key)
			assert.Equal(t, fmt.Sprintf("account_payments:%s", "RUB123456789"), key)
			return nil
		}

		err := r.Repo.CreatePayment(context.Background(), models.CreatePaymentRequest{
			ID:                   paymentID,
			PaymentType:          "transfer",
			Direction:            "outgoing",
			Amount:               1500.0,
			Currency:             "RUB",
			SenderType:           "internal",
			SenderAccountCode:    "RUB80311173817",
			SenderPhone:          "",
			SenderCardNumber:     "",
			RecipientType:        "internal",
			RecipientAccountCode: "RUB123456789",
			RecipientPhone:       "",
			RecipientCardNumber:  "",
			ProcessingCode:       "123456",
			Stan:                 "789012",
			AuthorizationCode:    "",
			Description:          "Dep to casino",
			Status:               "pending",
		})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		r := new(t)
		paymentID := generateTestUUID(t)
		now := time.Now()

		r.MockDB.ExpectExec("INSERT INTO payments").
			WithArgs(
				paymentID,
				"transfer",
				"outgoing",
				1500.0,
				"RUB",
				"internal",
				"RUB80311173817",
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				"internal",
				"RUB123456789",
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				"123456",
				"789012",
				sqlmock.AnyArg(),
				"Dep to casino",
				"pending",
				now,
				now,
			).WillReturnError(errors.New("database error"))

		err := r.Repo.CreatePayment(context.Background(), models.CreatePaymentRequest{
			ID:                   paymentID,
			PaymentType:          "transfer",
			Direction:            "outgoing",
			Amount:               1500.0,
			Currency:             "RUB",
			SenderType:           "internal",
			SenderAccountCode:    "RUB80311173817",
			SenderPhone:          "",
			SenderCardNumber:     "",
			RecipientType:        "internal",
			RecipientAccountCode: "RUB123456789",
			RecipientPhone:       "",
			RecipientCardNumber:  "",
			ProcessingCode:       "123456",
			Stan:                 "789012",
			AuthorizationCode:    "",
			Description:          "Dep to casino",
			Status:               "pending",
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestPaymentRepository_GetPayment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		now := time.Now()
		paymentID := generateTestUUID(t)

		rows := sqlmock.NewRows([]string{
			"id", "payment_type", "direction", "amount", "currency",
			"sender_type", "sender_account_code", "sender_phone", "sender_card_number",
			"recipient_type", "recipient_account_code", "recipient_phone", "recipient_card_number",
			"description", "status", "created_at", "updated_at",
		}).AddRow(
			paymentID,
			"transfer",
			"outgoing",
			1500.0,
			"RUB",
			"internal",
			"RUB80311173817",
			"",
			"",
			"internal",
			"RUB123456789",
			"",
			"",
			"Dep to casino",
			"pending",
			now,
			now,
		)

		r.MockDB.ExpectQuery(`SELECT id, payment_type, direction, amount, currency, 
				sender_type, sender_account_code, sender_phone, sender_card_number,
				recipient_type, recipient_account_code, recipient_phone, recipient_card_number,
				description, status, created_at, updated_at
		FROM payments 
		WHERE id = \$1`).
			WithArgs(paymentID).
			WillReturnRows(rows)

		result, err := r.Repo.GetPayment(context.Background(), models.GetPaymentRequest{
			ID: paymentID,
		})

		assert.NoError(t, err)
		assert.NotNil(t, result.Payment)
		assert.Equal(t, paymentID, result.ID)
		assert.Equal(t, "RUB80311173817", result.SenderAccountCode)
		assert.Equal(t, "RUB123456789", result.RecipientAccountCode)
		assert.Equal(t, 1500.0, result.Amount)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})

	t.Run("account not found", func(t *testing.T) {
		r := new(t)
		paymentID := generateTestUUID(t)

		r.MockDB.ExpectQuery(`SELECT id, payment_type, direction, amount, currency, 
				sender_type, sender_account_code, sender_phone, sender_card_number,
				recipient_type, recipient_account_code, recipient_phone, recipient_card_number,
				description, status, created_at, updated_at
		FROM payments 
		WHERE id = \$1`).
			WithArgs(paymentID).
			WillReturnError(sql.ErrNoRows)

		_, err := r.Repo.GetPayment(context.Background(), models.GetPaymentRequest{
			ID: paymentID,
		})

		assert.Error(t, err)
		assert.Equal(t, apperr.ErrPaymentNotFound, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestPaymentRepository_GetPayments(t *testing.T) {
	t.Run("success from cache", func(t *testing.T) {
		r := new(t)
		paymentIDOne := generateTestUUID(t)
		paymentIDTwo := generateTestUUID(t)

		expectedPayments := models.GetPaymentsResponse{
			Payments: []models.Payment{
				{
					ID:                   paymentIDOne,
					PaymentType:          "transfer",
					Direction:            "outgoing",
					Amount:               1500.0,
					Currency:             "RUB",
					SenderType:           "internal",
					SenderAccountCode:    "RUB80311173817",
					RecipientType:        "internal",
					RecipientAccountCode: "RUB123456789",
					Description:          "Dep to casino",
					Status:               "pending",
				},
				{
					ID:                   paymentIDTwo,
					PaymentType:          "transfer",
					Direction:            "incoming",
					Amount:               240.0,
					Currency:             "RUB",
					SenderType:           "internal",
					SenderAccountCode:    "RUB123456789",
					RecipientType:        "internal",
					RecipientAccountCode: "RUB80311173817",
					Description:          "Win",
					Status:               "completed",
				},
			},
		}

		expectedData, err := json.Marshal(expectedPayments)
		require.NoError(t, err)

		r.MockCache.GetFunc = func(ctx context.Context, key string) (string, error) {
			assert.Equal(t, fmt.Sprintf("account_payments:%s:limit:%d:offset:%d", "RUB80311173817", 10, 0), key)
			return string(expectedData), nil
		}

		result, err := r.Repo.GetPayments(context.Background(), models.GetPaymentsRequest{
			AccountCode: "RUB80311173817",
			Limit:       10,
			Offset:      0,
		})

		assert.NoError(t, err)
		assert.Len(t, result.Payments, 2)
		assert.Equal(t, paymentIDOne, result.Payments[0].ID)
		assert.Equal(t, paymentIDTwo, result.Payments[1].ID)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})

	t.Run("success from database", func(t *testing.T) {
		r := new(t)
		paymentIDOne := generateTestUUID(t)
		paymentIDTwo := generateTestUUID(t)

		r.MockCache.GetFunc = func(ctx context.Context, key string) (string, error) {
			return "", errors.New("cache miss")
		}

		r.MockCache.SetWithTTLFunc = func(ctx context.Context, key string, value any) error {
			assert.Equal(t, fmt.Sprintf("account_payments:%s:limit:%d:offset:%d", "RUB80311173817", 10, 0), key)

			// Проверяем, что в кэш сохраняются правильные данные
			dataStr, ok := value.(string)
			assert.True(t, ok)

			var cachedResponse models.GetPaymentsResponse
			err := json.Unmarshal([]byte(dataStr), &cachedResponse)
			assert.NoError(t, err)
			assert.Len(t, cachedResponse.Payments, 2)

			return nil
		}

		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "payment_type", "direction", "amount", "currency",
			"sender_type", "sender_account_code", "sender_phone", "sender_card_number",
			"recipient_type", "recipient_account_code", "recipient_phone", "recipient_card_number",
			"description", "status", "created_at", "updated_at",
		}).AddRow(
			paymentIDOne,
			"transfer",
			"outgoing",
			1500.0,
			"RUB",
			"internal",
			"RUB80311173817",
			"",
			"",
			"internal",
			"RUB123456789",
			"",
			"",
			"Dep to casino",
			"pending",
			now,
			now,
		).AddRow(
			paymentIDTwo,
			"transfer",
			"incoming",
			240.0,
			"RUB",
			"internal",
			"RUB123456789",
			"",
			"",
			"internal",
			"RUB80311173817",
			"",
			"",
			"Win",
			"completed",
			now,
			now,
		)

		r.MockDB.ExpectQuery(`SELECT id, payment_type, direction, amount, currency, 
				sender_type, sender_account_code, sender_phone, sender_card_number,
				recipient_type, recipient_account_code, recipient_phone, recipient_card_number,
				description, status, created_at, updated_at
		FROM payments 
		WHERE sender_account_code = \$1 or recipient_account_code = \$1
		LIMIT \$2 OFFSET \$3`).
			WithArgs("RUB80311173817", 10, 0).
			WillReturnRows(rows)

		result, err := r.Repo.GetPayments(context.Background(), models.GetPaymentsRequest{
			AccountCode: "RUB80311173817",
			Limit:       10,
			Offset:      0,
		})

		assert.NoError(t, err)
		assert.Len(t, result.Payments, 2)
		assert.Equal(t, paymentIDOne, result.Payments[0].ID)
		assert.Equal(t, paymentIDTwo, result.Payments[1].ID)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})

	t.Run("no accounts found", func(t *testing.T) {
		r := new(t)

		r.MockCache.GetFunc = func(ctx context.Context, key string) (string, error) {
			return "", errors.New("cache miss")
		}

		r.MockDB.ExpectQuery(`SELECT id, payment_type, direction, amount, currency, 
				sender_type, sender_account_code, sender_phone, sender_card_number,
				recipient_type, recipient_account_code, recipient_phone, recipient_card_number,
				description, status, created_at, updated_at
		FROM payments 
		WHERE sender_account_code = \$1 or recipient_account_code = \$1
		LIMIT \$2 OFFSET \$3`).
			WithArgs("RUB80311173817", 10, 0).
			WillReturnError(sql.ErrNoRows)

		_, err := r.Repo.GetPayments(context.Background(), models.GetPaymentsRequest{
			AccountCode: "RUB80311173817",
			Limit:       10,
			Offset:      0,
		})

		assert.Error(t, err)
		assert.Equal(t, apperr.ErrPaymentNotFound, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestPaymentRepository_UpdatePaymentStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		paymentID := generateTestUUID(t)

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE payments 
					SET status = \$1, updated_at = \$2 
					WHERE id = \$3`).
			WithArgs("complited", sqlmock.AnyArg(), paymentID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		r.MockDB.ExpectCommit()

		err := r.Repo.UpdatePaymentStatus(context.Background(), models.UpdatePaymentStatusRequest{
			ID:                   paymentID,
			SenderAccountCode:    "RUB80311173817",
			RecipientAccountCode: "RUB12345678",
			Status:               "complited",
		})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}
