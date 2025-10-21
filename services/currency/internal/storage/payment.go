package storage

import (
	billingerr "billing-service/internal/lib/errors"
	"billing-service/internal/models"

	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type TransactionRepository struct {
	db    *Database
	cache RedisCacheManagment
}

func NewTransactionRepository(db *Database, cache RedisCacheManagment) *TransactionRepository {
	return &TransactionRepository{
		db:    db,
		cache: cache,
	}
}

// UpdateCurrencyRate обновляет курс валюты
func (r *TransactionRepository) UpdateCurrencyRate(
	ctx context.Context,
	fromCurrency, toCurrency string,
	rate float64,
) error {
	const op = "storage.TransactionRepository.UpdateCurrencyRate"

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO currency_rates (from_currency, to_currency, rate)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (from_currency, to_currency) 
			 DO UPDATE SET rate = EXCLUDED.rate, updated_at = NOW()`,
			fromCurrency, toCurrency, rate)
		return err
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Инвалидируем кэш
	cacheKey := fmt.Sprintf("rate:%s:%s", fromCurrency, toCurrency)
	_ = r.cache.Delete(ctx, cacheKey)
	return nil
}

// ConvertCurrency конвертирует средства между валютами (сага-паттерн)
func (r *TransactionRepository) ConvertCurrency(
	ctx context.Context,
	operationID uuid.UUID,
	fromAccount, toAccount *models.CurrencyAccount,
	amount float64,
	description string,
) error {
	const op = "storage.TransactionRepository.ConvertCurrency"

	if amount <= 0 {
		return fmt.Errorf("%s: %w", op, billingerr.ErrInvalidAmount)
	}

	// Получаем курс обмена
	rate, err := r.GetCurrencyRate(ctx, fromAccount.Currency, toAccount.Currency)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Рассчитываем сумму в целевой валюте
	convertedAmount := amount * rate
	convertedAmount = math.Round(convertedAmount*100) / 100 // Округляем до копеек

	// Генерируем ID для компенсирующих операций
	compOperationID := uuid.New()
	withdrawOpID := uuid.New()
	depositOpID := uuid.New()

	err = r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		// Проверяем, не выполнялась ли уже эта операция
		var existingOpID uuid.UUID
		err := tx.GetContext(ctx, &existingOpID,
			`SELECT operation_id FROM balance_operations 
             WHERE operation_id = $1 FOR UPDATE`,
			operationID)
		if err == nil {
			return nil // Операция уже выполнена
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		// 1. Записываем начальное состояние саги
		_, err = tx.ExecContext(ctx,
			`INSERT INTO balance_operations (
                operation_id, account_code, user_id, currency, amount, 
                new_balance, operation_type, status, description
             ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			operationID, fromAccount.AccountCode, fromAccount.UserID, fromAccount.Currency, amount, fromAccount.Balance,
			"CONVERSION", "PENDING", description+fmt.Sprintf(`{"to_account_id":%v,"to_currency":"%s","rate":%f}`,
				toAccount.AccountCode, toAccount.Currency, rate))
		if err != nil {
			return err
		}

		// 2. Выполняем списание в исходной валюте
		if err := r.withdrawInTx(tx, withdrawOpID, fromAccount.AccountCode, amount, uuid.Nil, "Convert to "+toAccount.Currency); err != nil {
			// Записываем неудачу
			_, _ = tx.ExecContext(ctx,
				`UPDATE balance_operations SET status = 'FAILED' 
                 WHERE operation_id = $1`,
				operationID)
			return err
		}

		// 3. Выполняем зачисление в целевой валюте
		if err := r.depositInTx(tx, depositOpID, toAccount.AccountCode, convertedAmount, uuid.Nil, "Convert from "+fromAccount.Currency); err != nil {
			// Компенсируем списание
			r.depositInTx(tx, compOperationID, fromAccount.AccountCode, amount, uuid.Nil, "Compensation for failed conversion")

			// Записываем неудачу
			_, _ = tx.ExecContext(ctx,
				`UPDATE balance_operations SET status = 'FAILED' 
                 WHERE operation_id = $1`,
				operationID)
			return err
		}

		// 4. Помечаем операцию как завершенную
		_, err = tx.ExecContext(ctx,
			`UPDATE balance_operations SET status = 'COMPLETED', processed_at = NOW() 
             WHERE operation_id = $1`,
			operationID)
		return err
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Инвалидируем кэш балансов
	r.invalidateAccountCache(ctx, fromAccount.AccountCode)
	r.invalidateAccountCache(ctx, toAccount.AccountCode)
	r.invalidateUserAccountsCache(ctx, fromAccount.UserID.String())
	r.invalidateUserAccountsCache(ctx, toAccount.UserID.String())
	return nil
}

// GetPendingOperations возвращает список необработанных операций
func (r *TransactionRepository) GetPendingOperations(
	ctx context.Context,
	batchSize int,
) ([]models.BalanceOperation, error) {
	const op = "storage.TransactionRepository.GetPendingOperations"

	var operations []models.BalanceOperation

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &operations,
			`SELECT id, operation_id, account_code, user_id, currency, amount, 
					new_balance, operation_type, status, transaction_id, 
					description, created_at, processed_at
			 FROM balance_operations 
			 WHERE status = 'PENDING'
			 ORDER BY created_at
			 LIMIT $1`,
			batchSize)
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return operations, nil
}

