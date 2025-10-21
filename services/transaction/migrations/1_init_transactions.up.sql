-- =======================================================================
-- Создание таблицы transactions для хранения информации о платежах в системе
-- =======================================================================

-- Таблица для хранения данных о платежах между счетами
CREATE TABLE transactions (
    --  ID платежа
    id                          UUID PRIMARY KEY,

    -- Тип платежа: purchase, withdrawal, transfer, payment
    type                VARCHAR(20) NOT NULL,
    -- Сумма платежа
    amount                      DECIMAL(12, 2) NOT NULL CHECK (amount > 0),
    -- Валюта платежа в формате ISO 4217
    currency                    VARCHAR(3) NOT NULL,

    -- Тип платежа: internal, external
    sender_type                 VARCHAR(10) NOT NULL,
    -- Номер счета отправителя
    sender_account_code         VARCHAR(34), 
    -- Номер телефона привязанного к счету отправителя
    sender_phone                VARCHAR(15),
    -- Номер банковской карты отправителя
    sender_card_number          VARCHAR(20),

    -- Тип платежа: internal, external
    recipient_type              VARCHAR(10) NOT NULL,
    -- Номер счета получателя
    recipient_account_code      VARCHAR(34), 
    -- Номер телефона привязанного к счету получателя
    recipient_phone             VARCHAR(15),
    -- Номер банковской карты получателя
    recipient_card_number       VARCHAR(20),

    -- Тип финансовой операции и счета
    processing_code             VARCHAR(6) NOT NULL,
    -- STAN номер для внешних платежей
    stan                        VARCHAR(6) NOT NULL,
    -- Код подтверждения успешной авторизации операции
    authorization_code          VARCHAR(10),

    -- Описание платежа
    description                 TEXT,
    -- Статус платежа: pending, completed, failed, cancelled
    status                      VARCHAR(20) NOT NULL,
    -- Дата создания записи о платеже
    created_at                  TIMESTAMP NOT NULL DEFAULT NOW(),
    -- Дата последнего обновления
    updated_at                  TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Дата платежа
    date                        DATE NOT NULL DEFAULT CURRENT_DATE,
    -- STAN уникален в течении дня (а в моем пет-проекте ещё и для каждого типа платежа)
    UNIQUE(stan, date, type, sender_type)
); 
