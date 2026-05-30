// Пакет postgres реализовывает работу с базами данных
package postgres

import (
	"context"
	"fmt"

	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// Database - обёртка для sqlx.DB
type Database struct {
	*sqlx.DB
}

// NewDatabase создаёт новое подключение к БД
func NewDatabase(dsn string) (*Database, error) {
	traceDriver, err := otelsql.Register("pgx",
		otelsql.WithAttributes(
			semconv.DBSystemPostgreSQL,
		),
		otelsql.WithTracerProvider(otel.GetTracerProvider()),
		otelsql.WithSpanOptions(
			otelsql.SpanOptions{
				Ping:                 true,
				RowsNext:             false,
				DisableErrSkip:       false,
				DisableQuery:         false,
				OmitConnResetSession: true,
				OmitConnPrepare:      true,
			},
		),
	)
	if err != nil {
		return nil, fmt.Errorf("register traced driver: %w", err)
	}
	sqlx.BindDriver(traceDriver, sqlx.DOLLAR)

	db, err := sqlx.Connect(traceDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return &Database{db}, nil
}

// CloseConnect закрывает подключение к БД
func (db *Database) CloseConnect() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// WithTransaction выполняет функцию в транзакции.
// Автоматически коммитит при успехе, откат при ошибке.
func (db *Database) WithTransaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tracer := otel.Tracer("postgres")
	ctx, span := tracer.Start(ctx, "postgres.transaction",
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
		),
	)
	defer span.End()

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		return err
	}
	defer func() {
		if p := recover(); p != nil { // перехватываем panic
			_ = tx.Rollback()
			panic(p) // пробрасываем panic дальше после отката
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		span.RecordError(err)
		return err
	}

	if err := tx.Commit(); err != nil {
		span.RecordError(err)
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