// GetCurrencyRates возвращает курс обмена между валютами
func (r *TransactionRepository) GetCurrencyRate(ctx context.Context, fromCurrency, toCurrency string) (float64, error) {
	const op = "storage.TransactionRepository.GetCurrencyRate"

	if fromCurrency == toCurrency {
		return 1.0, nil
	}

	cacheKey := fmt.Sprintf("rate:%s:%s", fromCurrency, toCurrency)
	var rate float64

	// Проверяем кеш
	cachedRate, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		if _, err := fmt.Sscanf(cachedRate, "%f", &rate); err == nil {
			return rate, nil
		}
	}

	err = r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &rate,
			`SELECT rate FROM currency_rates 
			 WHERE from_currency = $1 AND to_currency = $2`,
			fromCurrency, toCurrency)
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("%s: %w", op, billingerr.ErrCurrencyRateNotFound)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	// Сохраняем в кэш
	_ = r.cache.Set(ctx, cacheKey, fmt.Sprintf("%.6f", rate))
	return rate, nil
}

// Вспомогательные методы для работы внутри транзакции
func (r *TransactionRepository) withdrawInTx(tx *sqlx.Tx,
	operationID uuid.UUID,
	accountCode string,
	amount float64,
	transactionID uuid.UUID,
	description string,
) error {
	// Получаем информацию о счете
	var userID uuid.UUID
	var currency string
	err := tx.QueryRowContext(context.Background(),
		`SELECT user_id, currency FROM currency_accounts 
     WHERE account_code = $1 FOR UPDATE`,
		accountCode,
	).Scan(&userID, &currency)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return billingerr.ErrAccountNotFound
		}
		return err
	}

	// Проверяем достаточность средств
	var currentBalance float64
	err = tx.GetContext(context.Background(), &currentBalance,
		`SELECT balance FROM currency_accounts 
         WHERE account_code = $1 FOR UPDATE`,
		accountCode)
	if err != nil {
		return err
	}

	if currentBalance < amount {
		return billingerr.ErrInsufficientFunds
	}

	// Обновляем баланс
	var newBalance float64
	err = tx.GetContext(context.Background(), &newBalance,
		`UPDATE currency_accounts 
         SET balance = balance - $1, updated_at = NOW()
         WHERE account_code = $2
         RETURNING balance`,
		amount, accountCode)
	if err != nil {
		return err
	}

	// Записываем операцию в историю
	_, err = tx.ExecContext(context.Background(),
		`INSERT INTO balance_operations (
            operation_id, account_code, user_id, currency, amount, 
            new_balance, operation_type, status, 
            transaction_id, description, processed_at
         ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())`,
		operationID, accountCode, userID, currency, amount, newBalance,
		"WITHDRAWAL", "COMPLETED", transactionID, description)
	return err
}

func (r *TransactionRepository) depositInTx(tx *sqlx.Tx,
	operationID uuid.UUID,
	accountCode string,
	amount float64,
	transactionID uuid.UUID,
	description string,
) error {
	// Получаем информацию о счете
	var userID uuid.UUID
	var currency string
	err := tx.QueryRowContext(context.Background(),
		`SELECT user_id, currency FROM currency_accounts 
     WHERE account_code = $1 FOR UPDATE`,
		accountCode,
	).Scan(&userID, &currency)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return billingerr.ErrAccountNotFound
		}
		return err
	}

	// Обновляем баланс
	var newBalance float64
	err = tx.GetContext(context.Background(), &newBalance,
		`UPDATE currency_accounts 
         SET balance = balance + $1, updated_at = NOW()
         WHERE account_code = $2
         RETURNING balance`,
		amount, accountCode)
	if err != nil {
		return err
	}

	// Записываем операцию в историю
	_, err = tx.ExecContext(context.Background(),
		`INSERT INTO balance_operations (
            operation_id, account_code, user_id, currency, amount, 
            new_balance, operation_type, status, 
            transaction_id, description, processed_at
         ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())`,
		operationID, accountCode, userID, currency, amount, newBalance,
		"DEPOSIT", "COMPLETED", transactionID, description)
	return err
}

func (r *TransactionRepository) invalidateAccountCache(ctx context.Context, accountCode string) {
	cacheKeys := []string{
		fmt.Sprintf("account:%v", accountCode),
		fmt.Sprintf("account_balance:%v", accountCode),
	}
	for _, key := range cacheKeys {
		_ = r.cache.Delete(ctx, key)
	}
	_ = r.cache.DeleteByPrefix(ctx, fmt.Sprintf("operations:%v:", accountCode))
}

func (r *TransactionRepository) invalidateUserAccountsCache(ctx context.Context, userID string) {
	cacheKeys := []string{
		fmt.Sprintf("user_accounts:%v", userID),
	}
	for _, key := range cacheKeys {
		_ = r.cache.Delete(ctx, key)
	}
}
