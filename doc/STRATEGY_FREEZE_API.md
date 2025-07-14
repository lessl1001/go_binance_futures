# 策略冻结风控功能 API 文档

## 功能概述

本功能实现了币种-策略-交易类型（实盘/测试）的独立风控管理，通过统计策略的盈亏情况，在达到亏损阈值时自动冻结策略执行，以控制风险。

## 主要特性

1. **独立风控**: 每个 `symbol-strategy-trade_type` 组合拥有独立的风控参数和冻结状态
2. **自动冻结**: 当连续亏损次数达到阈值时，自动冻结策略一段时间
3. **分类管理**: 真实交易与测试交易分别统计和冻结，互不影响
4. **持久化**: 风控信息持久化存储，服务重启后不丢失
5. **RESTful API**: 提供完整的 API 接口用于配置和管理

## 数据模型

### StrategyFreeze 表结构

| 字段名 | 类型 | 说明 |
|--------|------|------|
| id | int64 | 主键ID |
| symbol | string | 交易对符号（如 BTCUSDT） |
| strategy_name | string | 策略名称 |
| trade_type | string | 交易类型（real/test） |
| freeze_until | int64 | 冻结截止时间戳 |
| loss_count | int | 当前亏损次数 |
| freeze_on_loss_count | int | 达到此亏损次数后冻结（默认5） |
| freeze_hours | int | 冻结小时数（默认24） |
| created_at | int64 | 创建时间戳 |
| updated_at | int64 | 更新时间戳 |

## API 接口

### 1. 获取冻结配置列表

**GET** `/strategy-freeze`

**查询参数：**
- `page`: 页码（默认1）
- `pageSize`: 每页数量（默认20，最大100）

**响应示例：**
```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
    "list": [
      {
        "id": 1,
        "symbol": "BTCUSDT",
        "strategy_name": "line3_coin6",
        "trade_type": "real",
        "freeze_until": 0,
        "loss_count": 2,
        "freeze_on_loss_count": 5,
        "freeze_hours": 24,
        "created_at": 1672531200,
        "updated_at": 1672531200
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20
  }
}
```

### 2. 创建或更新冻结配置

**POST** `/strategy-freeze`

**请求体：**
```json
{
  "symbol": "BTCUSDT",
  "strategy_name": "line3_coin6",
  "trade_type": "real",
  "freeze_on_loss_count": 5,
  "freeze_hours": 24,
  "loss_count": 0
}
```

**响应示例：**
```json
{
  "code": 200,
  "message": "创建成功",
  "data": {
    "id": 1,
    "symbol": "BTCUSDT",
    "strategy_name": "line3_coin6",
    "trade_type": "real",
    "freeze_until": 0,
    "loss_count": 0,
    "freeze_on_loss_count": 5,
    "freeze_hours": 24,
    "created_at": 1672531200,
    "updated_at": 1672531200
  }
}
```

### 3. 获取单个冻结配置

**GET** `/strategy-freeze/config`

**查询参数：**
- `symbol`: 交易对符号（必填）
- `strategy_name`: 策略名称（必填）
- `trade_type`: 交易类型（必填，real/test）

**响应示例：**
```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
    "config": {
      "id": 1,
      "symbol": "BTCUSDT",
      "strategy_name": "line3_coin6",
      "trade_type": "real",
      "freeze_until": 0,
      "loss_count": 2,
      "freeze_on_loss_count": 5,
      "freeze_hours": 24,
      "created_at": 1672531200,
      "updated_at": 1672531200
    },
    "is_frozen": false,
    "remaining_time": 0
  }
}
```

### 4. 更新冻结配置

**PUT** `/strategy-freeze/:id`

**请求体：**
```json
{
  "freeze_on_loss_count": 3,
  "freeze_hours": 12,
  "loss_count": 1
}
```

**响应示例：**
```json
{
  "code": 200,
  "message": "更新成功",
  "data": {
    "id": 1,
    "symbol": "BTCUSDT",
    "strategy_name": "line3_coin6",
    "trade_type": "real",
    "freeze_until": 0,
    "loss_count": 1,
    "freeze_on_loss_count": 3,
    "freeze_hours": 12,
    "created_at": 1672531200,
    "updated_at": 1672531250
  }
}
```

### 5. 手动解除冻结

**POST** `/strategy-freeze/unfreeze`

**请求体：**
```json
{
  "symbol": "BTCUSDT",
  "strategy_name": "line3_coin6",
  "trade_type": "real"
}
```

**响应示例：**
```json
{
  "code": 200,
  "message": "解除冻结成功"
}
```

### 6. 重置亏损次数

**POST** `/strategy-freeze/reset-loss`

**请求体：**
```json
{
  "symbol": "BTCUSDT",
  "strategy_name": "line3_coin6",
  "trade_type": "real"
}
```

**响应示例：**
```json
{
  "code": 200,
  "message": "重置亏损次数成功"
}
```

## 使用示例

### 1. 前端配置风控参数

```javascript
// 创建或更新风控配置
const config = {
  symbol: "BTCUSDT",
  strategy_name: "line3_coin6", 
  trade_type: "real",
  freeze_on_loss_count: 5,  // 连续亏损5次后冻结
  freeze_hours: 24          // 冻结24小时
};

fetch('/strategy-freeze', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify(config)
})
.then(response => response.json())
.then(data => {
  console.log('配置成功:', data);
});
```

### 2. 查询冻结状态

```javascript
// 查询特定策略的冻结状态
const params = new URLSearchParams({
  symbol: 'BTCUSDT',
  strategy_name: 'line3_coin6',
  trade_type: 'real'
});

fetch(`/strategy-freeze/config?${params}`)
.then(response => response.json())
.then(data => {
  if (data.data.is_frozen) {
    console.log(`策略被冻结，剩余时间: ${data.data.remaining_time} 秒`);
  } else {
    console.log('策略未被冻结');
  }
});
```

### 3. 获取所有冻结配置

```javascript
// 获取第一页冻结配置
fetch('/strategy-freeze?page=1&pageSize=20')
.then(response => response.json())
.then(data => {
  console.log('冻结配置列表:', data.data.list);
  console.log('总计:', data.data.total);
});
```

### 4. 手动解除冻结

```javascript
// 手动解除冻结
const unfreezeData = {
  symbol: "BTCUSDT",
  strategy_name: "line3_coin6",
  trade_type: "real"
};

fetch('/strategy-freeze/unfreeze', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify(unfreezeData)
})
.then(response => response.json())
.then(data => {
  console.log('解除冻结结果:', data.message);
});
```

## 工作流程

1. **策略执行前检查**: 在策略执行前，系统会检查该 `symbol-strategy-trade_type` 组合是否被冻结
2. **记录交易结果**: 每次策略平仓后，系统会记录盈亏情况
   - 如果盈利，亏损次数清零
   - 如果亏损，亏损次数加1
3. **自动冻结**: 当亏损次数达到设定的阈值时，系统会自动冻结该策略组合指定的时间
4. **冻结期间**: 在冻结期间，该策略组合不会执行任何交易
5. **自动解冻**: 冻结时间到期后，系统会自动解除冻结状态

## 注意事项

1. 实盘交易（real）和测试交易（test）的风控是完全独立的
2. 每个 `symbol-strategy-trade_type` 组合都有独立的风控参数
3. 风控信息会持久化存储，服务重启后不会丢失
4. 可以通过 API 手动解除冻结或重置亏损次数
5. 系统会自动为新的策略组合创建默认风控配置（5次亏损后冻结24小时）

## 错误码说明

- `200`: 成功
- `400`: 参数错误
- `500`: 服务器内部错误