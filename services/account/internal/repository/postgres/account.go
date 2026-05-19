// Пакет postgres предоставляет реализацию работы с базами данных
package postgres

import (
	"account-service/internal/lib/errors/apperr"
	"account-service/internal/models"
	"strings"
	"time"

	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// AccountRepository реализует методы доступа к данным для работы со счетами
type AccountRepository struct {
	db     *Database
	logger *zap.Logger
}

// NewAccountRepository создает новый экземпляр AccountRepository
func NewAccountRepository(db *Database, logger *zap.Logger) *AccountRepository {
	return &AccountRepository{
		db:     db,
		logger: logger.With(zap.String("component", "account")),
	}
}

// CreateAccount создает новый счет пользователя
func (r *AccountRepository) CreateAccount(ctx context.Context, req *models.CreateAccountRequest) error {
	const op = "repository.account.CreateAccount"

	query := `
		INSERT INTO accounts (code, name, user_id, currency, 
										created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		req.Code,
		req.Name,
		req.UserID,
		req.Currency,
		time.Now(),
		time.Now(),
	)

	if err != nil {
		if strings.Contains(err.Error(), "23505") {
			return apperr.ErrAccountCodeNotUnique
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetAccount возвращает полную информацию о счете
func (r *AccountRepository) GetAccount(
	ctx context.Context,
	req *models.GetAccountRequest,
) (models.GetAccountResponse, error) {
	const op = "repository.account.GetAccount"

	query := `SELECT code, name, user_id, currency, balance, frozen_balance, 
						reserve_balance, status, created_at, updated_at
         FROM accounts
         WHERE code = $1`

	var resp models.GetAccountResponse

	err := r.db.GetContext(ctx, &resp, query, req.Code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.GetAccountResponse{}, apperr.ErrAccountNotFound
		}
		return models.GetAccountResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	return resp, nil
}

// GetAccounts возвращает все счета пользователя
func (r *AccountRepository) GetAccounts(
	ctx context.Context,
	req *models.GetAccountsRequest,
) (models.GetAccountsResponse, error) {
	const op = "repository.account.GetAccounts"

	var resp models.GetAccountsResponse
	var accounts []models.Account

	query := `
		SELECT code, name, user_id, currency, balance, frozen_balance,
		       	reserve_balance, status, created_at, updated_at
		FROM accounts 
		WHERE user_id = $1
		LIMIT $2 OFFSET $3
	`

	err := r.db.SelectContext(ctx, &accounts, query, req.UserID, req.Limit, req.Offset)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.GetAccountsResponse{}, apperr.ErrAccountNotFound
		}
		return models.GetAccountsResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	resp = models.GetAccountsResponse{Accounts: accounts}

	return resp, nil
}

// BlockAccount размораживает средства и блокирует счет для проведения любых операций
func (r *AccountRepository) BlockAccount(ctx context.Context, req *models.UpdateAccountRequest) error {
	const op = "repository.account.BlockAccount"

	query := `
		UPDATE accounts 
		SET status = $1, 
		    updated_at = $2,
			balance = balance + frozen_balance,
			frozen_balance = 0
		WHERE code = $3 
	`

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, query, models.Blocked, time.Now(), req.Code)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return apperr.ErrAccountUpdateFailed
		}
		if rowsAffected != 1 {
			return fmt.Errorf("expected 1 row affected, got %d", rowsAffected)
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, apperr.ErrAccountUpdateFailed) {
			return err
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// CloseAccount закрывает счет если баланс нулевой
func (r *AccountRepository) CloseAccount(ctx context.Context, req *models.UpdateAccountRequest) error {
	const op = "repository.account.BlockAccount"

	query := `
		UPDATE accounts 
		SET status = $1, 
		    updated_at = $2,
		WHERE code = $3 
			AND balance = 0
    		AND frozen_balance = 0
			AND reserve_balance = 0
	`

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, query, models.Closed, time.Now(), req.Code)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return apperr.ErrAccountUpdateFailed
		}
		if rowsAffected != 1 {
			return fmt.Errorf("expected 1 row affected, got %d", rowsAffected)
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, apperr.ErrAccountUpdateFailed) {
			return err
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// createEvent создает событие
func (r *AccountRepository) createEvent(ctx context.Context, tx *sqlx.Tx, req *models.CreateEventRequest) error {
	const op = "repository.event.createEvent"

	const query = `
		INSERT INTO events (id, transaction_id, partition_key, type, status, source, created_at, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := tx.ExecContext(ctx, query,
		req.ID,
		req.TransactionID,
		req.PartitionKey,
		req.Type,
		req.Status,
		req.Source,
		req.CreatedAt,
		req.Payload,
	)

	if err != nil {
		if strings.Contains(err.Error(), "23505") {
			return apperr.ErrEventIDNotUnique
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// FreezeBalance замораживает запрашиваемую сумму для проведения различных транзакций
func (r *AccountRepository) FreezeBalance(
	ctx context.Context,
	req *models.BalanceRequest,
	event *models.CreateEventRequest,
) error {
	const op = "repository.account.FreezeBalance"

	query := `
		UPDATE accounts 
		SET frozen_balance = frozen_balance + $1, 
		    balance = balance - $1,
		    updated_at = $2
		WHERE code = $3 
			AND balance >= $1 
			AND status = 'active'
	`

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, query, req.Amount, time.Now(), req.Code)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return apperr.ErrAccountUpdateFailed
		}
		if rowsAffected != 1 {
			return fmt.Errorf("expected 1 row affected, got %d", rowsAffected)
		}

		if err = r.createEvent(ctx, tx, event); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, apperr.ErrAccountUpdateFailed) {
			return err
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// UnfreezeBalance размораживает заблокированную сумму при откате
func (r *AccountRepository) UnfreezeBalance(
	ctx context.Context,
	req *models.BalanceRequest,
	event *models.CreateEventRequest,
) error {
	const op = "repository.account.UnfreezeBalance"

	query := `
		UPDATE accounts 
		SET frozen_balance = frozen_balance - $1, 
		    balance = balance + $1,
		    updated_at = $2
		WHERE code = $3 
			AND frozen_balance >= $1
			AND status IN ('active', 'blocked')
	`

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, query, req.Amount, time.Now(), req.Code)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected != 1 {
			return fmt.Errorf("expected 1 row affected, got %d", rowsAffected)
		}

		if err = r.createEvent(ctx, tx, event); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// ReserveDeposit резервирует сумму для пополнения
func (r *AccountRepository) ReserveDeposit(
	ctx context.Context,
	req *models.BalanceRequest,
	event *models.CreateEventRequest,
) error {
	const op = "repository.account.ReserveDeposit"

	query := `
        UPDATE accounts 
        SET reserve_balance = reserve_balance + $1,
            updated_at = $2
        WHERE code = $3 
            AND status = 'active'
    `

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, query, req.Amount, time.Now(), req.Code)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected != 1 {
			return fmt.Errorf("expected 1 row affected, got %d", rowsAffected)
		}

		if err = r.createEvent(ctx, tx, event); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// UnreserveDeposit отменяет зарезервированное пополнение
func (r *AccountRepository) UnreserveDeposit(
	ctx context.Context,
	req *models.BalanceRequest,
	event *models.CreateEventRequest,
) error {
	const op = "repository.account.UnreserveDeposit"

	query := `
        UPDATE accounts 
        SET reserve_balance = reserve_balance - $1,
            updated_at = $2
        WHERE code = $3 
            AND reserve_balance >= $1
            AND status IN ('active', 'blocked')
    `

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, query, req.Amount, time.Now(), req.Code)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected != 1 {
			return fmt.Errorf("expected 1 row affected, got %d", rowsAffected)
		}

		if err = r.createEvent(ctx, tx, event); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// WithdrawBalance списывает с замороженных средств требуемую сумму
func (r *AccountRepository) WithdrawBalance(
	ctx context.Context,
	req *models.BalanceRequest,
	event *models.CreateEventRequest,
) error {
	const op = "repository.account.WithdrawBalance"

	query := `
        UPDATE accounts 
        SET frozen_balance = frozen_balance - $1,
            updated_at = $2
        WHERE code = $3 
			AND frozen_balance >= $1
			AND status = 'active'
    `

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, query, req.Amount, time.Now(), req.Code)
		if err != nil {
			return err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected != 1 {
			return fmt.Errorf("expected 1 row affected, got %d", rowsAffected)
		}

		if err = r.createEvent(ctx, tx, event); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// DepositBalance пополняет баланс счета на требуемую сумму
func (r *AccountRepository) DepositBalance(
	ctx context.Context,
	req *models.BalanceRequest,
	event *models.CreateEventRequest,
) error {
	const op = "repository.account.DepositBalance"

	query := `
		UPDATE accounts 
		SET reserve_balance = reserve_balance - $1,
			balance = balance + $1,
		    updated_at = $2
		WHERE code = $3 
			AND reserve_balance >= $1
			AND status = 'active'
	`

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, query, req.Amount, time.Now(), req.Code)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected != 1 {
			return fmt.Errorf("expected 1 row affected, got %d", rowsAffected)
		}

		if err = r.createEvent(ctx, tx, event); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// BlockAccountWithEvent размораживает средства и блокирует счет для проведения любых операций
func (r *AccountRepository) BlockAccountWithEvent(
	ctx context.Context,
	req *models.UpdateAccountRequest,
	event *models.CreateEventRequest,
) error {
	const op = "repository.account.BlockAccountWithEvent"

	query := `
		UPDATE accounts 
		SET status = $1, 
		    updated_at = $2,
			balance = balance + frozen_balance,
			frozen_balance = 0
		WHERE code = $3 
	`

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, query, models.Blocked, time.Now(), req.Code)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return apperr.ErrAccountUpdateFailed
		}
		if rowsAffected != 1 {
			return fmt.Errorf("expected 1 row affected, got %d", rowsAffected)
		}

		if err = r.createEvent(ctx, tx, event); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, apperr.ErrAccountUpdateFailed) {
			return err
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
