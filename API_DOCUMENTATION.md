# AI驱动量化策略参数优化API文档

## 概述

本文档描述了AI驱动量化策略参数优化相关的RESTful API接口，包括回测API和策略部署API。

## 认证

所有API请求都需要通过JWT认证。在请求头中包含：
```
Authorization: Bearer <your-jwt-token>
```

## 回测API

### 创建回测任务

**POST** `/api/backtest`

创建新的回测任务，支持自定义策略表达式、参数组、回测区间、币种等。

#### 请求体

```json
{
  "name": "策略回测任务名称",
  "strategy": "close > ma5 && rsi < 30",
  "parameters": [
    {"ma5": 5, "rsi_period": 14},
    {"ma5": 10, "rsi_period": 21}
  ],
  "symbol": "BTCUSDT",
  "start_time": 1640995200,
  "end_time": 1672531200,
  "concurrent": 5
}
```

#### 参数说明

- `name`: 任务名称，必填
- `strategy`: 策略表达式，必填，支持数学和逻辑运算
- `parameters`: 参数组合数组，必填，每个组合是一个参数对象
- `symbol`: 测试币种，必填，如"BTCUSDT"
- `start_time`: 回测开始时间，Unix时间戳
- `end_time`: 回测结束时间，Unix时间戳
- `concurrent`: 并发数量，可选，默认5，最大20

#### 策略表达式说明

策略表达式支持以下变量：
- `close`: 收盘价
- `open`: 开盘价
- `high`: 最高价
- `low`: 最低价
- `ma5`, `ma10`, `ma20`: 移动平均线
- `rsi`: RSI指标
- `volume`: 成交量

支持的运算符：
- 比较运算符: `>`, `<`, `>=`, `<=`, `==`, `!=`
- 逻辑运算符: `&&`, `||`, `!`
- 数学运算符: `+`, `-`, `*`, `/`

#### 响应

```json
{
  "task_id": "bt_1640995200_123456",
  "status": "success",
  "message": "Backtest task created successfully"
}
```

### 获取回测任务列表

**GET** `/api/backtest`

获取回测任务列表，支持分页和过滤。

#### 查询参数

- `page`: 页码，可选，默认1
- `pageSize`: 每页数量，可选，默认20
- `status`: 状态过滤，可选，值为：pending, running, completed, failed

#### 响应

```json
{
  "status": "success",
  "data": {
    "tasks": [
      {
        "id": 1,
        "task_id": "bt_1640995200_123456",
        "name": "策略回测任务名称",
        "status": "completed",
        "strategy": "close > ma5 && rsi < 30",
        "symbol": "BTCUSDT",
        "start_time": 1640995200,
        "end_time": 1672531200,
        "created_by": "admin",
        "progress": 100,
        "create_time": 1640995200,
        "update_time": 1640995300
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20
  }
}
```

### 获取回测结果

**GET** `/api/backtest/{taskId}/results`

获取指定任务的回测结果。

#### 路径参数

- `taskId`: 任务ID

#### 响应

```json
{
  "task_id": "bt_1640995200_123456",
  "status": "completed",
  "results": [
    {
      "id": 1,
      "task_id": "bt_1640995200_123456",
      "result_id": "bt_1640995200_123456_0",
      "parameters": "{\"ma5\": 5, \"rsi_period\": 14}",
      "total_return": 0.15,
      "annual_return": 0.25,
      "max_drawdown": 0.05,
      "sharpe_ratio": 3.0,
      "win_rate": 0.65,
      "trade_count": 100,
      "profit_factor": 1.5,
      "equity_curve": "[{\"time\": 1640995200, \"equity\": 10000}]",
      "trade_list": "[{\"time\": 1640995200, \"side\": \"buy\", \"price\": 100, \"pnl\": 50}]",
      "create_time": 1640995200
    }
  ],
  "best_result": {
    "id": 1,
    "result_id": "bt_1640995200_123456_0",
    "sharpe_ratio": 3.0,
    "total_return": 0.15
  }
}
```

#### 回测指标说明

- `total_return`: 总收益率
- `annual_return`: 年化收益率
- `max_drawdown`: 最大回撤
- `sharpe_ratio`: 夏普比率
- `win_rate`: 胜率
- `trade_count`: 交易次数
- `profit_factor`: 利润因子
- `equity_curve`: 权益曲线数据（JSON数组）
- `trade_list`: 交易记录（JSON数组）

### 删除回测任务

**DELETE** `/api/backtest/{taskId}`

删除指定的回测任务及其结果。

#### 路径参数

- `taskId`: 任务ID

#### 响应

```json
{
  "status": "success",
  "message": "Task deleted successfully"
}
```

## 策略部署API

### 部署策略

**POST** `/api/deploy_strategy`

部署最优参数/策略到实盘配置，包含权限校验与操作日志。

#### 请求体

