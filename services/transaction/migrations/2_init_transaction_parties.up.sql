-- migrations/2_init_transaction_parties.up.sql
-- =======================================================================
-- Создание таблицы transaction_parties для хранения информации об участниках транзакции
-- =======================================================================

-- Роли участников
CREATE TYPE party_role AS ENUM ('SENDER', 'RECIPIENT');

-- Типы участников
CREATE TYPE party_type AS ENUM ('INTERNAL', 'EXTERNAL');

-- Участники транзакции
CREATE TABLE transaction_parties (
    --  ID транзакции
    transaction_id              UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,

    -- Роли участников: party_role
    role                        party_role NOT NULL,

    -- Типы участников: party_type
    party_type                  party_type NOT NULL,

    -- Реквизиты как key-value
    -- Для INTERNAL:
    -- {"ACCOUNT_CODE": "40817810500000000001", "PHONE": "+79999999999"}

    -- Для EXTERNAL (международный перевод):
    -- {"IBAN": "DE89370400440532013000", "SWIFT_BIC": "COBADEFFXXX", "NAME": "Max Mustermann"}

    -- Для EXTERNAL (российский СБП):
    -- {"PHONE": "+79999999999", "BANK_CODE": "044444444", "BANK_CODE_TYPE": "LOCAL_BIC", "NAME": "Иван И."}

    -- Для EXTERNAL (по номеру карты):
    -- {"CARD_NUMBER": "411111******1111", "NAME": "IVAN IVANOV"}
    identifiers   JSONB NOT NULL DEFAULT '{}',

    PRIMARY KEY (transaction_id, role)
);

-- Поиск рефандов по исходной транзакции для валидации при создании рефанда
CREATE INDEX idx_transactions_parent ON transactions(parent_transaction_id)
    WHERE parent_transaction_id IS NOT NULL;
