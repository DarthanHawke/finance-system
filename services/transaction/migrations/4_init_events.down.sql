-- migrations/4_init_events.down.sql
-- ===================================================
-- Down-миграция: Удаление таблицы events
-- ===================================================

-- Удаление таблицы events
DROP TABLE IF EXISTS events;
-- Удаление типов ENUM
DROP TYPE IF EXISTS event_status;