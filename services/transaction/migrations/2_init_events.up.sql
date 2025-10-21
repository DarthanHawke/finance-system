-- =======================================================================
-- Создание таблицы events для хранения событий
-- =======================================================================

-- Таблица для хранения событий
CREATE TABLE events (
    -- ID события
    id              UUID PRIMARY KEY,

    -- ID платежа
    transaction_id      UUID NOT NULL REFERENCES transactions(id),

    -- Тип события
    type            VARCHAR(50) NOT NULL,

    -- Статус события
    status          VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'completed')),
    
    -- Сервис в котором создано событие
    source          VARCHAR(50) NOT NULL,

    -- Время создания события
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Время начала работы с событием
    processed_at    TIMESTAMP WITH TIME ZONE,

    -- Данные события
    payload         JSONB NOT NULL,

    -- Для платежа выполнится не более 1 одинакового события
    UNIQUE(transaction_id, type)
);
