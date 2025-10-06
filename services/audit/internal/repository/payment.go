package repository

import (
	billingerr "billing-service/internal/lib/errors"
	"billing-service/internal/models"
	"encoding/json"

	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PaymentRepository struct {
	db    *Database
	cache RedisCacheManagment
}

func NewPaymentRepository(db *Database, cache RedisCacheManagment) *PaymentRepository {
	return &PaymentRepository{
		db:    db,
		cache: cache,
	}
}

// Transfer выполняет перевод между счетами (сага-паттерн)
func (r *PaymentRepository) Transfer(
	ctx context.Context,
	operationID uuid.UUID,
	fromAccount, toAccount *models.CurrencyAccount,
	amount float64,
	paymentID uuid.UUID,
	description string,
) error {
	const op = "repository.PaymentRepository.Transfer"

	if amount <= 0 {
		return fmt.Errorf("%s: %w", op, billingerr.ErrInvalidAmount)
	}

	// Генерируем ID для компенсирующих операций
	compensationID := uuid.New()
	withdrawOpID := uuid.New()
	depositOpID := uuid.New()

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		// 1. Проверяем, не выполнялась ли уже эта операция
		var existingOpID uuid.UUID
		err := tx.GetContext(ctx, &existingOpID,
			`SELECT operation_id FROM balance_operations 
             WHERE operation_id = $1 FOR UPDATE`,
			operationID)
		if err == nil {
			return nil // Операция уже выполнена
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check existing operation failed: %w", err)
		}

		// 2. Записываем начальное состояние саги
		_, err = tx.ExecContext(ctx,
			`INSERT INTO balance_operations (
                operation_id, account_code, user_id, currency, amount, new_balance,
                operation_type, status, payment_id, description, created_at
             ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())`,
			operationID,
			fromAccount.AccountCode,
			fromAccount.UserID,
			fromAccount.Currency,
			amount,
			fromAccount.Balance,
			"TRANSFER",
			"PENDING",
			paymentID,
			description+fmt.Sprintf(`{"from_account_id":%v,"to_account_id":%v}`, fromAccount.AccountCode, toAccount.AccountCode),
		)
		if err != nil {
			return fmt.Errorf("insert initial operation failed: %w", err)
		}

		// 3. Списание со счета отправителя
		if err := r.withdrawInTx(tx, withdrawOpID, fromAccount.AccountCode, amount, paymentID,
			fmt.Sprintf("Transfer to account %v", toAccount.AccountCode)); err != nil {

			// Помечаем операцию как неудачную
			if markErr := r.markOperationFailed(tx, operationID, err.Error()); markErr != nil {
				return fmt.Errorf("mark operation failed error: %v, original error: %w", markErr, err)
			}
			return fmt.Errorf("withdraw failed: %w", err)
		}

		// 4. Зачисление на счет получателя
		if err := r.depositInTx(tx, depositOpID, toAccount.AccountCode, amount, paymentID,
			fmt.Sprintf("Transfer from account %v", fromAccount.AccountCode)); err != nil {

			// Компенсируем списание (возвращаем средства)
			if compErr := r.depositInTx(tx, compensationID, fromAccount.AccountCode, amount, paymentID,
				fmt.Sprintf("Compensation for failed transfer to account %v", toAccount.AccountCode)); compErr != nil {

				return fmt.Errorf("compensation failed: %v, original error: %w", compErr, err)
			}

			// Помечаем операцию как неудачную
			if markErr := r.markOperationFailed(tx, operationID, err.Error()); markErr != nil {
				return fmt.Errorf("mark operation failed error: %v, original error: %w", markErr, err)
			}
			return fmt.Errorf("deposit failed: %w", err)
		}

		// 5. Помечаем операцию как успешную
		_, err = tx.ExecContext(ctx,
			`UPDATE balance_operations 
             SET status = 'COMPLETED', processed_at = NOW()
             WHERE operation_id = $1`,
			operationID)
		if err != nil {
			return fmt.Errorf("mark operation completed failed: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Инвалидируем кэш для обоих счетов
	r.invalidateAccountCache(ctx, fromAccount.AccountCode)
	r.invalidateAccountCache(ctx, toAccount.AccountCode)
	r.invalidateUserAccountsCache(ctx, fromAccount.UserID.String())
	r.invalidateUserAccountsCache(ctx, toAccount.UserID.String())

	return nil
}

// Deposit добавляет средства на счет с записью в историю
func (r *PaymentRepository) Deposit(
	ctx context.Context,
	operationID uuid.UUID,
	accountCode string,
	amount float64,
	paymentID uuid.UUID,
	description string,
) error {
	const op = "repository.PaymentRepository.Deposit"

	if amount <= 0 {
		return fmt.Errorf("%s: %w", op, billingerr.ErrInvalidAmount)
	}
	var userID uuid.UUID
	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
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

		// Получаем информацию о счете
		var currency string
		err = tx.QueryRowContext(ctx,
			`SELECT user_id, currency FROM currency_accounts 
     WHERE account_code = $1 FOR UPDATE`,
			accountCode).Scan(&userID, &currency)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return billingerr.ErrAccountNotFound
			}
			return err
		}
		// Обновляем баланс
		var newBalance float64
		err = tx.GetContext(ctx, &newBalance,
			`UPDATE currency_accounts 
             SET balance = balance + $1, updated_at = NOW()
             WHERE account_code = $2
             RETURNING balance`,
			amount, accountCode)
		if err != nil {
			return err
		}

		// Записываем операцию в историю
		_, err = tx.ExecContext(ctx,
			`INSERT INTO balance_operations (
                operation_id, account_code, user_id, currency, amount, 
                new_balance, operation_type, status, 
                payment_id, description, processed_at
             ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())`,
			operationID, accountCode, userID, currency, amount, newBalance,
			"DEPOSIT", "COMPLETED", paymentID, description)
		return err
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Инвалидируем кэш баланса
	r.invalidateAccountCache(ctx, accountCode)
	r.invalidateUserAccountsCache(ctx, userID.String())
	return nil
}

// Withdraw списывает средства со счета
func (r *PaymentRepository) Withdraw(
	ctx context.Context,
	operationID uuid.UUID,
	accountCode string,
	amount float64,
	paymentID uuid.UUID,
	description string,
) error {
	const op = "repository.PaymentRepository.Withdraw"

	if amount <= 0 {
		return fmt.Errorf("%s: %w", op, billingerr.ErrInvalidAmount)
	}
	var userID uuid.UUID
	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		// 1. Проверяем, не выполнялась ли уже эта операция
		var existingOpID uuid.UUID
		err := tx.GetContext(ctx, &existingOpID,
			`SELECT operation_id FROM balance_operations 
             WHERE operation_id = $1 FOR UPDATE`,
			operationID)
		if err == nil {
			return nil // Операция уже выполнена
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check existing operation failed: %w", err)
		}

		// 2. Получаем информацию о счете
		var currency string
		var balance float64
		err = tx.QueryRowContext(ctx,
			`SELECT user_id, currency, balance 
     FROM currency_accounts 
     WHERE account_code = $1 FOR UPDATE`,
			accountCode,
		).Scan(&userID, &currency, &balance)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return billingerr.ErrAccountNotFound
			}
			return fmt.Errorf("get account failed: %w", err)
		}

		// 3. Проверяем достаточность средств
		if balance < amount {
			return billingerr.ErrInsufficientFunds
		}

		// 4. Списание средств
		var newBalance float64
		err = tx.GetContext(ctx, &newBalance,
			`UPDATE currency_accounts 
             SET balance = balance - $1, updated_at = NOW()
             WHERE account_code = $2
             RETURNING balance`,
			amount, accountCode)
		if err != nil {
			return fmt.Errorf("update balance failed: %w", err)
		}

		// 5. Записываем операцию
		_, err = tx.ExecContext(ctx,
			`INSERT INTO balance_operations (
                operation_id, account_code, user_id, currency, amount, 
                new_balance, operation_type, status, 
                payment_id, description, processed_at
             ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())`,
			operationID,
			accountCode,
			userID,
			currency,
			amount,
			newBalance,
			"WITHDRAWAL",
			"COMPLETED",
			paymentID,
			description)
		if err != nil {
			return fmt.Errorf("insert operation failed: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Инвалидируем кэш
	r.invalidateAccountCache(ctx, accountCode)
	r.invalidateUserAccountsCache(ctx, userID.String())

	return nil
}

// Вспомогательные методы для работы внутри транзакции
func (r *PaymentRepository) withdrawInTx(tx *sqlx.Tx,
	operationID uuid.UUID,
	accountCode string,
	amount float64,
	paymentID uuid.UUID,
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
            payment_id, description, processed_at
         ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())`,
		operationID, accountCode, userID, currency, amount, newBalance,
		"WITHDRAWAL", "COMPLETED", paymentID, description)
	return err
}

func (r *PaymentRepository) depositInTx(tx *sqlx.Tx,
	operationID uuid.UUID,
	accountCode string,
	amount float64,
	paymentID uuid.UUID,
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
            payment_id, description, processed_at
         ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())`,
		operationID, accountCode, userID, currency, amount, newBalance,
		"DEPOSIT", "COMPLETED", paymentID, description)
	return err
}

