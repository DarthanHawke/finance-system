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

	"account-service/internal/lib/errors/apperr"
	"account-service/internal/models"
	"account-service/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestAccountRepository содержит все зависимости для тестов
type TestAccountRepository struct {
	Repo      *repository.AccountRepository
	DB        *repository.Database
	MockDB    sqlmock.Sqlmock
	MockCache *MockRedisCache
}

// new создает TestAccountRepository
func new(t *testing.T) *TestAccountRepository {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	database := &repository.Database{DB: sqlxDB}

	mockCache := &MockRedisCache{}
	logger := zap.NewNop()

	repo := repository.NewAccountRepository(database, mockCache, logger)

	t.Cleanup(func() {
		db.Close()
	})

	return &TestAccountRepository{
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

func TestAccountRepository_CreateAccount(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		userID := generateTestUUID(t)
		now := time.Now()

		r.MockDB.ExpectExec("INSERT INTO accounts").
			WithArgs("RUB80311173817", "Casino dep account", userID, "RUB", now, now).
			WillReturnResult(sqlmock.NewResult(1, 1))

		r.MockCache.DeleteFunc = func(ctx context.Context, key string) error {
			assert.Equal(t, fmt.Sprintf("user_accounts:%s", userID), key)
			return nil
		}

		err := r.Repo.CreateAccount(context.Background(), models.CreateAccountRequest{
			Code:     "RUB80311173817",
			Name:     "Casino dep account",
			UserID:   userID,
			Currency: "RUB",
		})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		r := new(t)
		userID := generateTestUUID(t)
		now := time.Now()

		r.MockDB.ExpectExec("INSERT INTO accounts").
			WithArgs("RUB80311173817", "Casino dep account", userID, "USD", now, now).
			WillReturnError(errors.New("database error"))

		err := r.Repo.CreateAccount(context.Background(), models.CreateAccountRequest{
			Code:     "RUB80311173817",
			Name:     "Casino dep account",
			UserID:   userID,
			Currency: "USD",
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestAccountRepository_GetAccount(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)

		now := time.Now()
		userID := generateTestUUID(t)
		rows := sqlmock.NewRows([]string{
			"code", "name", "user_id", "currency",
			"balance", "blocked_funds", "status", "created_at", "updated_at",
		}).AddRow(
			"USD0101010101", "Credit account", userID, "USD",
			1000.42, 0, "active", now, now,
		)

		r.MockDB.ExpectQuery(`SELECT code, name, user_id, currency, 
					balance, blocked_funds, status, created_at, updated_at
         FROM accounts
         WHERE code = \$1`).
			WithArgs("USD0101010101").
			WillReturnRows(rows)

		result, err := r.Repo.GetAccount(context.Background(), models.GetAccountRequest{
			Code: "USD0101010101",
		})

		assert.NoError(t, err)
		assert.NotNil(t, result.Account)
		assert.Equal(t, "USD0101010101", result.Code)
		assert.Equal(t, "Credit account", result.Name)
		assert.Equal(t, 1000.42, result.Balance)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})

	t.Run("account not found", func(t *testing.T) {
		r := new(t)

		r.MockDB.ExpectQuery(`SELECT code, name, user_id, currency, 
					 balance, blocked_funds, status, created_at, updated_at
         FROM accounts
         WHERE code = \$1`).
			WithArgs("USD0101010101").
			WillReturnError(sql.ErrNoRows)

		_, err := r.Repo.GetAccount(context.Background(), models.GetAccountRequest{
			Code: "USD0101010101",
		})

		assert.Error(t, err)
		assert.Equal(t, apperr.ErrAccountNotFound, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestAccountRepository_GetAccounts(t *testing.T) {
	t.Run("success from cache", func(t *testing.T) {
		r := new(t)
		userID := generateTestUUID(t)

		expectedAccounts := models.GetAccountsResponse{
			Accounts: []models.Account{
				{
					Code:         "RUB12345678",
					Name:         "Account 1",
					UserID:       userID,
					Currency:     "RUB",
					Balance:      2356,
					BlockedFunds: 0,
					Status:       "active",
					CreatedAt:    time.Date(2025, 9, 3, 12, 5, 0, 0, time.UTC),
					UpdatedAt:    time.Date(2025, 9, 3, 12, 7, 0, 0, time.UTC),
				},
				{
					Code:         "USD12345678",
					Name:         "Account 2",
					UserID:       userID,
					Currency:     "USD",
					Balance:      765,
					BlockedFunds: 0,
					Status:       "blocked",
					CreatedAt:    time.Date(2025, 9, 3, 12, 8, 0, 0, time.UTC),
					UpdatedAt:    time.Date(2025, 9, 3, 12, 9, 0, 0, time.UTC),
				},
			},
		}

		expectedData, err := json.Marshal(expectedAccounts)
		require.NoError(t, err)

		r.MockCache.GetFunc = func(ctx context.Context, key string) (string, error) {
			assert.Equal(t, fmt.Sprintf("user_accounts:%s:limit:%d:offset:%d", userID, 10, 0), key)
			return string(expectedData), nil
		}

		result, err := r.Repo.GetAccounts(context.Background(), models.GetAccountsRequest{
			UserID: userID,
			Limit:  10,
			Offset: 0,
		})

		assert.NoError(t, err)
		assert.Len(t, result.Accounts, 2)
		assert.Equal(t, "RUB12345678", result.Accounts[0].Code)
		assert.Equal(t, "USD12345678", result.Accounts[1].Code)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})

	t.Run("success from database", func(t *testing.T) {
		r := new(t)
		userID := generateTestUUID(t)

		r.MockCache.GetFunc = func(ctx context.Context, key string) (string, error) {
			return "", errors.New("cache miss")
		}

		r.MockCache.SetWithTTLFunc = func(ctx context.Context, key string, value any) error {
			assert.Equal(t, fmt.Sprintf("user_accounts:%s:limit:%d:offset:%d", userID, 10, 0), key)

			// Проверяем, что в кэш сохраняются правильные данные
			dataStr, ok := value.(string)
			assert.True(t, ok)

			var cachedResponse models.GetAccountsResponse
			err := json.Unmarshal([]byte(dataStr), &cachedResponse)
			assert.NoError(t, err)
			assert.Len(t, cachedResponse.Accounts, 2)

			return nil
		}

		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"code", "name", "user_id", "currency", "balance",
			"blocked_funds", "status", "created_at", "updated_at",
		}).
			AddRow("RUB12345678", "Account 1", userID, "RUB", 4125.0, 0, "active", now, now).
			AddRow("USD12345678", "Account 2", userID, "USD", 123.0, 0, "blocked", now, now)

		r.MockDB.ExpectQuery(`SELECT code, name, user_id, currency, balance,
		       blocked_funds, status, created_at, updated_at
		FROM accounts 
		WHERE user_id = \$1
		LIMIT \$2 OFFSET \$3`).
			WithArgs(userID, 10, 0).
			WillReturnRows(rows)

		result, err := r.Repo.GetAccounts(context.Background(), models.GetAccountsRequest{
			UserID: userID,
			Limit:  10,
			Offset: 0,
		})

		assert.NoError(t, err)
		assert.Len(t, result.Accounts, 2)
		assert.Equal(t, "RUB12345678", result.Accounts[0].Code)
		assert.Equal(t, "USD12345678", result.Accounts[1].Code)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})

	t.Run("no accounts found", func(t *testing.T) {
		r := new(t)
		userID := generateTestUUID(t)

		r.MockCache.GetFunc = func(ctx context.Context, key string) (string, error) {
			return "", errors.New("cache miss")
		}

		r.MockDB.ExpectQuery(`SELECT code, name, user_id, currency, balance, 
		       blocked_funds, status, created_at, updated_at
		FROM accounts 
		WHERE user_id = \$1
		LIMIT \$2 OFFSET \$3`).
			WithArgs(userID, 10, 0).
			WillReturnError(sql.ErrNoRows)

		_, err := r.Repo.GetAccounts(context.Background(), models.GetAccountsRequest{
			UserID: userID,
			Limit:  10,
			Offset: 0,
		})

		assert.Error(t, err)
		assert.Equal(t, apperr.ErrAccountNotFound, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestAccountRepository_BlockFunds(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		userID := generateTestUUID(t)

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE accounts 
        SET blocked_funds = blocked_funds \+ \$1, 
            balance = balance - \$1,
            updated_at = \$2
        WHERE code = \$3 AND user_id = \$4 AND balance >= \$1 AND status = 'active'`).
			WithArgs(6.6, sqlmock.AnyArg(), "RUB12345678", userID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		r.MockDB.ExpectCommit()

		err := r.Repo.BlockFunds(context.Background(), models.FundsRequest{
			UserID: userID,
			Code:   "RUB12345678",
			Amount: 6.6,
		})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})

	t.Run("insufficient funds", func(t *testing.T) {
		r := new(t)
		userID := generateTestUUID(t)

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE accounts 
        SET blocked_funds = blocked_funds \+ \$1, 
            balance = balance - \$1,
            updated_at = \$2
        WHERE code = \$3 AND user_id = \$4 AND balance >= \$1 AND status = 'active'`).
			WithArgs(6.6, sqlmock.AnyArg(), "RUB12345678", userID).
			WillReturnResult(sqlmock.NewResult(0, 0))
		r.MockDB.ExpectRollback()

		err := r.Repo.BlockFunds(context.Background(), models.FundsRequest{
			UserID: userID,
			Code:   "RUB12345678",
			Amount: 6.6,
		})

		assert.Error(t, err)
		assert.True(t, errors.Is(err, apperr.ErrInsufficientFunds), "Expected ErrInsufficientFunds, got: %v", err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestAccountRepository_UnblockFunds(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		userID := generateTestUUID(t)

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE accounts 
		SET blocked_funds = blocked_funds \- \$1, 
		    balance = balance \+ \$1,
		    updated_at = \$2
		WHERE code = \$3 AND user_id = \$4 AND blocked_funds >= \$1 AND status IN \('active', 'blocked'\)`).
			WithArgs(6.6, sqlmock.AnyArg(), "RUB12345678", userID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		r.MockDB.ExpectCommit()

		err := r.Repo.UnblockFunds(context.Background(), models.FundsRequest{
			UserID: userID,
			Code:   "RUB12345678",
			Amount: 6.6,
		})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestAccountRepository_DepositFunds(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		userID := generateTestUUID(t)

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE accounts 
		SET balance = balance \+ \$1,
		    updated_at = \$2
		WHERE code = \$3 AND user_id = \$4 AND status = 'active'`).
			WithArgs(6.6, sqlmock.AnyArg(), "RUB12345678", userID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		r.MockDB.ExpectCommit()

		err := r.Repo.DepositFunds(context.Background(), models.FundsRequest{
			UserID: userID,
			Code:   "RUB12345678",
			Amount: 6.6,
		})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestAccountRepository_WithdrawFunds(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		userID := generateTestUUID(t)

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE accounts 
        SET blocked_funds = blocked_funds \- \$1,
            updated_at = \$2
        WHERE code = \$3 AND user_id = \$4 AND blocked_funds >= \$1 AND status = 'active'`).
			WithArgs(6.6, sqlmock.AnyArg(), "RUB12345678", userID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		r.MockDB.ExpectCommit()

		err := r.Repo.WithdrawFunds(context.Background(), models.FundsRequest{
			UserID: userID,
			Code:   "RUB12345678",
			Amount: 6.6,
		})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestAccountRepository_UpdateStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		userID := generateTestUUID(t)

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE accounts 
        SET status = \$1, 
            updated_at = \$2
        WHERE code = \$3 AND user_id = \$4`).
			WithArgs("blocked", sqlmock.AnyArg(), "RUB12345678", userID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		r.MockDB.ExpectCommit()

		r.MockCache.DeleteFunc = func(ctx context.Context, key string) error {
			assert.Equal(t, fmt.Sprintf("user_accounts:%s", userID), key)
			return nil
		}

		err := r.Repo.UpdateStatus(context.Background(), models.UpdateStatusRequest{
			UserID: userID,
			Code:   "RUB12345678",
			Status: "blocked",
		})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}
