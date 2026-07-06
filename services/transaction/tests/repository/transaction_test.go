package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"transaction-service/internal/lib/errors/apperr"
	"transaction-service/internal/models"
	repository "transaction-service/internal/repository/postgres"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testRepository struct {
	TransactionRepository *repository.TransactionRepository
	EventRepository       *repository.EventRepository
	SagaRepository        *repository.SagaRepository
	DB                    *repository.Database
	Mock                  sqlmock.Sqlmock
}

func newTestRepository(t *testing.T) *testRepository {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	database := &repository.Database{DB: sqlxDB}

	t.Cleanup(func() { db.Close() })

	return &testRepository{
		TransactionRepository: repository.NewTransactionRepository(database),
		EventRepository:       repository.NewEventRepository(database),
		SagaRepository:        repository.NewSagaRepository(database),
		DB:                    database,
		Mock:                  mock,
	}
}

func TestUpdateSagaState_Success(t *testing.T) {
	r := newTestRepository(t)
	txID := uuid.New()
	now := time.Now()

	r.Mock.ExpectBegin()

	tx, err := r.DB.BeginTxx(context.Background(), nil)
	require.NoError(t, err)

	r.Mock.ExpectExec(`UPDATE transactions SET saga_state`).
		WithArgs("SENDER_FREEZING", nil, now, txID, "INIT").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r.Mock.ExpectCommit()

	status := (*string)(nil)
	err = r.TransactionRepository.UpdateSagaState(context.Background(), tx, &models.UpdateSagaStateRequest{
		TransactionID: txID,
		ExpectedFrom:  "INIT",
		SagaState:     "SENDER_FREEZING",
		Status:        status,
		UpdatedAt:     now,
	})
	assert.NoError(t, err)

	err = tx.Commit()
	assert.NoError(t, err)
	assert.NoError(t, r.Mock.ExpectationsWereMet())
}

func TestUpdateSagaState_WrongExpectedState(t *testing.T) {
	r := newTestRepository(t)
	txID := uuid.New()
	now := time.Now()

	r.Mock.ExpectBegin()
	tx, err := r.DB.BeginTxx(context.Background(), nil)
	require.NoError(t, err)

	r.Mock.ExpectExec(`UPDATE transactions SET saga_state`).
		WithArgs("SENDER_FREEZING", nil, now, txID, "INIT").
		WillReturnResult(sqlmock.NewResult(0, 0))

	rows := sqlmock.NewRows([]string{"saga_state"}).AddRow("ALREADY_FREEZING")
	r.Mock.ExpectQuery(`SELECT saga_state FROM transactions WHERE id = \$1`).
		WithArgs(txID).
		WillReturnRows(rows)

	err = r.TransactionRepository.UpdateSagaState(context.Background(), tx, &models.UpdateSagaStateRequest{
		TransactionID: txID,
		ExpectedFrom:  "INIT",
		SagaState:     "SENDER_FREEZING",
		Status:        nil,
		UpdatedAt:     now,
	})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrInvalidSagaState))

	_ = tx.Rollback()
	assert.NoError(t, r.Mock.ExpectationsWereMet())
}

func TestUpdateSagaState_AlreadyAdvanced(t *testing.T) {
	r := newTestRepository(t)
	txID := uuid.New()
	now := time.Now()

	r.Mock.ExpectBegin()
	tx, _ := r.DB.BeginTxx(context.Background(), nil)

	r.Mock.ExpectExec(`UPDATE transactions SET saga_state`).
		WithArgs("SENDER_FREEZING", nil, now, txID, "INIT").
		WillReturnResult(sqlmock.NewResult(0, 0))

	rows := sqlmock.NewRows([]string{"saga_state"}).AddRow("SENDER_FREEZING")
	r.Mock.ExpectQuery(`SELECT saga_state FROM transactions WHERE id = \$1`).
		WithArgs(txID).
		WillReturnRows(rows)

	err := r.TransactionRepository.UpdateSagaState(context.Background(), tx, &models.UpdateSagaStateRequest{
		TransactionID: txID,
		ExpectedFrom:  "INIT",
		SagaState:     "SENDER_FREEZING",
		Status:        nil,
		UpdatedAt:     now,
	})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrSagaAlreadyAdvanced))
}

