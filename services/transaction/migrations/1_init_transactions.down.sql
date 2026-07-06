-- migrations/1_init_transactions.down.sql
-- ===================================================
-- Down-миграция: Удаление таблицы transactions
-- ===================================================

-- Удаление таблицы transactions
DROP TABLE IF EXISTS transactions;
-- Удаление типов ENUM
DROP TYPE IF EXISTS initiator_type;
DROP TYPE IF EXISTS transaction_status;
DROP TYPE IF EXISTS saga_type;
DROP TYPE IF EXISTS transaction_type;
