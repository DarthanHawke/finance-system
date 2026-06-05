-- migrations/1_init_transactions.up.sql
-- =======================================================================
-- Создание таблицы transactions для хранения информации о транзакциях в системе
-- =======================================================================

-- Типы транзакций
CREATE TYPE transaction_type AS ENUM (
    'ON_US_TRANSFER',            -- Перевод между счетами внутри системы
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

-- Статусы транзакций
CREATE TYPE transaction_status AS ENUM (
    'PROCESSING',                -- Активная сага, выполняются шаги
    'AWAITING_EXTERNAL',         -- Ожидаем ответа/средств от внешней системы
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
    parent_transaction_id       UUID REFERENCES transactions(id),

    -- Тип транзакции: transaction_type
    type                        transaction_type NOT NULL,
    -- Статус транзакции: transaction_status
    status                      transaction_status NOT NULL DEFAULT 'PROCESSING',

    -- Сумма транзакции
    amount                      DECIMAL(12, 2) NOT NULL CHECK (amount > 0),
    -- Валюта транзакции в формате ISO 4217
    currency                    VARCHAR(3) NOT NULL,

    -- Инициатор
    initiator                   initiator_type NOT NULL,

    -- Тип финансовой операции и счета
    processing_code             VARCHAR(6),
    -- STAN номер для внешних платежей
    stan                        VARCHAR(6),
    -- Код подтверждения успешной авторизации операции
    authorization_code          VARCHAR(10),
    -- Идентификатор от внешней системы (UETR, RRN и т.д.)
    external_reference          VARCHAR(64),

    -- Описание платежа
    description                 TEXT,

    -- Дата создания записи о транзакции
    created_at                  TIMESTAMP NOT NULL DEFAULT NOW(),
    -- Дата последнего обновления
    updated_at                  TIMESTAMP NOT NULL DEFAULT NOW(),
); 
