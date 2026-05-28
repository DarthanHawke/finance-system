// пакет repository_test - тесты для пакета repository
package repository_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"
	repository "transaction-service/internal/repository/postgres"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransactionRepository содержит все зависимости для тестов
type TestTransactionRepository struct {
	Repo      *repository.TransactionRepository
	DB        *repository.Database
	MockDB    sqlmock.Sqlmock
	MockCache *MockRedisCache
}

// new создает TestAccountRepository
func new(t *testing.T) *TestTransactionRepository {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	database := &repository.Database{DB: sqlxDB}

	mockCache := &MockRedisCache{}

	repo := repository.NewTransactionRepository(database, mockCache)

	t.Cleanup(func() {
		db.Close()
	})

	return &TestTransactionRepository{
		Repo:      repo,
		DB:        database,
		MockDB:    mock,
		MockCache: mockCache,
	}
}

// MockRedisCache реализует RedisCacheManager для тестов
type MockRedisCache struct {
	GetFunc            func(ctx context.Context, key string) (string, error)
	SetFunc            func(ctx context.Context, key string, value any) error
	DeleteFunc         func(ctx context.Context, key string) error
	DeleteByPrefixFunc func(ctx context.Context, prefix string) error
}

func (m *MockRedisCache) Get(ctx context.Context, key string) (string, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key)
	}
	return "", nil
}

