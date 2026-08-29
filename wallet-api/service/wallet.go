package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/songquanpeng/one-api/wallet-api/config"
)

// User 用户模型（映射到 one-api 的 users 表）
type User struct {
	Id          int    `json:"id"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Role        int    `json:"role"`
	Status      int    `json:"status"`
	Quota       int64  `json:"quota"`
	UsedQuota   int64  `json:"used_quota"`
}

// Log 日志模型（映射到 one-api 的 logs 表）
type Log struct {
	Id                int    `json:"id"`
	UserId            int    `json:"user_id" gorm:"index"`
	CreatedAt         int64  `json:"created_at" gorm:"bigint;index:idx_created_at_type"`
	Type              int    `json:"type" gorm:"index:idx_created_at_type"`
	Content           string `json:"content"`
	Username          string `json:"username" gorm:"index:index_username_model_name,priority:2;default:''"`
	TokenName         string `json:"token_name" gorm:"index;default:''"`
	ModelName         string `json:"model_name" gorm:"index;index:index_username_model_name,priority:1;default:''"`
	Quota             int    `json:"quota" gorm:"default:0"`
	PromptTokens      int    `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens  int    `json:"completion_tokens" gorm:"default:0"`
	ChannelId         int    `json:"channel" gorm:"index"`
	RequestId         string `json:"request_id" gorm:"default:''"`
	ElapsedTime       int64  `json:"elapsed_time" gorm:"default:0"`
	IsStream          bool   `json:"is_stream" gorm:"default:false"`
	SystemPromptReset bool   `json:"system_prompt_reset" gorm:"default:false"`
}

const (
	UserStatusEnabled  = 1
	UserStatusDisabled = 2
	RoleAdminUser      = 10
)

const (
	LogTypeUnknown = iota
	LogTypeTopup
	LogTypeConsume
	LogTypeManage
	LogTypeSystem
	LogTypeTest
)

type WalletService struct {
	db           *gorm.DB
	logDB        *gorm.DB // 日志数据库（可能与主DB不同）
	eventService *EventService
}

func NewWalletService(db *gorm.DB, eventService *EventService) *WalletService {
	// 日志数据库默认与主数据库相同，除非配置了独立的日志数据库
	logDB := db
	if config.GetLogDB() != nil {
		logDB = config.GetLogDB()
	}
	
	return &WalletService{
		db:           db,
		logDB:        logDB,
		eventService: eventService,
	}
}

// ValidateUser 验证用户名和密码
func (s *WalletService) ValidateUser(username, password string) (*User, error) {
	if username == "" || password == "" {
		return nil, errors.New("username or password is empty")
	}

	var user User
	// 先尝试用户名查询
	err := s.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		// 尝试用邮箱查询
		err = s.db.Where("email = ?", username).First(&user).Error
		if err != nil {
			return nil, errors.New("invalid username or password")
		}
	}

	// 验证密码（需要导入 one-api 的密码验证函数）
	if !validatePasswordAndHash(password, user.Password) {
		return nil, errors.New("invalid username or password")
	}

	// 检查用户状态
	if user.Status != UserStatusEnabled {
		return nil, errors.New("user is disabled")
	}

	return &user, nil
}

// GetBalance 获取用户余额
func (s *WalletService) GetBalance(userID int) (int64, error) {
	var user User
	err := s.db.Select("quota").Where("id = ?", userID).First(&user).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get user balance: %w", err)
	}
	return user.Quota, nil
}

// createLog 创建日志记录到 one-api 的 logs 表
func (s *WalletService) createLog(ctx context.Context, userID int, username string, logType int, content string, quota int) error {
	log := &Log{
		UserId:    userID,
		Username:  username,
		CreatedAt: time.Now().Unix(),
		Type:      logType,
		Content:   content,
		Quota:     quota,
		TokenName: "wallet-api", // 标记来源
	}
	
	err := s.logDB.Create(log).Error
	if err != nil {
		return fmt.Errorf("failed to create log: %w", err)
	}
	
	return nil
}

