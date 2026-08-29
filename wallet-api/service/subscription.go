package service

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/songquanpeng/one-api/wallet-api/config"
)

type SubscriptionService struct {
	db *gorm.DB
}

func NewSubscriptionService(db *gorm.DB) *SubscriptionService {
	return &SubscriptionService{
		db: db,
	}
}

// AddSubscription 添加订阅域名
func (s *SubscriptionService) AddSubscription(webhookURL, description string) (*config.Subscription, error) {
	if webhookURL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	subscription := &config.Subscription{
		WebhookURL:  webhookURL,
		Description: description,
		Active:      true,
	}

	if err := s.db.Create(subscription).Error; err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	return subscription, nil
}

// RemoveSubscription 删除订阅
func (s *SubscriptionService) RemoveSubscription(id uint) error {
	result := s.db.Delete(&config.Subscription{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete subscription: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("subscription not found")
	}
	return nil
}

// ListSubscriptions 列出所有订阅
func (s *SubscriptionService) ListSubscriptions(active *bool) ([]config.Subscription, error) {
	var subscriptions []config.Subscription
	query := s.db.Order("created_at DESC")

	if active != nil {
		query = query.Where("active = ?", *active)
	}

	if err := query.Find(&subscriptions).Error; err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	return subscriptions, nil
}

// GetSubscription 获取单个订阅
func (s *SubscriptionService) GetSubscription(id uint) (*config.Subscription, error) {
	var subscription config.Subscription
	if err := s.db.First(&subscription, id).Error; err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	return &subscription, nil
}

// UpdateSubscriptionStatus 更新订阅状态
func (s *SubscriptionService) UpdateSubscriptionStatus(id uint, active bool) error {
	result := s.db.Model(&config.Subscription{}).Where("id = ?", id).Update("active", active)
	if result.Error != nil {
		return fmt.Errorf("failed to update subscription status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("subscription not found")
	}
	return nil
}

// GetActiveSubscriptions 获取所有活跃的订阅（用于推送通知）
func (s *SubscriptionService) GetActiveSubscriptions() ([]config.Subscription, error) {
	var subscriptions []config.Subscription
	if err := s.db.Where("active = ?", true).Find(&subscriptions).Error; err != nil {
		return nil, fmt.Errorf("failed to get active subscriptions: %w", err)
	}
	return subscriptions, nil
}
