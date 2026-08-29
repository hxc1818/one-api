# Wallet API 示例客户端

演示如何从其他服务中消费 Wallet API 的余额更新事件。

## 安装依赖

```bash
go mod init wallet-api-client
go get github.com/go-redis/redis/v8
```

## 消费者示例代码

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
)

type BalanceUpdateEvent struct {
	UserID        int       `json:"user_id"`
	Username      string    `json:"username"`
	BalanceBefore int64     `json:"balance_before"`
	BalanceAfter  int64     `json:"balance_after"`
	ChangeAmount  int64     `json:"change_amount"`
	Type          string    `json:"type"` // deduct, refund
	TransactionID uint      `json:"transaction_id"`
	Timestamp     time.Time `json:"timestamp"`
	Description   string    `json:"description"`
}

func main() {
	// 连接 Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // 如果有密码
		DB:       0,
	})

	ctx := context.Background()

	// 测试连接
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	streamKey := "wallet:balance:updates"
	groupName := "my-service-group"      // 你的服务消费者组名
	consumerName := "consumer-instance-1" // 消费者实例名

	// 创建消费者组（如果不存在）
	err := rdb.XGroupCreateMkStream(ctx, streamKey, groupName, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		log.Printf("Warning: Failed to create consumer group: %v", err)
	} else {
		log.Printf("Consumer group '%s' created or already exists", groupName)
	}

	// 优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Starting to consume balance update events...")

	// 消费循环
	go func() {
		for {
			// 读取消息（阻塞 5 秒）
			streams, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    groupName,
				Consumer: consumerName,
				Streams:  []string{streamKey, ">"},
				Count:    10,
				Block:    5 * time.Second,
			}).Result()

			if err != nil {
				if err == redis.Nil {
					// 没有新消息
					continue
				}
				log.Printf("Error reading stream: %v", err)
				time.Sleep(time.Second)
				continue
			}

			// 处理消息
			for _, stream := range streams {
				for _, message := range stream.Messages {
					if err := processMessage(ctx, rdb, streamKey, groupName, message); err != nil {
						log.Printf("Error processing message %s: %v", message.ID, err)
						// 可以选择重试或记录到死信队列
					}
				}
			}
		}
	}()

	// 等待退出信号
	<-sigChan
	log.Println("Shutting down...")
}

func processMessage(ctx context.Context, rdb *redis.Client, streamKey, groupName string, message redis.XMessage) error {
	// 解析事件数据
	dataStr, ok := message.Values["data"].(string)
	if !ok {
		return fmt.Errorf("invalid message format")
	}

	var event BalanceUpdateEvent
	if err := json.Unmarshal([]byte(dataStr), &event); err != nil {
		return fmt.Errorf("failed to parse event: %w", err)
	}

	// 打印事件信息
	log.Printf("=== Balance Update Event ===")
	log.Printf("User: %s (ID: %d)", event.Username, event.UserID)
	log.Printf("Type: %s", event.Type)
	log.Printf("Change: %d", event.ChangeAmount)
	log.Printf("Balance: %d -> %d", event.BalanceBefore, event.BalanceAfter)
	log.Printf("Transaction ID: %d", event.TransactionID)
	log.Printf("Description: %s", event.Description)
	log.Printf("Timestamp: %s", event.Timestamp.Format(time.RFC3339))
	log.Printf("===========================")

	// 这里添加你的业务逻辑
	// 例如：同步到你的数据库、触发其他操作等
	if err := syncToYourService(event); err != nil {
		return fmt.Errorf("failed to sync to service: %w", err)
	}

	// 确认消息已处理
	if err := rdb.XAck(ctx, streamKey, groupName, message.ID).Err(); err != nil {
		return fmt.Errorf("failed to ack message: %w", err)
	}

	return nil
}