func TestUpdateSagaState_TransactionNotFound(t *testing.T) {
	r := newTestRepository(t)
	txID := uuid.New()
	now := time.Now()

	r.Mock.ExpectBegin()

	tx, err := r.DB.BeginTxx(context.Background(), nil)
	require.NoError(t, err)

	r.Mock.ExpectExec(`UPDATE transactions SET saga_state`).
		WithArgs("SENDER_FREEZING", nil, now, txID, "INIT").
		WillReturnResult(sqlmock.NewResult(0, 0))

	r.Mock.ExpectQuery(`SELECT saga_state FROM transactions WHERE id =`).
		WithArgs(txID).
		WillReturnError(sql.ErrNoRows)

	err = r.TransactionRepository.UpdateSagaState(context.Background(), tx, &models.UpdateSagaStateRequest{
		TransactionID: txID,
		ExpectedFrom:  "INIT",
		SagaState:     "SENDER_FREEZING",
		Status:        nil,
		UpdatedAt:     now,
	})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrTransactionNotFound))

	_ = tx.Rollback()
	assert.NoError(t, r.Mock.ExpectationsWereMet())
}

func TestGetPendingEvents_Success(t *testing.T) {
	r := newTestRepository(t)
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "transaction_id", "partition_key", "event_type", "event_status", "source",
		"trace_id", "span_id", "created_at", "processed_at", "payload",
	}).AddRow(
		uuid.New(), uuid.New(), "key1", "freeze.request", "pending", "transaction-service",
		"trace1", "span1", now, nil, []byte(`{}`),
	).AddRow(
		uuid.New(), uuid.New(), "key2", "credit.request", "pending", "transaction-service",
		"trace2", "span2", now, nil, []byte(`{}`),
	)

	r.Mock.ExpectQuery(`UPDATE events SET processed_at`).
		WithArgs(sqlmock.AnyArg(), "pending", sqlmock.AnyArg(), 10).
		WillReturnRows(rows)

	resp, err := r.EventRepository.GetPendingEvents(context.Background(), &models.GetEventRequest{Limit: 10})
	assert.NoError(t, err)
	assert.Len(t, resp.Events, 2)
	assert.NoError(t, r.Mock.ExpectationsWereMet())
}

func TestGetPendingEvents_Empty(t *testing.T) {
	r := newTestRepository(t)

	rows := sqlmock.NewRows([]string{
		"id", "transaction_id", "partition_key", "event_type", "event_status", "source",
		"trace_id", "span_id", "created_at", "processed_at", "payload",
	})

	r.Mock.ExpectQuery(`UPDATE events SET processed_at`).
		WithArgs(sqlmock.AnyArg(), "pending", sqlmock.AnyArg(), 10).
		WillReturnRows(rows)

	resp, err := r.EventRepository.GetPendingEvents(context.Background(), &models.GetEventRequest{Limit: 10})
	assert.NoError(t, err)
	assert.Empty(t, resp.Events)
}

func TestInsertTransaction_Duplicate(t *testing.T) {
	r := newTestRepository(t)

	r.Mock.ExpectBegin()
	tx, _ := r.DB.BeginTxx(context.Background(), nil)

	r.Mock.ExpectExec(`INSERT INTO transactions`).
		WillReturnError(&pgconn.PgError{Code: "23505"})

	err := r.TransactionRepository.InsertTransaction(context.Background(), tx, &models.InsertTransactionRequest{
		Transaction: &models.Transaction{ID: uuid.New()},
	})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrTransactionIDNotUnique))
}

func TestInsertEvent_Duplicate(t *testing.T) {
	r := newTestRepository(t)

	r.Mock.ExpectBegin()
	tx, _ := r.DB.BeginTxx(context.Background(), nil)

	r.Mock.ExpectExec(`INSERT INTO events`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := r.EventRepository.InsertEvent(context.Background(), tx, &models.InsertEventRequest{
		Event: &models.Event{ID: uuid.New()},
	})
	assert.NoError(t, err)
}
