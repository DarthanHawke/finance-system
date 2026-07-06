-- migrations/4_init_events.up.sql
-- =======================================================================
-- Создание таблицы events для хранения событий
-- =======================================================================

-- Статусы событий
CREATE TYPE event_status AS ENUM (
    'pending',                   -- Записано в БД, не отправлено
    'published'                  -- Успешно отправлено
);

-- Таблица для хранения событий
CREATE TABLE events (
    -- ID события
    id              UUID PRIMARY KEY,

    -- ID транзакции
    transaction_id  UUID NOT NULL REFERENCES transactions(id),

    -- Шаг саги
    step_name       VARCHAR(50) NOT NULL,
   
    -- Ключ для партиционирования
    partition_key   VARCHAR(64) NOT NULL,

    -- Тип события
    event_type            VARCHAR(50) NOT NULL,

    -- Статус события
    event_status          event_status NOT NULL,
    
    -- Сервис в котором создано событие
    source          VARCHAR(50) NOT NULL,

    -- Контекст трассировки
    trace_id        VARCHAR(32) NOT NULL,
    span_id         VARCHAR(16) NOT NULL,

    -- Время создания события
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Время начала работы с событием
    processed_at    TIMESTAMP WITH TIME ZONE,

    -- Данные события
    payload         JSONB NOT NULL,

    -- Для транзакции и шага выполнится не более 1 одинакового события
    UNIQUE(transaction_id, event_type, step_name)
);

-- Для быстрого поиска pending событий
CREATE INDEX idx_events_pending ON events(event_status, processed_at, created_at)
    WHERE status = 'pending';
