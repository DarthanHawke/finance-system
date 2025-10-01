-- Таблица курсов валют
CREATE TABLE currency_rates (
    from_currency  VARCHAR(3) NOT NULL,
    to_currency    VARCHAR(3) NOT NULL,
    rate           DECIMAL(19, 6) NOT NULL,
    updated_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (from_currency, to_currency)
);

INSERT INTO currency_rates (from_currency, to_currency, rate) VALUES
-- RUB -> *
('RUB', 'USD', 0.011),    -- 1 RUB = 0.011 USD
('RUB', 'EUR', 0.010),    -- 1 RUB = 0.010 EUR
-- USD -> *
('USD', 'RUB', 90.0),     -- 1 USD = 90.0 RUB
('USD', 'EUR', 0.92),     -- 1 USD = 0.92 EUR
-- EUR -> *
('EUR', 'RUB', 100.0),    -- 1 EUR = 100.0 RUB
('EUR', 'USD', 1.09)      -- 1 EUR = 1.09 USD
ON CONFLICT (from_currency, to_currency) 
DO UPDATE SET rate = EXCLUDED.rate, updated_at = NOW();

INSERT INTO currency_rates (from_currency, to_currency, rate) VALUES
('RUB', 'RUB', 1.0),
('USD', 'USD', 1.0),
('EUR', 'EUR', 1.0)
ON CONFLICT (from_currency, to_currency) 
DO UPDATE SET rate = EXCLUDED.rate, updated_at = NOW();