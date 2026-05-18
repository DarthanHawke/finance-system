// Пакет repository_test - тесты для пакета repository
package repository_test

import (
	"context"
	"database/sql"
	"errors"
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
	Repo   *repository.AccountRepository
	DB     *repository.Database
	MockDB sqlmock.Sqlmock
}

// new создает TestAccountRepository
func new(t *testing.T) *TestAccountRepository {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	database := &repository.Database{DB: sqlxDB}

	logger := zap.NewNop()

	repo := repository.NewAccountRepository(database, logger)

	t.Cleanup(func() {
		db.Close()
	})

	return &TestAccountRepository{
		Repo:   repo,
		DB:     database,
		MockDB: mock,
	}
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

		err := r.Repo.CreateAccount(context.Background(), &models.CreateAccountRequest{
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

		err := r.Repo.CreateAccount(context.Background(), &models.CreateAccountRequest{
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
			"balance", "frozen_balance", "reserve_balance", "status", "created_at", "updated_at",
		}).AddRow(
			"USD0101010101", "Credit account", userID, "USD",
			1000.42, 0, 0, "active", now, now,
		)

		r.MockDB.ExpectQuery(`SELECT code, name, user_id, currency, 
					balance, frozen_balance, reserve_balance, status, created_at, updated_at
         FROM accounts
         WHERE code = \$1`).
			WithArgs("USD0101010101").
			WillReturnRows(rows)

		result, err := r.Repo.GetAccount(context.Background(), &models.GetAccountRequest{
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
					 balance, frozen_balance, reserve_balance, status, created_at, updated_at
         FROM accounts
         WHERE code = \$1`).
			WithArgs("USD0101010101").
			WillReturnError(sql.ErrNoRows)

		_, err := r.Repo.GetAccount(context.Background(), &models.GetAccountRequest{
			Code: "USD0101010101",
		})

		assert.Error(t, err)
		assert.Equal(t, apperr.ErrAccountNotFound, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestAccountRepository_GetAccounts(t *testing.T) {
	t.Run("success from database", func(t *testing.T) {
		r := new(t)
		userID := generateTestUUID(t)

		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"code", "name", "user_id", "currency", "balance",
			"frozen_balance", "reserve_balance", "status", "created_at", "updated_at",
		}).
			AddRow("RUB12345678", "Account 1", userID, "RUB", 4125.0, 0, 0, "active", now, now).
			AddRow("USD12345678", "Account 2", userID, "USD", 123.0, 0, 0, "blocked", now, now)

		r.MockDB.ExpectQuery(`SELECT code, name, user_id, currency, balance,
		       frozen_balance, reserve_balance, status, created_at, updated_at
		FROM accounts 
		WHERE user_id = \$1
		LIMIT \$2 OFFSET \$3`).
			WithArgs(userID, 10, 0).
			WillReturnRows(rows)

		result, err := r.Repo.GetAccounts(context.Background(), &models.GetAccountsRequest{
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

		r.MockDB.ExpectQuery(`SELECT code, name, user_id, currency, balance, 
		       frozen_balance, reserve_balance, status, created_at, updated_at
		FROM accounts 
		WHERE user_id = \$1
		LIMIT \$2 OFFSET \$3`).
			WithArgs(userID, 10, 0).
			WillReturnError(sql.ErrNoRows)

		_, err := r.Repo.GetAccounts(context.Background(), &models.GetAccountsRequest{
			UserID: userID,
			Limit:  10,
			Offset: 0,
		})

		assert.Error(t, err)
		assert.Equal(t, apperr.ErrAccountNotFound, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestAccountRepository_FreezeBalance(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		eventID := generateTestUUID(t)
		transactionID := generateTestUUID(t)
		now := time.Now()
		expectedPayload := `{"code":"RUB12345678","amount":"6.6","currency":"RUB"}`

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE accounts 
		SET frozen_balance = frozen_balance \+ \$1, 
		    balance = balance \- \$1,
		    updated_at = \$2
		WHERE code = \$3 
			AND balance >= \$1 
			AND status = 'active'`).
			WithArgs(6.6, sqlmock.AnyArg(), "RUB12345678").
			WillReturnResult(sqlmock.NewResult(0, 1))
		r.MockDB.ExpectExec("INSERT INTO events").
			WithArgs(
				eventID,
				transactionID,
				"freeze.request",
				"pending",
				"transaction-service",
				now,
				[]byte(expectedPayload),
			).WillReturnResult(sqlmock.NewResult(1, 1))

		r.MockDB.ExpectCommit()

		err := r.Repo.FreezeBalance(context.Background(),
			&models.BalanceRequest{
				Code:          "RUB12345678",
				Amount:        6.6,
				TransactionID: transactionID,
			},
			&models.CreateEventRequest{
				Event: &models.Event{
					ID:            eventID,
					TransactionID: transactionID,
					Type:          "freeze.request",
					Status:        "pending",
					Source:        "transaction-service",
					CreatedAt:     now,
					Payload:       []byte(expectedPayload),
				},
			})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})

	t.Run("insufficient funds", func(t *testing.T) {
		r := new(t)
		eventID := generateTestUUID(t)
		transactionID := generateTestUUID(t)
		now := time.Now()
		expectedPayload := `{"code":"RUB12345678","amount":"6.6","currency":"RUB"}`

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE accounts 
		SET frozen_balance = frozen_balance \+ \$1, 
		    balance = balance \- \$1,
		    updated_at = \$2
		WHERE code = \$3 
			AND balance >= \$1 
			AND status = 'active'`).
			WithArgs(6.6, sqlmock.AnyArg(), "RUB12345678").
			WillReturnResult(sqlmock.NewResult(0, 0))
		r.MockDB.ExpectRollback()

		err := r.Repo.FreezeBalance(context.Background(),
			&models.BalanceRequest{
				Code:          "RUB12345678",
				Amount:        6.6,
				TransactionID: transactionID,
			},
			&models.CreateEventRequest{
				Event: &models.Event{
					ID:            eventID,
					TransactionID: transactionID,
					Type:          "freeze.request",
					Status:        "pending",
					Source:        "transaction-service",
					CreatedAt:     now,
					Payload:       []byte(expectedPayload),
				},
			})

		assert.Error(t, err)
		assert.True(t, errors.Is(err, apperr.ErrAccountUpdateFailed), "Expected ErrAccountUpdateFailed, got: %v", err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestAccountRepository_UnfreezeBalance(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		eventID := generateTestUUID(t)
		transactionID := generateTestUUID(t)
		now := time.Now()
		expectedPayload := `{"code":"RUB12345678","amount":"6.6","currency":"RUB"}`

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE accounts 
		SET frozen_balance = frozen_balance \- \$1, 
		    balance = balance \+ \$1,
		    updated_at = \$2
		WHERE code = \$3 
			AND frozen_balance >= \$1 
			AND status IN \('active', 'blocked'\)`).
			WithArgs(6.6, sqlmock.AnyArg(), "RUB12345678").
			WillReturnResult(sqlmock.NewResult(0, 1))
		r.MockDB.ExpectExec("INSERT INTO events").
			WithArgs(
				eventID,
				transactionID,
				"unfreeze.request",
				"pending",
				"transaction-service",
				now,
				[]byte(expectedPayload),
			).WillReturnResult(sqlmock.NewResult(1, 1))

		r.MockDB.ExpectCommit()

		err := r.Repo.UnfreezeBalance(context.Background(),
			&models.BalanceRequest{
				Code:          "RUB12345678",
				Amount:        6.6,
				TransactionID: transactionID,
			},
			&models.CreateEventRequest{
				Event: &models.Event{
					ID:            eventID,
					TransactionID: transactionID,
					Type:          "unfreeze.request",
					Status:        "pending",
					Source:        "transaction-service",
					CreatedAt:     now,
					Payload:       []byte(expectedPayload),
				},
			})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestAccountRepository_ReserveDeposit(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		eventID := generateTestUUID(t)
		transactionID := generateTestUUID(t)
		now := time.Now()
		expectedPayload := `{"code":"RUB12345678","amount":"6.6","currency":"RUB"}`

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE accounts 
        SET reserve_balance = reserve_balance \+ \$1, 
            updated_at = \$2
        WHERE code = \$3 
            AND status = 'active'`).
			WithArgs(6.6, sqlmock.AnyArg(), "RUB12345678").
			WillReturnResult(sqlmock.NewResult(0, 1))
		r.MockDB.ExpectExec("INSERT INTO events").
			WithArgs(
				eventID,
				transactionID,
				"reserve.request",
				"pending",
				"transaction-service",
				now,
				[]byte(expectedPayload),
			).WillReturnResult(sqlmock.NewResult(1, 1))

		r.MockDB.ExpectCommit()

		err := r.Repo.ReserveDeposit(context.Background(),
			&models.BalanceRequest{
				Code:          "RUB12345678",
				Amount:        6.6,
				TransactionID: transactionID,
			},
			&models.CreateEventRequest{
				Event: &models.Event{
					ID:            eventID,
					TransactionID: transactionID,
					Type:          "reserve.request",
					Status:        "pending",
					Source:        "transaction-service",
					CreatedAt:     now,
					Payload:       []byte(expectedPayload),
				},
			})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})

	t.Run("err", func(t *testing.T) {
		r := new(t)
		eventID := generateTestUUID(t)
		transactionID := generateTestUUID(t)
		now := time.Now()
		expectedPayload := `{"code":"RUB12345678","amount":"6.6","currency":"RUB"}`

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE accounts 
        SET reserve_balance = reserve_balance \+ \$1, 
            updated_at = \$2
        WHERE code = \$3 
            AND status = 'active'`).
			WithArgs(6.6, sqlmock.AnyArg(), "RUB12345678").
			WillReturnResult(sqlmock.NewResult(0, 0))
		r.MockDB.ExpectRollback()

		err := r.Repo.ReserveDeposit(context.Background(),
			&models.BalanceRequest{
				Code:          "RUB12345678",
				Amount:        6.6,
				TransactionID: transactionID,
			},
			&models.CreateEventRequest{
				Event: &models.Event{
					ID:            eventID,
					TransactionID: transactionID,
					Type:          "reserve.request",
					Status:        "pending",
					Source:        "transaction-service",
					CreatedAt:     now,
					Payload:       []byte(expectedPayload),
				},
			})

		assert.Error(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestAccountRepository_UnreserveDeposit(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		eventID := generateTestUUID(t)
		transactionID := generateTestUUID(t)
		now := time.Now()
		expectedPayload := `{"code":RUB12345678,"amount":6.6,"currency":"RUB"}`

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE accounts 
        SET reserve_balance = reserve_balance - \$1,
            updated_at = \$2
        WHERE code = \$3 
            AND reserve_balance >= \$1
            AND status IN \('active', 'blocked'\)`).
			WithArgs(6.6, sqlmock.AnyArg(), "RUB12345678").
			WillReturnResult(sqlmock.NewResult(0, 1))
		r.MockDB.ExpectExec("INSERT INTO events").
			WithArgs(
				eventID,
				transactionID,
				"unreserve.request",
				"pending",
				"transaction-service",
				now,
				[]byte(expectedPayload),
			).WillReturnResult(sqlmock.NewResult(1, 1))

		r.MockDB.ExpectCommit()

		err := r.Repo.UnreserveDeposit(context.Background(),
			&models.BalanceRequest{
				Code:          "RUB12345678",
				Amount:        6.6,
				TransactionID: transactionID,
			},
			&models.CreateEventRequest{
				Event: &models.Event{
					ID:            eventID,
					TransactionID: transactionID,
					Type:          "unreserve.request",
					Status:        "pending",
					Source:        "transaction-service",
					CreatedAt:     now,
					Payload:       []byte(expectedPayload),
				},
			})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestAccountRepository_WithdrawFunds(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		eventID := generateTestUUID(t)
		transactionID := generateTestUUID(t)
		now := time.Now()
		expectedPayload := `{"code":RUB12345678,"amount":6.6,"currency":"RUB"}`

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE accounts 
        SET frozen_balance = frozen_balance \- \$1,
            updated_at = \$2
        WHERE code = \$3 
			AND frozen_balance >= \$1
			AND status = 'active'`).
			WithArgs(6.6, sqlmock.AnyArg(), "RUB12345678").
			WillReturnResult(sqlmock.NewResult(0, 1))
		r.MockDB.ExpectExec("INSERT INTO events").
			WithArgs(
				eventID,
				transactionID,
				"withdraw.request",
				"pending",
				"transaction-service",
				now,
				[]byte(expectedPayload),
			).WillReturnResult(sqlmock.NewResult(1, 1))

		r.MockDB.ExpectCommit()

		err := r.Repo.WithdrawBalance(context.Background(),
			&models.BalanceRequest{
				Code:          "RUB12345678",
				Amount:        6.6,
				TransactionID: transactionID,
			},
			&models.CreateEventRequest{
				Event: &models.Event{
					ID:            eventID,
					TransactionID: transactionID,
					Type:          "withdraw.request",
					Status:        "pending",
					Source:        "transaction-service",
					CreatedAt:     now,
					Payload:       []byte(expectedPayload),
				},
			})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestAccountRepository_DepositFunds(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		eventID := generateTestUUID(t)
		transactionID := generateTestUUID(t)
		now := time.Now()
		expectedPayload := `{"code":"RUB12345678","amount":"6.6","currency":"RUB"}`

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE accounts 
		SET reserve_balance = reserve_balance \- \$1,
			balance = balance \+ \$1,
		    updated_at = \$2
		WHERE code = \$3 
			AND reserve_balance >= \$1
			AND status = 'active'`).
			WithArgs(6.6, sqlmock.AnyArg(), "RUB12345678").
			WillReturnResult(sqlmock.NewResult(0, 1))
		r.MockDB.ExpectExec("INSERT INTO events").
			WithArgs(
				eventID,
				transactionID,
				"deposit.request",
				"pending",
				"transaction-service",
				now,
				[]byte(expectedPayload),
			).WillReturnResult(sqlmock.NewResult(1, 1))

		r.MockDB.ExpectCommit()

		err := r.Repo.DepositBalance(context.Background(),
			&models.BalanceRequest{
				Code:          "RUB12345678",
				Amount:        6.6,
				TransactionID: transactionID,
			},
			&models.CreateEventRequest{
				Event: &models.Event{
					ID:            eventID,
					TransactionID: transactionID,
					Type:          "deposit.request",
					Status:        "pending",
					Source:        "transaction-service",
					CreatedAt:     now,
					Payload:       []byte(expectedPayload),
				},
			})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestAccountRepository_BlockAccountWithEvent(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)
		eventID := generateTestUUID(t)
		transactionID := generateTestUUID(t)
		now := time.Now()
		expectedPayload := `{"code":"RUB12345678","amount":"6.6","currency":"RUB"}`

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`
		UPDATE accounts 
		SET status = \$1, 
		    updated_at = \$2,
			balance = balance \+ frozen_balance,
			frozen_balance = 0
		WHERE code = \$3 
	`).
			WithArgs(models.Blocked, sqlmock.AnyArg(), "RUB12345678").
			WillReturnResult(sqlmock.NewResult(0, 1))
		r.MockDB.ExpectExec("INSERT INTO events").
			WithArgs(
				eventID,
				transactionID,
				"blockaccount.request",
				"pending",
				"transaction-service",
				now,
				[]byte(expectedPayload),
			).WillReturnResult(sqlmock.NewResult(1, 1))

		r.MockDB.ExpectCommit()

		err := r.Repo.BlockAccountWithEvent(context.Background(), &models.UpdateAccountRequest{
			Code: "RUB12345678",
		},
			&models.CreateEventRequest{
				Event: &models.Event{
					ID:            eventID,
					TransactionID: transactionID,
					Type:          "blockaccount.request",
					Status:        "pending",
					Source:        "transaction-service",
					CreatedAt:     now,
					Payload:       []byte(expectedPayload),
				},
			})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}

func TestAccountRepository_CloseAccount(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := new(t)

		r.MockDB.ExpectBegin()
		r.MockDB.ExpectExec(`UPDATE accounts 
		SET status = \$1, 
		    updated_at = \$2,
		WHERE code = \$3 
			AND balance = 0
    		AND frozen_balance = 0
			AND reserve_balance = 0`).
			WithArgs(models.Closed, sqlmock.AnyArg(), "RUB12345678").
			WillReturnResult(sqlmock.NewResult(0, 1))
		r.MockDB.ExpectCommit()

		err := r.Repo.CloseAccount(context.Background(), &models.UpdateAccountRequest{
			Code: "RUB12345678",
		})

		assert.NoError(t, err)
		assert.NoError(t, r.MockDB.ExpectationsWereMet())
	})
}