// GetByUser возвращает все payment_id для user_id из таблицы balance_operations
func (r *PaymentRepository) GetPaymentsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	const op = "repository.PaymentRepository.GetPaymentsByUser"

	stmt, err := r.db.Prepare(`
        SELECT DISTINCT payment_id
        FROM balance_operations 
        WHERE user_id = $1 AND payment_id IS NOT NULL
    `)
	if err != nil {
		return nil, fmt.Errorf("%s: prepare query failed: %w", op, err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, billingerr.ErrPaymentNotFound)
		}
		return nil, fmt.Errorf("%s: execute query failed: %w", op, err)
	}
	defer rows.Close()

	var payments []uuid.UUID
	for rows.Next() {
		var paymentID uuid.UUID
		if err := rows.Scan(&paymentID); err != nil {
			return nil, fmt.Errorf("%s: scan row failed: %w", op, err)
		}
		payments = append(payments, paymentID)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows iteration failed: %w", op, err)
	}

	return payments, nil
}

// UpdateCurrencyRate обновляет курс валюты
func (r *PaymentRepository) UpdateCurrencyRate(
	ctx context.Context,
	fromCurrency, toCurrency string,
	rate float64,
) error {
	const op = "repository.PaymentRepository.UpdateCurrencyRate"

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
func (r *PaymentRepository) ConvertCurrency(
	ctx context.Context,
	operationID uuid.UUID,
	fromAccount, toAccount *models.CurrencyAccount,
	amount float64,
	description string,
) error {
	const op = "repository.PaymentRepository.ConvertCurrency"

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
func (r *PaymentRepository) GetPendingOperations(
	ctx context.Context,
	batchSize int,
) ([]models.BalanceOperation, error) {
	const op = "repository.PaymentRepository.GetPendingOperations"

	var operations []models.BalanceOperation

	err := r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		return tx.SelectContext(ctx, &operations,
			`SELECT id, operation_id, account_code, user_id, currency, amount, 
					new_balance, operation_type, status, payment_id, 
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

// GetOperationHistory возвращает историю операций по счету
func (r *PaymentRepository) GetOperationHistory(
	ctx context.Context,
	accountCode string,
	limit, offset int,
) ([]models.BalanceOperation, error) {
	const op = "repository.PaymentRepository.GetOperationHistory"

	// Валидация параметров пагинации
	if limit < 1 || limit > 100 {
		limit = 50 // Значение по умолчанию
	}
	if offset < 0 {
		offset = 0
	}

	cacheKey := fmt.Sprintf("operations:%v:%v:%v", accountCode, offset, limit)
	var operations []models.BalanceOperation

	// Проверяем кэш
	cachedData, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		if err := json.Unmarshal([]byte(cachedData), &operations); err == nil {
			return operations, nil
		}
	}

	err = r.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		// 1. Проверяем существование счета
		var accountExists bool
		err := tx.GetContext(ctx, &accountExists,
			`SELECT EXISTS(SELECT 1 FROM currency_accounts WHERE account_code = $1)`,
			accountCode)
		if err != nil {
			return fmt.Errorf("check account exists failed: %w", err)
		}
		if !accountExists {
			return billingerr.ErrAccountNotFound
		}

		// 2. Получаем историю операций
		err = tx.SelectContext(ctx, &operations,
			`SELECT 
                id, operation_id, account_code, currency, amount,
                new_balance, operation_type, status,
                payment_id, description, created_at,
                processed_at
             FROM balance_operations
             WHERE account_code = $1
             ORDER BY created_at DESC
             LIMIT $2 OFFSET $3`,
			accountCode, limit, offset)
		if err != nil {
			return fmt.Errorf("select operations failed: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Сохраняем в кэш
	if data, err := json.Marshal(operations); err == nil {
		_ = r.cache.Set(ctx, cacheKey, string(data))
	}

	return operations, nil
}

func (r *PaymentRepository) markOperationFailed(tx *sqlx.Tx, operationID uuid.UUID, errorMsg string) error {
	_, err := tx.ExecContext(context.Background(),
		`UPDATE balance_operations 
         SET status = 'FAILED', description = description || '; ' || $2
         WHERE operation_id = $1`,
		operationID, errorMsg)
	return err
}

func (r *PaymentRepository) invalidateAccountCache(ctx context.Context, accountCode string) {
	cacheKeys := []string{
		fmt.Sprintf("account:%v", accountCode),
		fmt.Sprintf("account_balance:%v", accountCode),
	}
	for _, key := range cacheKeys {
		_ = r.cache.Delete(ctx, key)
	}
	_ = r.cache.DeleteByPrefix(ctx, fmt.Sprintf("operations:%v:", accountCode))
}

func (r *PaymentRepository) invalidateUserAccountsCache(ctx context.Context, userID string) {
	cacheKeys := []string{
		fmt.Sprintf("user_accounts:%v", userID),
	}
	for _, key := range cacheKeys {
		_ = r.cache.Delete(ctx, key)
	}
}

// GetCurrencyRates возвращает курс обмена между валютами
func (r *PaymentRepository) GetCurrencyRate(ctx context.Context, fromCurrency, toCurrency string) (float64, error) {
	const op = "repository.PaymentRepository.GetCurrencyRate"

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
