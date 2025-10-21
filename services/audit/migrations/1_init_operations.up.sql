-- =============================================================================
-- Миграция: Создание таблицы operations - Журнал операций и транзакций системы
-- =============================================================================

-- Таблица для хранения данных операций
CREATE TABLE operations (
    -- ID записи в журнале
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Идентификатор операции (может объединять несколько записей)
    operation_id UUID NOT NULL,
    
    -- Тип операции: transaction, transfer, conversion, deposit, withdrawal
    operation_type VARCHAR(50) NOT NULL,
    
    -- Сервис-источник операции
    service_source VARCHAR(50) NOT NULL,
    
    -- Код связанного счёта
    account_code VARCHAR(34),
    
    -- Идентификатор пользователя, связанного с операцией
    user_id UUID,
    
    -- Ссылка на платеж
    transaction_id UUID,
    
    -- Исходная валюта
    from_currency VARCHAR(3),
    
    -- Целевая валюта
    to_currency VARCHAR(3),
    
    -- Сумма операции
    amount DECIMAL(15, 2),
    
    -- Новый баланс после операции
    new_balance DECIMAL(15, 2),
    
    -- Курс конвертации
    rate DECIMAL(15, 6),
    
    -- Описание операции
    description TEXT,
    
    -- Статус операции
    status VARCHAR(20) NOT NULL,
    
    -- Дополнительные метаданные операции
    metadata JSONB,
    
    -- Дата создания записи об операции
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

COMMENT ON TABLE operations IS 'Журнал всех операций системы';
COMMENT ON COLUMN operations.id IS 'Уникальный идентификатор записи';
COMMENT ON COLUMN operations.operation_id IS 'Идентификатор бизнес-операции';
COMMENT ON COLUMN operations.operation_type IS 'Тип операции';
COMMENT ON COLUMN operations.service_source IS 'Сервис-источник операции';
COMMENT ON COLUMN operations.account_code IS 'Код счёта, к которому относится операция';
COMMENT ON COLUMN operations.user_id IS 'Идентификатор пользователя, связанного с операцией';
COMMENT ON COLUMN operations.transaction_id IS 'Ссылка на связанный платёж';