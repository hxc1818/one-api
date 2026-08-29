# Wallet API

这是一个独立的钱包 API 服务，用于与 One-API 的用户余额进行同步和管理。

## 功能特性

- ✅ 用户认证（用户名密码验证）
- ✅ 查询用户余额
- ✅ 发起扣款操作
- ✅ 回退扣款操作
- ✅ 订阅域名管理
- ✅ 基于 Redis Stream 的余额变更事件推送
- ✅ 完整的交易记录
- ❌ 不支持充值操作（仅扣款和回退）

## 架构设计

### 事件推送机制

使用 **Redis Stream** 作为消息队列，实现高性能的异步事件推送：

1. **事件生产者**：当余额发生变更（扣款/回退）时，将事件推送到 Redis Stream
2. **事件消费者**：订阅方可以通过消费者组（Consumer Group）订阅事件
3. **解耦设计**：扣款操作立即返回，不受订阅方网络影响
4. **可靠性**：支持消息确认、重试机制

### 数据库表结构

#### transactions 表
记录所有交易（扣款、回退）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| user_id | int | 用户ID |
| username | string | 用户名 |
| type | string | 类型：deduct/refund |
| amount | int64 | 金额 |
| balance_before | int64 | 变更前余额 |
| balance_after | int64 | 变更后余额 |
| description | string | 描述 |
| ref_txn_id | uint | 关联交易ID（回退时使用） |
| created_at | timestamp | 创建时间 |

#### subscriptions 表
订阅域名管理

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| webhook_url | string | Webhook URL（唯一） |
| description | string | 描述 |
| active | bool | 是否启用 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

## 环境变量配置

```bash
# 数据库配置（与 one-api 使用相同的数据库）
SQL_DSN=root:password@tcp(localhost:3306)/oneapi?charset=utf8mb4&parseTime=True&loc=Local

# Redis 配置（必需）
REDIS_CONN_STRING=redis://localhost:6379/0
REDIS_PASSWORD=your_redis_password

# Redis Stream 配置
REDIS_STREAM_KEY=wallet:balance:updates  # Stream 键名
REDIS_STREAM_MAXLEN=10000                # 最大消息数

# JWT 密钥
JWT_SECRET=your-secret-key-change-me

# 服务端口
WALLET_API_PORT=8081
```

## API 接口文档

### 1. 用户认证

**POST** `/api/v1/auth/verify`

验证用户名密码，返回 JWT Token 和用户余额。

请求体：
```json
{
  "username": "user123",
  "password": "password123"
}
```

响应：
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user_id": 1,
  "username": "user123",
  "balance": 1000000
}
```

### 2. 查询余额

**GET** `/api/v1/balance`

需要 JWT Token 认证。

响应：
```json
{
  "user_id": 1,
  "balance": 1000000
}
```

### 3. 发起扣款

**POST** `/api/v1/deduct`

需要 JWT Token 认证。

请求体：
```json
{
  "amount": 100,
  "description": "API 调用费用"
}
```

响应：
```json
{
  "success": true,
  "transaction_id": 123,
  "balance_before": 1000000,
  "balance_after": 999900,
  "amount": 100
}
```

### 4. 回退扣款

**POST** `/api/v1/refund`

需要 JWT Token 认证。只能回退扣款操作，不能充值。

请求体：
```json
{
  "transaction_id": 123,
  "description": "退款原因"
}
```

响应：
```json
{
  "success": true,
  "transaction_id": 124,
  "balance_before": 999900,
  "balance_after": 1000000,
  "amount": 100,
  "ref_txn_id": 123
}
```

### 5. 查询交易记录

**GET** `/api/v1/transactions?page=1&page_size=20`

需要 JWT Token 认证。

响应：
```json
{
  "transactions": [
    {
      "id": 124,
      "user_id": 1,
      "username": "user123",
      "type": "refund",
      "amount": 100,
      "balance_before": 999900,
      "balance_after": 1000000,
      "description": "退款原因",
      "ref_txn_id": 123,
      "created_at": "2024-01-01T12:00:00Z"
    }
  ],
  "total": 50,
  "page": 1,
  "page_size": 20
}
```

### 6. 添加订阅（管理员）

**POST** `/api/v1/subscriptions`

需要管理员权限。

请求体：
```json
{
  "webhook_url": "https://your-service.com/webhook",
  "description": "生态服务 A"
}
```

### 7. 删除订阅（管理员）

**DELETE** `/api/v1/subscriptions/:id`

需要管理员权限。

### 8. 列出订阅（管理员）

**GET** `/api/v1/subscriptions`

需要管理员权限。

## Redis Stream 事件格式

### Stream Key
默认：`wallet:balance:updates`

### 事件数据结构
```json
{
  "user_id": 1,
  "username": "user123",
  "balance_before": 1000000,
  "balance_after": 999900,
  "change_amount": -100,
  "type": "deduct",
  "transaction_id": 123,
  "timestamp": "2024-01-01T12:00:00Z",
  "description": "API 调用费用"
}
```

## 订阅方消费示例

### 方式 1：使用 XREAD（简单模式）

```bash
# 从最新位置开始读取
redis-cli XREAD BLOCK 5000 STREAMS wallet:balance:updates $

