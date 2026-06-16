-- migrations/3_init_saga_steps.up.sql
-- =======================================================================
-- Создание таблицы saga_steps для хранения информации о саге
-- =======================================================================

-- Статусы шагов саги
CREATE TYPE step_status AS ENUM (
    'STARTED',
    'COMPLETED',
    'FAILED'
);

-- Вид шага
CREATE TYPE step_kind AS ENUM (
    'ACTION',
    'COMPENSATION'
);

-- Шаги саги
CREATE TABLE saga_steps (
    -- ID шага
    id              UUID PRIMARY KEY,
    -- ID транзакции
    transaction_id  UUID NOT NULL REFERENCES transactions(id),
    -- ID события
    event_id        UUID NOT NULL REFERENCES events(id),

    -- Шаг
    step_name       VARCHAR(50) NOT NULL,
    --Тип шага
    step_kind       step_kind NOT NULL,
    -- Статус
    status          step_status NOT NULL,

    -- Дата создания записи о шаге саги
    created_at                  TIMESTAMP NOT NULL DEFAULT NOW(),
    -- Дата последнего обновления
    updated_at                  TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(transaction_id, step_name, step_kind)
);