```json
{
  "name": "最优策略",
  "symbol": "ETHUSDT",
  "parameters": {
    "ma5": 5,
    "rsi_period": 14
  },
  "strategy": "close > ma5 && rsi < 30",
  "backtest_task_id": "bt_1640995200_123456",
  "backtest_result_id": "bt_1640995200_123456_0",
  "force": false
}
```

#### 参数说明

- `name`: 策略名称，必填
- `symbol`: 应用币种，必填
- `parameters`: 最优参数，必填，JSON对象
- `strategy`: 策略表达式，必填
- `backtest_task_id`: 关联的回测任务ID，可选
- `backtest_result_id`: 关联的回测结果ID，可选
- `force`: 强制部署，可选，覆盖现有策略

#### 响应

```json
{
  "strategy_id": "ds_1640995200_123456",
  "status": "success",
  "message": "Strategy deployed successfully"
}
```

### 获取已部署策略列表

**GET** `/api/deploy_strategy`

获取已部署策略列表，支持分页和过滤。

#### 查询参数

- `page`: 页码，可选，默认1
- `pageSize`: 每页数量，可选，默认20
- `status`: 状态过滤，可选，值为：active, inactive, error
- `symbol`: 币种过滤，可选

#### 响应

```json
{
  "status": "success",
  "data": {
    "strategies": [
      {
        "id": 1,
        "strategy_id": "ds_1640995200_123456",
        "name": "最优策略",
        "symbol": "ETHUSDT",
        "parameters": "{\"ma5\": 5, \"rsi_period\": 14}",
        "strategy": "close > ma5 && rsi < 30",
        "status": "active",
        "deployed_by": "admin",
        "deploy_time": 1640995200,
        "live_return": 0.08,
        "live_trade_count": 50,
        "create_time": 1640995200,
        "update_time": 1640995300
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20
  }
}
```

### 更新策略状态

**PUT** `/api/deploy_strategy/{strategyId}`

更新已部署策略的状态（激活/停用）。

#### 路径参数

- `strategyId`: 策略ID

#### 请求体

```json
{
  "status": "inactive"
}
```

#### 参数说明

- `status`: 新状态，必填，值为：active, inactive

#### 响应

```json
{
  "status": "success",
  "message": "Strategy status updated successfully"
}
```

### 删除策略

**DELETE** `/api/deploy_strategy/{strategyId}`

删除已部署的策略。

#### 路径参数

- `strategyId`: 策略ID

#### 响应

```json
{
  "status": "success",
  "message": "Strategy deleted successfully"
}
```

## 操作日志API

### 获取操作日志

**GET** `/api/operation_logs`

获取操作日志列表，支持分页和过滤。

#### 查询参数

- `page`: 页码，可选，默认1
- `pageSize`: 每页数量，可选，默认20
- `operation`: 操作类型过滤，可选
- `resourceType`: 资源类型过滤，可选

#### 响应

```json
{
  "status": "success",
  "data": {
    "logs": [
      {
        "id": 1,
        "user_id": "admin",
        "operation": "DEPLOY_STRATEGY",
        "resource_type": "deployed_strategy",
        "resource_id": "ds_1640995200_123456",
        "details": "{\"name\": \"最优策略\", \"symbol\": \"ETHUSDT\"}",
        "ip_address": "127.0.0.1",
        "user_agent": "Mozilla/5.0...",
        "create_time": 1640995200
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20
  }
}
```

## 错误处理

所有API都使用统一的错误格式：

```json
{
  "status": "error",
  "message": "错误描述信息"
}
```

常见错误码：
- 400: 请求参数错误
- 401: 未授权
- 403: 权限不足
- 404: 资源不存在
- 429: 请求过于频繁
- 500: 服务器内部错误

## 并发和性能

- 回测任务支持并发执行，默认并发数为5，最大20
- 所有API都有速率限制，防止滥用
- 批量操作使用分页，避免单次请求数据过大

## 安全性

- 所有API都需要JWT认证
- 敏感操作需要额外的权限校验
- 完整的操作日志记录
- IP地址和用户代理记录
- 参数验证和SQL注入防护

## 示例

### 完整回测流程示例

```bash
# 1. 创建回测任务
curl -X POST /api/backtest \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MA+RSI策略回测",
    "strategy": "close > ma5 && rsi < 30",
    "parameters": [
      {"ma5": 5, "rsi_period": 14},
      {"ma5": 10, "rsi_period": 21}
    ],
    "symbol": "BTCUSDT",
    "start_time": 1640995200,
    "end_time": 1672531200,
    "concurrent": 5
  }'

# 2. 查看回测结果
curl -X GET /api/backtest/bt_1640995200_123456/results \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# 3. 部署最优策略
curl -X POST /api/deploy_strategy \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "最优MA+RSI策略",
    "symbol": "BTCUSDT",
    "parameters": {"ma5": 5, "rsi_period": 14},
    "strategy": "close > ma5 && rsi < 30",
    "backtest_task_id": "bt_1640995200_123456",
    "backtest_result_id": "bt_1640995200_123456_0"
  }'
```