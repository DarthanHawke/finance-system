-- migrations/2_init_transaction_parties.down.sql
-- ===================================================
-- Down-миграция: Удаление таблицы transaction_parties
-- ===================================================

-- Удаление таблицы transaction_parties
DROP TABLE IF EXISTS transaction_parties;
-- Удаление типов ENUM
DROP TYPE IF EXISTS party_type;
DROP TYPE IF EXISTS party_role;