# 从指定位置读取
redis-cli XREAD BLOCK 5000 STREAMS wallet:balance:updates 1234567890123-0
```

### 方式 2：使用消费者组（推荐）

```bash
# 创建消费者组
redis-cli XGROUP CREATE wallet:balance:updates mygroup 0 MKSTREAM

# 消费消息
redis-cli XREADGROUP GROUP mygroup consumer1 BLOCK 5000 COUNT 10 STREAMS wallet:balance:updates >

# 确认消息
redis-cli XACK wallet:balance:updates mygroup <message-id>
```

### Go 代码示例

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"

    "github.com/go-redis/redis/v8"
)

type BalanceUpdateEvent struct {
    UserID        int       `json:"user_id"`
    Username      string    `json:"username"`
    BalanceBefore int64     `json:"balance_before"`
    BalanceAfter  int64     `json:"balance_after"`
    ChangeAmount  int64     `json:"change_amount"`
    Type          string    `json:"type"`
    TransactionID uint      `json:"transaction_id"`
    Timestamp     time.Time `json:"timestamp"`
    Description   string    `json:"description"`
}

func main() {
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    ctx := context.Background()
    streamKey := "wallet:balance:updates"
    groupName := "my-service-group"
    consumerName := "consumer-1"

    // 创建消费者组（如果不存在）
    rdb.XGroupCreateMkStream(ctx, streamKey, groupName, "0").Err()

    for {
        // 读取消息
        streams, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
            Group:    groupName,
            Consumer: consumerName,
            Streams:  []string{streamKey, ">"},
            Count:    10,
            Block:    5 * time.Second,
        }).Result()

        if err != nil {
            if err == redis.Nil {
                continue
            }
            log.Printf("Error reading stream: %v", err)
            time.Sleep(time.Second)
            continue
        }

        for _, stream := range streams {
            for _, message := range stream.Messages {
                // 解析事件数据
                dataStr := message.Values["data"].(string)
                var event BalanceUpdateEvent
                if err := json.Unmarshal([]byte(dataStr), &event); err != nil {
                    log.Printf("Error parsing event: %v", err)
                    continue
                }

                // 处理事件
                fmt.Printf("User %s balance changed: %d -> %d (type: %s)\n",
                    event.Username, event.BalanceBefore, event.BalanceAfter, event.Type)

                // 确认消息已处理
                rdb.XAck(ctx, streamKey, groupName, message.ID).Err()
            }
        }
    }
}
```

## 部署运行

### 1. 编译

```bash
cd wallet-api
go mod tidy
go build -o wallet-api
```

### 2. 运行

```bash
# 设置环境变量
export SQL_DSN="root:password@tcp(localhost:3306)/oneapi?charset=utf8mb4&parseTime=True&loc=Local"
export REDIS_CONN_STRING="redis://localhost:6379/0"
export JWT_SECRET="your-secret-key"
export WALLET_API_PORT="8081"

# 运行服务
./wallet-api
```

### 3. Docker 部署

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o wallet-api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/wallet-api .
EXPOSE 8081
CMD ["./wallet-api"]
```

## 安全说明

1. **JWT Token**：所有需要认证的接口都需要在 Header 中携带 JWT Token
2. **管理员权限**：订阅管理接口仅限管理员用户使用
3. **密码验证**：使用与 one-api 相同的密码哈希验证机制
4. **事务保证**：扣款和回退操作使用数据库事务，保证数据一致性
5. **防重复回退**：每笔扣款只能回退一次

## 性能特点

- ✅ **异步事件推送**：扣款操作不阻塞，立即返回
- ✅ **Redis Stream 高性能**：支持每秒数万条消息
- ✅ **消费者组**：多个订阅方可以并行消费
- ✅ **数据库连接池**：优化数据库连接性能
- ✅ **事务保证**：确保余额变更和记录写入的原子性

## 注意事项

1. 本 API **不支持充值操作**，只能扣款和回退
2. 回退操作只能回退扣款，每笔扣款只能回退一次
3. 需要与 one-api 使用相同的数据库
4. Redis 是必需的依赖，用于事件推送
5. 订阅方需要自己实现消费者逻辑来处理余额变更事件
