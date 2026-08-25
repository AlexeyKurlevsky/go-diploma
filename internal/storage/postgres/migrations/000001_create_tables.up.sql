CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login         VARCHAR(255) UNIQUE NOT NULL,
    password_hash BYTEA NOT NULL,    -- bcrypt хеш
    created_at    TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    number      VARCHAR(50) UNIQUE NOT NULL,  -- номер заказа (проверяется алгоритмом Луна)
    status      VARCHAR(20) NOT NULL,         -- NEW, REGISTERED, PROCESSING, INVALID, PROCESSED
    accrual     DECIMAL(10,2) DEFAULT NULL,   -- начисленные баллы (может быть NULL)
    uploaded_at TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

CREATE TABLE IF NOT EXISTS withdrawals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_number VARCHAR(50) NOT NULL,        -- номер заказа, за который списываются баллы
    amount       DECIMAL(10,2) NOT NULL,      -- списанная сумма (положительная)
    processed_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals(user_id);

-- Материализованное представление, вычисляющее текущий баланс каждого пользователя
CREATE MATERIALIZED VIEW IF NOT EXISTS user_balance AS
SELECT
    u.id AS user_id,
    COALESCE(SUM(o.accrual), 0) - COALESCE(SUM(w.amount), 0) AS balance,
    COALESCE(SUM(o.accrual), 0) AS total_accrued,
    COALESCE(SUM(w.amount), 0) AS total_withdrawn
FROM users u
LEFT JOIN orders o ON u.id = o.user_id AND o.status = 'PROCESSED'  -- только подтверждённые заказы
LEFT JOIN withdrawals w ON u.id = w.user_id
GROUP BY u.id;
CREATE UNIQUE INDEX idx_user_balance_user_id ON user_balance (user_id);