package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/songquanpeng/one-api/wallet-api/config"
	"github.com/songquanpeng/one-api/wallet-api/middleware"
	"github.com/songquanpeng/one-api/wallet-api/service"
)

var (
	walletService       *service.WalletService
	subscriptionService *service.SubscriptionService
	eventService        *service.EventService
)

func init() {
	// 延迟初始化，在 main 函数中调用 InitHandlers
}

func InitHandlers() {
	eventService = service.NewEventService(config.GetRedis())
	walletService = service.NewWalletService(config.GetDB(), eventService)
	subscriptionService = service.NewSubscriptionService(config.GetDB())
}

// VerifyUser 验证用户名密码并返回 JWT Token
func VerifyUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	user, err := walletService.ValidateUser(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 生成 JWT Token
	claims := &middleware.Claims{
		UserID:   user.Id,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.GetConfig().JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// 获取余额
	balance, err := walletService.GetBalance(user.Id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get balance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":    tokenString,
		"user_id":  user.Id,
		"username": user.Username,
		"balance":  balance,
	})
}

// GetBalance 获取用户余额
func GetBalance(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	balance, err := walletService.GetBalance(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"balance": balance,
	})
}

// DeductBalance 扣款
func DeductBalance(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	var req struct {
		Amount      int64  `json:"amount" binding:"required,gt=0"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	txn, err := walletService.DeductBalance(c.Request.Context(), userID, req.Amount, req.Description)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"transaction_id": txn.ID,
		"balance_before": txn.BalanceBefore,
		"balance_after":  txn.BalanceAfter,
		"amount":         txn.Amount,
	})
}

// RefundDeduction 回退扣款
func RefundDeduction(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	var req struct {
		TransactionID uint   `json:"transaction_id" binding:"required"`
		Description   string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	txn, err := walletService.RefundDeduction(c.Request.Context(), req.TransactionID, req.Description)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证是否是用户自己的交易
	if txn.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"transaction_id": txn.ID,
		"balance_before": txn.BalanceBefore,
		"balance_after":  txn.BalanceAfter,
		"amount":         txn.Amount,
		"ref_txn_id":     txn.RefTxnID,
	})
}

// GetTransactions 获取交易记录
func GetTransactions(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	// 分页参数
	var req struct {
		Page     int `form:"page" binding:"omitempty,min=1"`
		PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters"})
		return
	}

	// 默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	offset := (req.Page - 1) * req.PageSize
	transactions, total, err := walletService.GetTransactions(userID, req.PageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"total":        total,
		"page":         req.Page,
		"page_size":    req.PageSize,
	})
}

// AddSubscription 添加订阅
func AddSubscription(c *gin.Context) {
	var req struct {
		WebhookURL  string `json:"webhook_url" binding:"required,url"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	subscription, err := subscriptionService.AddSubscription(req.WebhookURL, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"subscription": subscription,
	})
}

// RemoveSubscription 删除订阅
func RemoveSubscription(c *gin.Context) {
	var req struct {
		ID uint `uri:"id" binding:"required"`
	}

	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
		return
	}

	if err := subscriptionService.RemoveSubscription(req.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ListSubscriptions 列出所有订阅
func ListSubscriptions(c *gin.Context) {
	subscriptions, err := subscriptionService.ListSubscriptions(nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscriptions": subscriptions,
	})
}