func syncToYourService(event BalanceUpdateEvent) error {
	// 实现你的同步逻辑
	// 例如：
	// 1. 更新本地数据库
	// 2. 发送 webhook 通知
	// 3. 触发其他业务流程
	
	// 这里只是示例
	fmt.Printf("Syncing user %d balance to local service...\n", event.UserID)
	
	return nil
}
```

## HTTP API 调用示例

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	baseURL = "http://localhost:8081/api/v1"
)

type AuthResponse struct {
	Token    string `json:"token"`
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Balance  int64  `json:"balance"`
}

type DeductRequest struct {
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
}

type DeductResponse struct {
	Success       bool  `json:"success"`
	TransactionID uint  `json:"transaction_id"`
	BalanceBefore int64 `json:"balance_before"`
	BalanceAfter  int64 `json:"balance_after"`
	Amount        int64 `json:"amount"`
}

func main() {
	// 1. 用户认证
	token, err := authenticate("user123", "password123")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Authenticated successfully, token: %s\n", token)

	// 2. 查询余额
	balance, err := getBalance(token)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Current balance: %d\n", balance)

	// 3. 扣款
	txnID, err := deductBalance(token, 100, "测试扣款")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Deduction successful, transaction ID: %d\n", txnID)

	// 4. 查询余额（验证扣款）
	balance, err = getBalance(token)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Balance after deduction: %d\n", balance)
}

func authenticate(username, password string) (string, error) {
	payload := map[string]string{
		"username": username,
		"password": password,
	}
	
	data, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/auth/verify", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("authentication failed: %s", body)
	}

	var result AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Token, nil
}

func getBalance(token string) (int64, error) {
	req, _ := http.NewRequest("GET", baseURL+"/balance", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("get balance failed: %s", body)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return int64(result["balance"].(float64)), nil
}

func deductBalance(token string, amount int64, description string) (uint, error) {
	payload := DeductRequest{
		Amount:      amount,
		Description: description,
	}

	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", baseURL+"/deduct", bytes.NewBuffer(data))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("deduct balance failed: %s", body)
	}

	var result DeductResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.TransactionID, nil
}
```

## Python 消费者示例

```python
import redis
import json
import time
import signal
import sys

class BalanceEventConsumer:
    def __init__(self, redis_host='localhost', redis_port=6379, redis_password=None):
        self.redis_client = redis.Redis(
            host=redis_host,
            port=redis_port,
            password=redis_password,
            decode_responses=True
        )
        self.stream_key = 'wallet:balance:updates'
        self.group_name = 'python-service-group'
        self.consumer_name = 'consumer-1'
        self.running = True
        
        # 创建消费者组
        try:
            self.redis_client.xgroup_create(
                self.stream_key,
                self.group_name,
                id='0',
                mkstream=True
            )
            print(f"Consumer group '{self.group_name}' created")
        except redis.ResponseError as e:
            if 'BUSYGROUP' in str(e):
                print(f"Consumer group '{self.group_name}' already exists")
            else:
                raise

    def process_event(self, event_data):
        """处理余额更新事件"""
        print("=== Balance Update Event ===")
        print(f"User: {event_data['username']} (ID: {event_data['user_id']})")
        print(f"Type: {event_data['type']}")
        print(f"Change: {event_data['change_amount']}")
        print(f"Balance: {event_data['balance_before']} -> {event_data['balance_after']}")
        print(f"Transaction ID: {event_data['transaction_id']}")
        print(f"Description: {event_data['description']}")
        print("===========================\n")
        
        # 在这里添加你的业务逻辑
        # 例如：同步到数据库、发送通知等

    def consume(self):
        """消费事件循环"""
        print(f"Starting to consume events from stream '{self.stream_key}'...")
        
        while self.running:
            try:
                # 读取消息
                messages = self.redis_client.xreadgroup(
                    self.group_name,
                    self.consumer_name,
                    {self.stream_key: '>'},
                    count=10,
                    block=5000
                )
                
                if not messages:
                    continue
                
                for stream_name, stream_messages in messages:
                    for message_id, message_data in stream_messages:
                        try:
                            # 解析事件数据
                            event_json = message_data.get('data')
                            if event_json:
                                event = json.loads(event_json)
                                self.process_event(event)
                            
                            # 确认消息
                            self.redis_client.xack(
                                self.stream_key,
                                self.group_name,
                                message_id
                            )
                        except Exception as e:
                            print(f"Error processing message {message_id}: {e}")
                            # 可以选择重试或移到死信队列
                            
            except Exception as e:
                print(f"Error in consume loop: {e}")
                time.sleep(1)

    def stop(self):
        """停止消费"""
        self.running = False
        print("Stopping consumer...")

def signal_handler(sig, frame):
    """处理退出信号"""
    print('\nReceived shutdown signal')
    consumer.stop()
    sys.exit(0)

if __name__ == '__main__':
    # 创建消费者
    consumer = BalanceEventConsumer(
        redis_host='localhost',
        redis_port=6379,
        redis_password=None
    )
    
    # 注册信号处理
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    # 开始消费
    consumer.consume()
```

## 运行说明

1. 启动 Wallet API 服务
2. 运行消费者代码监听余额变更
3. 通过 API 调用进行扣款操作
4. 消费者会收到余额变更事件并处理
