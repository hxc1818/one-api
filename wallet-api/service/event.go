package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/songquanpeng/one-api/wallet-api/config"
)

// BalanceUpdateEvent 余额更新事件
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

// EventService 事件推送服务
type EventService struct {
	rdb       *redis.Client
	streamKey string
	maxLen    int64
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

func NewEventService(rdb *redis.Client) *EventService {
	cfg := config.GetConfig()
	return &EventService{
		rdb:       rdb,
		streamKey: cfg.StreamKey,
		maxLen:    cfg.StreamMaxLen,
	}
}

// Start 启动事件服务
func (s *EventService) Start(ctx context.Context) {
	s.ctx, s.cancel = context.WithCancel(ctx)
	log.Printf("Event service started, stream key: %s", s.streamKey)
}

// Stop 停止事件服务
func (s *EventService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	log.Println("Event service stopped")
}

// PublishBalanceUpdate 发布余额更新事件到 Redis Stream
func (s *EventService) PublishBalanceUpdate(event *BalanceUpdateEvent) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	args := &redis.XAddArgs{
		Stream: s.streamKey,
		MaxLen: s.maxLen,
		Approx: true, // 使用近似裁剪，性能更好
		Values: map[string]interface{}{
			"event_type": "balance_update",
			"data":       string(data),
			"timestamp":  event.Timestamp.Unix(),
		},
	}

	id, err := s.rdb.XAdd(s.ctx, args).Result()
	if err != nil {
		return fmt.Errorf("failed to publish event to Redis Stream: %w", err)
	}

	log.Printf("Published balance update event: user_id=%d, stream_id=%s", event.UserID, id)
	return nil
}

// GetStreamKey 获取 Stream Key
func (s *EventService) GetStreamKey() string {
	return s.streamKey
}

// ReadEvents 读取事件（供订阅方使用）
// 这是一个示例方法，实际订阅方应该自己实现消费逻辑
func (s *EventService) ReadEvents(lastID string, count int64, block time.Duration) ([]redis.XMessage, error) {
	if lastID == "" {
		lastID = "0" // 从头开始读取
	}

	streams, err := s.rdb.XRead(s.ctx, &redis.XReadArgs{
		Streams: []string{s.streamKey, lastID},
		Count:   count,
		Block:   block,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, nil // 没有新消息
		}
		return nil, fmt.Errorf("failed to read from stream: %w", err)
	}

	if len(streams) == 0 {
		return nil, nil
	}

	return streams[0].Messages, nil
}

// CreateConsumerGroup 创建消费者组
func (s *EventService) CreateConsumerGroup(groupName string) error {
	err := s.rdb.XGroupCreateMkStream(s.ctx, s.streamKey, groupName, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	log.Printf("Consumer group '%s' created or already exists", groupName)
	return nil
}

// ReadGroupEvents 使用消费者组读取事件
func (s *EventService) ReadGroupEvents(groupName, consumerName string, count int64, block time.Duration) ([]redis.XMessage, error) {
	streams, err := s.rdb.XReadGroup(s.ctx, &redis.XReadGroupArgs{
		Group:    groupName,
		Consumer: consumerName,
		Streams:  []string{s.streamKey, ">"},
		Count:    count,
		Block:    block,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read from consumer group: %w", err)
	}

	if len(streams) == 0 {
		return nil, nil
	}

	return streams[0].Messages, nil
}

// AckMessage 确认消息已处理
func (s *EventService) AckMessage(groupName string, messageID string) error {
	return s.rdb.XAck(s.ctx, s.streamKey, groupName, messageID).Err()
}
