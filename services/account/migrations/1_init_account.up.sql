-- =======================================================================
-- Создание таблицы payments для хранения информации о валютных счетах 
-- =======================================================================

-- Таблица для хранения данных о валютных счетах
CREATE TABLE accounts (
    -- Уникальный код счёта для идентификации в платежных операциях
    code            VARCHAR(34) PRIMARY KEY,
    -- Название счёта
    name            VARCHAR(100),
    -- Идентификатор владельца счёта (внешний ключ)
    user_id         UUID NOT NULL,

    -- Валюта счёта в формате ISO 4217
    currency        VARCHAR(3) NOT NULL,
    -- Доступный баланс на счёте
    balance         DECIMAL(15, 2) NOT NULL DEFAULT 0 CHECK (balance >= 0),
    -- Заблокированые средства для снятия
    frozen_balance  DECIMAL(15, 2) NOT NULL DEFAULT 0 CHECK (frozen_balance >= 0),
    -- Заблокированые средства для пополнения
    reserve_balance DECIMAL(15, 2) NOT NULL DEFAULT 0 CHECK (reserve_balance >= 0),

    -- статус счета: active, blocked, closed
    status           VARCHAR(20) DEFAULT 'active'
    -- Дата создания счёта
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    -- Дата последнего обновления информации о счёте
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
);
