-- 策略冻结风控表
CREATE TABLE IF NOT EXISTS strategy_freeze (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol VARCHAR(50) NOT NULL,
    strategy_name VARCHAR(100) NOT NULL,
    trade_type VARCHAR(10) NOT NULL,
    freeze_until INTEGER NOT NULL DEFAULT 0,
    loss_count INTEGER NOT NULL DEFAULT 0,
    freeze_on_loss_count INTEGER NOT NULL DEFAULT 5,
    freeze_hours INTEGER NOT NULL DEFAULT 24,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- 创建唯一索引确保每个symbol-strategy-trade_type组合唯一
CREATE UNIQUE INDEX IF NOT EXISTS idx_strategy_freeze_unique ON strategy_freeze(symbol, strategy_name, trade_type);