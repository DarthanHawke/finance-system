package repository

/* legacy code
import (
	billingerr "billing-service/internal/lib/errors"
	"billing-service/internal/models"
	"encoding/json"
	"strings"

	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type AccountRepository struct {
	db    *Database
	cache RedisCacheManagment
}

func NewAccountRepository(db *Database, cache RedisCacheManagment) *AccountRepository {
	return &AccountRepository{
		db:    db,
		cache: cache,
	}
}

// CreateAccount создает счет с именем
func (r *AccountRepository) CreateAccount(
	ctx context.Context,
	accountCode string,
	userID uuid.UUID,
	currency string,
	name string,
) error {
	const op = "storage.AccountRepository.CreateAccount"

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {

		_, err := tx.ExecContext(
			ctx,
			`INSERT INTO currency_accounts (account_code, user_id, currency, account_name, created_at, updated_at)
         VALUES ($1, $2, $3, $4, NOW(), NOW())`,
			accountCode, userID, currency, name)
		if err != nil {
			if strings.Contains(err.Error(), "23505") {
				return fmt.Errorf("%s: %w", op, billingerr.ErrAccountCodeUnique)
			}
			return fmt.Errorf("%s: %w", op, err)
		}

		return nil
	})

	_ = r.cache.Delete(ctx, fmt.Sprintf("user_accounts:%v", userID))

	return err
}

// GetAccount возвращает полную информацию о счете.
func (r *AccountRepository) GetAccount(
	ctx context.Context,
	account_code string,
) (*models.CurrencyAccount, error) {
	const op = "storage.AccountRepository.GetAccount"

	var account models.CurrencyAccount
	err := r.db.GetContext(
		ctx,
		&account,
		`SELECT account_code, account_name, user_id, currency, balance, blocked_amount, created_at, updated_at
         FROM currency_accounts
         WHERE account_code = $1`,
		account_code,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, billingerr.ErrAccountNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &account, nil
}

// GetAccountId возвращает UUID счета
func (r *AccountRepository) GetAccountId(
	ctx context.Context,
	account_code string,
) (uuid.UUID, error) {
	const op = "storage.AccountRepository.GetAccountId"

	var account_id uuid.UUID
	err := r.db.GetContext(
		ctx,
		&account_id,
		`SELECT id
         FROM currency_accounts
         WHERE account_code = $1`,
		account_code,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, billingerr.ErrAccountNotFound
		}
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return account_id, nil
}

// GetUserAccounts возвращает все счета пользователя
func (r *AccountRepository) GetUserAccounts(ctx context.Context, userID uuid.UUID) ([]models.CurrencyAccount, error) {
	const op = "storage.AccountRepository.GetUserAccounts"

	cacheKey := fmt.Sprintf("user_accounts:%s", userID)
	var accounts []models.CurrencyAccount

	// Проверяем кеш
	cachedData, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		if err := json.Unmarshal([]byte(cachedData), &accounts); err == nil {
			return accounts, nil
		}
	}

	err = r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &accounts,
			`SELECT account_code, account_name, user_id, currency, balance, blocked_amount, created_at, updated_at
             FROM currency_accounts
             WHERE user_id = $1
             ORDER BY created_at`,
			userID)
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Сохраняем в кэш
	if data, err := json.Marshal(accounts); err == nil {
		_ = r.cache.Set(ctx, cacheKey, string(data))
	}

	return accounts, nil
}

// GetBalance возвращает баланс по валюте
func (r *AccountRepository) GetBalance(ctx context.Context, account_code string) (float64, error) {
	const op = "storage.AccountRepository.GetBalance"

	cacheKey := fmt.Sprintf("account_balance:%v", account_code)
	var balance float64

	// Проверяем кеш
	cachedBalance, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		if _, err := fmt.Sscanf(cachedBalance, "%f", &balance); err == nil {
			return balance, nil
		}
	}

	err = r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &balance,
			`SELECT balance FROM currency_accounts
             WHERE account_code = $1`,
			account_code)
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("%s: %w", op, billingerr.ErrAccountNotFound)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	// Сохраняем в кэш
	_ = r.cache.Set(ctx, cacheKey, fmt.Sprintf("%.2f", balance))
	return balance, nil
}
*/