func (m *MockRedisCache) Set(ctx context.Context, key string, value any) error {
	if m.SetFunc != nil {
		return m.SetFunc(ctx, key, value)
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

func TestTransactionRepository_CreateTransaction(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		transactionID := generateTestUUID(t)
		eventID := generateTestUUID(t)
		now := time.Now()
		expectedPayload := `{"amount":1500,"currency":"RUB","sender":"RUB80311173817","recipient":"RUB123456789"}`

		r.MockDB.ExpectBegin()

		r.MockDB.ExpectExec("INSERT INTO transactions").
			WithArgs(
				transactionID,
				"transfer",
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

		r.MockDB.ExpectExec("INSERT INTO events").
			WithArgs(
				eventID,
				transactionID,
				"key",
				"transaction_created",
				"pending",
				"transaction-service",
				now,
				[]byte(expectedPayload),
			).WillReturnResult(sqlmock.NewResult(1, 1))

		r.MockDB.ExpectCommit()

		r.MockCache.DeleteFunc = func(ctx context.Context, key string) error {
			assert.Equal(t, fmt.Sprintf("account_transactions:%s", "RUB80311173817"), key)
			assert.Equal(t, fmt.Sprintf("account_transactions:%s", "RUB123456789"), key)
			return nil
		}

		err := r.Repo.CreateTransaction(context.Background(),
			&models.CreateTransactionRequest{
				Transaction: &models.Transaction{
					ID:                   transactionID,
					Type:                 "transfer",
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
					CreatedAt:            now,
					UpdatedAt:            now,
				},
			},
			&models.CreateEventRequest{
				Event: &models.Event{
					ID:            eventID,
					TransactionID: transactionID,
					PartitionKey:  "key",
					Type:          "transaction_created",
					Status:        "pending",
					Source:        "transaction-service",
					CreatedAt:     now,
					Payload:       []byte(expectedPayload),
				},
			})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		r := new(t)
		transactionID := generateTestUUID(t)
		eventID := generateTestUUID(t)
		now := time.Now()
		expectedPayload := `{"amount":1500,"currency":"RUB","sender":"RUB80311173817","recipient":"RUB123456789"}`

		r.MockDB.ExpectBegin()

		r.MockDB.ExpectExec("INSERT INTO transactions").
			WithArgs(
				transactionID,
				"transfer",
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

		r.MockDB.ExpectRollback()

		err := r.Repo.CreateTransaction(context.Background(),
			&models.CreateTransactionRequest{
				Transaction: &models.Transaction{
					ID:                   transactionID,
					Type:                 "transfer",
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
					CreatedAt:            now,
					UpdatedAt:            now,
				},
			},
			&models.CreateEventRequest{
				Event: &models.Event{
					ID:            eventID,
					TransactionID: transactionID,
					PartitionKey:  "key",
					Type:          "transaction_created",
					Status:        "pending",
					Source:        "transaction-service",
					CreatedAt:     now,
					Payload:       []byte(expectedPayload),
				},
			})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestTransactionRepository_GetTransaction(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		now := time.Now()
		transactionID := generateTestUUID(t)

		rows := sqlmock.NewRows([]string{
			"id", "type", "amount", "currency",
			"sender_type", "sender_account_code", "sender_phone", "sender_card_number",
			"recipient_type", "recipient_account_code", "recipient_phone", "recipient_card_number",
			"description", "status", "created_at", "updated_at",
		}).AddRow(
			transactionID,
			"transfer",
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

		r.MockDB.ExpectQuery(`SELECT id, type, amount, currency, 
				sender_type, sender_account_code, sender_phone, sender_card_number,
				recipient_type, recipient_account_code, recipient_phone, recipient_card_number,
				description, status, created_at, updated_at
		FROM transactions 
		WHERE id = \$1`).
			WithArgs(transactionID).
			WillReturnRows(rows)

		result, err := r.Repo.GetTransaction(context.Background(), &models.GetTransactionRequest{
			ID: transactionID,
		})

		assert.NoError(t, err)
		assert.NotNil(t, result.Transaction)
		assert.Equal(t, transactionID, result.ID)
		assert.Equal(t, "RUB80311173817", result.SenderAccountCode)
		assert.Equal(t, "RUB123456789", result.RecipientAccountCode)
		assert.Equal(t, 1500.0, result.Amount)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})

	t.Run("account not found", func(t *testing.T) {
		r := new(t)
		transactionID := generateTestUUID(t)

		r.MockDB.ExpectQuery(`SELECT id, type, amount, currency, 
				sender_type, sender_account_code, sender_phone, sender_card_number,
				recipient_type, recipient_account_code, recipient_phone, recipient_card_number,
				description, status, created_at, updated_at
		FROM transactions 
		WHERE id = \$1`).
			WithArgs(transactionID).
			WillReturnError(sql.ErrNoRows)

		_, err := r.Repo.GetTransaction(context.Background(), &models.GetTransactionRequest{
			ID: transactionID,
		})

		assert.Error(t, err)
		assert.Equal(t, apperr.ErrTransactionNotFound, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestTransactionRepository_GetTransactions(t *testing.T) {
	t.Run("success from cache", func(t *testing.T) {
		r := new(t)
		transactionIDOne := generateTestUUID(t)
		transactionIDTwo := generateTestUUID(t)

		expectedTransactions := models.GetTransactionsResponse{
			Transactions: []models.Transaction{
				{
					ID:                   transactionIDOne,
					Type:                 "transfer",
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
					ID:                   transactionIDTwo,
					Type:                 "transfer",
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

		expectedData, err := json.Marshal(expectedTransactions)
		require.NoError(t, err)

		r.MockCache.GetFunc = func(ctx context.Context, key string) (string, error) {
			assert.Equal(t, fmt.Sprintf("account_transactions:%s:limit:%d:offset:%d", "RUB80311173817", 10, 0), key)
			return string(expectedData), nil
		}

		result, err := r.Repo.GetTransactions(context.Background(), &models.GetTransactionsRequest{
			AccountCode: "RUB80311173817",
			Limit:       10,
			Offset:      0,
		})

		assert.NoError(t, err)
		assert.Len(t, result.Transactions, 2)
		assert.Equal(t, transactionIDOne, result.Transactions[0].ID)
		assert.Equal(t, transactionIDTwo, result.Transactions[1].ID)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})

	t.Run("success from database", func(t *testing.T) {
		r := new(t)
		transactionIDOne := generateTestUUID(t)
		transactionIDTwo := generateTestUUID(t)

		r.MockCache.GetFunc = func(ctx context.Context, key string) (string, error) {
			return "", errors.New("cache miss")
		}

		r.MockCache.SetFunc = func(ctx context.Context, key string, value any) error {
			assert.Equal(t, fmt.Sprintf("account_transactions:%s:limit:%d:offset:%d", "RUB80311173817", 10, 0), key)

			// проверяем, что в кэш сохраняются правильные данные
			dataStr, ok := value.(string)
			assert.True(t, ok)

			var cachedResponse models.GetTransactionsResponse
			err := json.Unmarshal([]byte(dataStr), &cachedResponse)
			assert.NoError(t, err)
			assert.Len(t, cachedResponse.Transactions, 2)

			return nil
		}

		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "type", "amount", "currency",
			"sender_type", "sender_account_code", "sender_phone", "sender_card_number",
			"recipient_type", "recipient_account_code", "recipient_phone", "recipient_card_number",
			"description", "status", "created_at", "updated_at",
		}).AddRow(
			transactionIDOne,
			"transfer",
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
			transactionIDTwo,
			"transfer",
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

		r.MockDB.ExpectQuery(`SELECT id, type, amount, currency, 
				sender_type, sender_account_code, sender_phone, sender_card_number,
				recipient_type, recipient_account_code, recipient_phone, recipient_card_number,
				description, status, created_at, updated_at
		FROM transactions 
		WHERE sender_account_code = \$1 or recipient_account_code = \$1
		LIMIT \$2 OFFSET \$3`).
			WithArgs("RUB80311173817", 10, 0).
			WillReturnRows(rows)

		result, err := r.Repo.GetTransactions(context.Background(), &models.GetTransactionsRequest{
			AccountCode: "RUB80311173817",
			Limit:       10,
			Offset:      0,
		})

		assert.NoError(t, err)
		assert.Len(t, result.Transactions, 2)
		assert.Equal(t, transactionIDOne, result.Transactions[0].ID)
		assert.Equal(t, transactionIDTwo, result.Transactions[1].ID)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})

	t.Run("no accounts found", func(t *testing.T) {
		r := new(t)

		r.MockCache.GetFunc = func(ctx context.Context, key string) (string, error) {
			return "", errors.New("cache miss")
		}

		r.MockDB.ExpectQuery(`SELECT id, type, amount, currency, 
				sender_type, sender_account_code, sender_phone, sender_card_number,
				recipient_type, recipient_account_code, recipient_phone, recipient_card_number,
				description, status, created_at, updated_at
		FROM transactions 
		WHERE sender_account_code = \$1 or recipient_account_code = \$1
		LIMIT \$2 OFFSET \$3`).
			WithArgs("RUB80311173817", 10, 0).
			WillReturnError(sql.ErrNoRows)

		_, err := r.Repo.GetTransactions(context.Background(), &models.GetTransactionsRequest{
			AccountCode: "RUB80311173817",
			Limit:       10,
			Offset:      0,
		})

		assert.Error(t, err)
		assert.Equal(t, apperr.ErrTransactionNotFound, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestTransactionRepository_UpdateTransactionStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		transactionID := generateTestUUID(t)

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE transactions 
					SET status = \$1, updated_at = \$2 
					WHERE id = \$3`).
			WithArgs("complited", sqlmock.AnyArg(), transactionID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		r.MockDB.ExpectCommit()

		err := r.Repo.UpdateTransactionStatus(context.Background(), &models.UpdateTransactionStatusRequest{
			ID:                   transactionID,
			SenderAccountCode:    "RUB80311173817",
			RecipientAccountCode: "RUB12345678",
			Status:               "complited",
		})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}
