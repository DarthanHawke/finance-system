-- migrations/3_init_saga_steps.down.sql
-- ===================================================
-- Down-миграция: Удаление таблицы saga_steps
-- ===================================================

-- Удаление таблицы saga_steps
DROP TABLE IF EXISTS saga_steps;
-- Удаление типов ENUM
DROP TYPE IF EXISTS step_kind;
DROP TYPE IF EXISTS step_status;
