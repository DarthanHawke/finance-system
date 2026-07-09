-- migrations/1_init_transactions.up.sql
-- =======================================================================
-- Создание таблицы transactions для хранения информации о транзакциях в системе
-- =======================================================================

-- Типы транзакций
CREATE TYPE transaction_type AS ENUM (
    'ON_US_TRANSFER',            -- Перевод между счетами внутри системы
    'ON_US_FX_TRANSFER',         -- Перевод между счетами внутри системы с валютным обменом
    'INBOUND_REMITTANCE',        -- Входящий перевод из внешней системы
    'OUTBOUND_REMITTANCE',       -- Исходящий перевод во внешнюю систему
    'DIRECT_DEBIT',              -- Списание по требованию получателя внутри системы
    'INBOUND_DIRECT_DEBIT',      -- Входящий перевод по мандату из внешней системы
    'OUTBOUND_DIRECT_DEBIT',     -- Исходящий перевод по мандату во внешнюю систему
    'TOPUP_REQUEST',             -- Запрос на входящий перевод из внешнего источника(me-to-me)
    'REFUND',                    -- Возврат
    'CHARGEBACK',                -- Принудительный возврат, инициированный внешней системой
    'ADJUSTMENT'                 -- Административная операция
);

-- Типы саг
CREATE TYPE saga_type AS ENUM (
    'ON_US_TRANSFER',
    'ON_US_FX_TRANSFER',
    'OUTBOUND_REMITTANCE',
    'INBOUND_REMITTANCE',
    'TOPUP_REQUEST',
    'REFUND_INTERNAL',
    'REFUND_OUTBOUND',
    'REFUND_INBOUND',
    'CHARGEBACK_INTERNAL',
    'CHARGEBACK_OUTBOUND',
    'CHARGEBACK_INBOUND',
    'ADJUSTMENT_DEBIT',
    'ADJUSTMENT_CREDIT'
);

-- Статусы транзакций
CREATE TYPE transaction_status AS ENUM (
    'PROCESSING',                -- Активная сага, выполняются шаги
    'COMPLETED',                 -- Успешно завершена
    'CANCELLED',                 -- Отменена
    'BLOCKED'                    -- Счета заблокированы, требуется ручное вмешательство
);

-- Типы инициаторов
CREATE TYPE initiator_type AS ENUM (
    'CUSTOMER',                  -- Клиент через API
    'EXTERNAL',                  -- Внешняя система
    'SYSTEM'                     -- Внутренний процесс (администратор, автоплатёж)
);

-- Таблица для хранения данных о транзакциях между счетами
CREATE TABLE transactions (
    --  ID транзакции
    id                          UUID PRIMARY KEY,
    -- Идемпотентность(для ON_US_TRANSFER, при работе с внешними сервисами используем external_reference)
    idempotency_key             VARCHAR(64) UNIQUE,
    -- Исходная операция(при возврате)
    parent_transaction_id       UUID,
    -- ID котриовки на обмен валюты
    fx_deal_id                  UUID,

    -- Тип транзакции: transaction_type
    transaction_type            transaction_type NOT NULL,
    -- Тип саги 
    saga_type                   saga_type NOT NULL,
    -- Текущее состояние в state machine
    saga_state                  TEXT NOT NULL,
    -- Статус транзакции: transaction_status
    transaction_status          transaction_status NOT NULL,

    -- Сумма транзакции
    amount                      DECIMAL(12, 2) NOT NULL CHECK (amount > 0),
    -- Исходная валюта транзакции
    source_currency             VARCHAR(3) NOT NULL,
    -- Целевая валюта транзакции
    target_currency             VARCHAR(3) NOT NULL,
    -- Сумма комиссии
    fee_amount                  DECIMAL(12, 2) NOT NULL DEFAULT 0,
    -- Валюта комиссии
    fee_currency                VARCHAR(3) NOT NULL,

    -- Описание платежа
    transaction_description     TEXT,

    -- Дата создания записи о транзакции
    created_at                  TIMESTAMP NOT NULL,
    -- Дата последнего обновления
    updated_at                  TIMESTAMP NOT NULL,
); 

-- Поиск рефандов по исходной транзакции для валидации при создании рефанда
CREATE INDEX idx_transactions_parent ON transactions(parent_transaction_id)
    WHERE parent_transaction_id IS NOT NULL;