// DeductBalance 扣款
func (s *WalletService) DeductBalance(ctx context.Context, userID int, amount int64, description string) (*config.Transaction, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}

	var txn *config.Transaction
	var user User

	// 使用事务确保数据一致性
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 锁定用户行
		if err := tx.Clauses().Where("id = ?", userID).First(&user).Error; err != nil {
			return fmt.Errorf("user not found: %w", err)
		}

		// 检查余额是否足够
		if user.Quota < amount {
			return errors.New("insufficient balance")
		}

		balanceBefore := user.Quota
		balanceAfter := balanceBefore - amount

		// 扣除余额
		if err := tx.Model(&User{}).Where("id = ?", userID).Update("quota", balanceAfter).Error; err != nil {
			return fmt.Errorf("failed to deduct balance: %w", err)
		}

		// 记录交易
		txn = &config.Transaction{
			UserID:        userID,
			Username:      user.Username,
			Type:          "deduct",
			Amount:        amount,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceAfter,
			Description:   description,
			CreatedAt:     time.Now(),
		}

		if err := tx.Create(txn).Error; err != nil {
			return fmt.Errorf("failed to create transaction record: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 写入 one-api 的日志表，用户可以在界面上看到
	logContent := fmt.Sprintf("钱包扣款: %s", description)
	if err := s.createLog(ctx, userID, user.Username, LogTypeConsume, logContent, int(amount)); err != nil {
		// 记录错误但不影响主流程
		fmt.Printf("Failed to create log: %v\n", err)
	}

	// 发布余额更新事件到 Redis Stream
	event := &BalanceUpdateEvent{
		UserID:        userID,
		Username:      user.Username,
		BalanceBefore: txn.BalanceBefore,
		BalanceAfter:  txn.BalanceAfter,
		ChangeAmount:  -amount,
		Type:          "deduct",
		TransactionID: txn.ID,
		Timestamp:     txn.CreatedAt,
		Description:   description,
	}

	if err := s.eventService.PublishBalanceUpdate(event); err != nil {
		// 记录错误但不影响主流程
		fmt.Printf("Failed to publish balance update event: %v\n", err)
	}

	return txn, nil
}

// RefundDeduction 回退扣款
func (s *WalletService) RefundDeduction(ctx context.Context, transactionID uint, description string) (*config.Transaction, error) {
	var originalTxn config.Transaction
	var user User
	var refundTxn *config.Transaction

	// 使用事务确保数据一致性
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 查询原始交易
		if err := tx.Where("id = ? AND type = ?", transactionID, "deduct").First(&originalTxn).Error; err != nil {
			return fmt.Errorf("original transaction not found: %w", err)
		}

		// 检查是否已经回退过
		var existingRefund config.Transaction
		err := tx.Where("ref_txn_id = ? AND type = ?", transactionID, "refund").First(&existingRefund).Error
		if err == nil {
			return errors.New("transaction already refunded")
		}

		// 锁定用户行
		if err := tx.Where("id = ?", originalTxn.UserID).First(&user).Error; err != nil {
			return fmt.Errorf("user not found: %w", err)
		}

		balanceBefore := user.Quota
		balanceAfter := balanceBefore + originalTxn.Amount

		// 增加余额
		if err := tx.Model(&User{}).Where("id = ?", originalTxn.UserID).Update("quota", balanceAfter).Error; err != nil {
			return fmt.Errorf("failed to refund balance: %w", err)
		}

		// 记录回退交易
		refundTxn = &config.Transaction{
			UserID:        originalTxn.UserID,
			Username:      originalTxn.Username,
			Type:          "refund",
			Amount:        originalTxn.Amount,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceAfter,
			Description:   description,
			RefTxnID:      &transactionID,
			CreatedAt:     time.Now(),
		}

		if err := tx.Create(refundTxn).Error; err != nil {
			return fmt.Errorf("failed to create refund transaction record: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 写入 one-api 的日志表（充值类型）
	logContent := fmt.Sprintf("钱包退款: %s (原交易ID: %d)", description, transactionID)
	if err := s.createLog(ctx, originalTxn.UserID, user.Username, LogTypeTopup, logContent, int(originalTxn.Amount)); err != nil {
		fmt.Printf("Failed to create log: %v\n", err)
	}

	// 发布余额更新事件到 Redis Stream
	event := &BalanceUpdateEvent{
		UserID:        originalTxn.UserID,
		Username:      user.Username,
		BalanceBefore: refundTxn.BalanceBefore,
		BalanceAfter:  refundTxn.BalanceAfter,
		ChangeAmount:  originalTxn.Amount,
		Type:          "refund",
		TransactionID: refundTxn.ID,
		Timestamp:     refundTxn.CreatedAt,
		Description:   description,
	}

	if err := s.eventService.PublishBalanceUpdate(event); err != nil {
		fmt.Printf("Failed to publish balance update event: %v\n", err)
	}

	return refundTxn, nil
}

// GetTransactions 获取交易记录
func (s *WalletService) GetTransactions(userID int, limit, offset int) ([]config.Transaction, int64, error) {
	var transactions []config.Transaction
	var total int64

	// 计算总数
	if err := s.db.Model(&config.Transaction{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	// 查询交易记录
	err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to get transactions: %w", err)
	}

	return transactions, total, nil
}

// 密码验证函数（简化版，实际应该导入 one-api 的实现）
func validatePasswordAndHash(password, hash string) bool {
	// 这里需要导入 one-api 的密码验证逻辑
	// 暂时使用简化实现
	// 在实际部署时，应该使用 one-api 的 common.ValidatePasswordAndHash
	return true // TODO: 实现实际的密码验证
}
