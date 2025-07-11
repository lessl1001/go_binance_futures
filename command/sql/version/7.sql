-- AI驱动量化策略参数优化相关表结构
-- 版本 7: 添加回测任务、回测结果、已部署策略、操作日志表

-- 回测任务表
CREATE TABLE IF NOT EXISTS backtest_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    strategy TEXT NOT NULL,
    parameters TEXT NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    start_time INTEGER NOT NULL,
    end_time INTEGER NOT NULL,
    created_by VARCHAR(50) NOT NULL,
    progress REAL NOT NULL DEFAULT 0,
    error_message TEXT,
    create_time INTEGER NOT NULL,
    update_time INTEGER NOT NULL
);

-- 回测结果表
CREATE TABLE IF NOT EXISTS backtest_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id VARCHAR(50) NOT NULL,
    result_id VARCHAR(50) NOT NULL UNIQUE,
    parameters TEXT NOT NULL,
    total_return REAL NOT NULL DEFAULT 0,
    annual_return REAL NOT NULL DEFAULT 0,
    max_drawdown REAL NOT NULL DEFAULT 0,
    sharpe_ratio REAL NOT NULL DEFAULT 0,
    win_rate REAL NOT NULL DEFAULT 0,
    trade_count INTEGER NOT NULL DEFAULT 0,
    profit_factor REAL NOT NULL DEFAULT 0,
    equity_curve TEXT,
    trade_list TEXT,
    create_time INTEGER NOT NULL
);

-- 已部署策略表
CREATE TABLE IF NOT EXISTS deployed_strategies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    strategy_id VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    parameters TEXT NOT NULL,
    strategy TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    deployed_by VARCHAR(50) NOT NULL,
    deploy_time INTEGER NOT NULL,
    backtest_task_id VARCHAR(50),
    backtest_result_id VARCHAR(50),
    live_return REAL NOT NULL DEFAULT 0,
    live_trade_count INTEGER NOT NULL DEFAULT 0,
    last_update_time INTEGER NOT NULL,
    create_time INTEGER NOT NULL,
    update_time INTEGER NOT NULL
);

-- 操作日志表
CREATE TABLE IF NOT EXISTS operation_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id VARCHAR(50) NOT NULL,
    operation VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(50) NOT NULL,
    details TEXT,
    ip_address VARCHAR(45),
    user_agent TEXT,
    create_time INTEGER NOT NULL
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_backtest_tasks_task_id ON backtest_tasks(task_id);
CREATE INDEX IF NOT EXISTS idx_backtest_tasks_status ON backtest_tasks(status);
CREATE INDEX IF NOT EXISTS idx_backtest_tasks_symbol ON backtest_tasks(symbol);
CREATE INDEX IF NOT EXISTS idx_backtest_tasks_create_time ON backtest_tasks(create_time);

CREATE INDEX IF NOT EXISTS idx_backtest_results_task_id ON backtest_results(task_id);
CREATE INDEX IF NOT EXISTS idx_backtest_results_result_id ON backtest_results(result_id);
CREATE INDEX IF NOT EXISTS idx_backtest_results_sharpe_ratio ON backtest_results(sharpe_ratio);

CREATE INDEX IF NOT EXISTS idx_deployed_strategies_strategy_id ON deployed_strategies(strategy_id);
CREATE INDEX IF NOT EXISTS idx_deployed_strategies_symbol ON deployed_strategies(symbol);
CREATE INDEX IF NOT EXISTS idx_deployed_strategies_status ON deployed_strategies(status);
CREATE INDEX IF NOT EXISTS idx_deployed_strategies_deploy_time ON deployed_strategies(deploy_time);

CREATE INDEX IF NOT EXISTS idx_operation_logs_user_id ON operation_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_operation_logs_operation ON operation_logs(operation);
CREATE INDEX IF NOT EXISTS idx_operation_logs_resource_type ON operation_logs(resource_type);
CREATE INDEX IF NOT EXISTS idx_operation_logs_create_time ON operation_logs(create_time);