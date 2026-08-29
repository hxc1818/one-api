package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/wallet-api/config"
	"github.com/songquanpeng/one-api/wallet-api/handler"
	"github.com/songquanpeng/one-api/wallet-api/middleware"
	"github.com/songquanpeng/one-api/wallet-api/service"
)

func main() {
	// 初始化配置
	if err := config.InitConfig(); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	// 初始化数据库
	if err := config.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer config.CloseDB()

	// 初始化 Redis
	if err := config.InitRedis(); err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer config.CloseRedis()

	// 初始化事件推送服务
	eventService := service.NewEventService(config.GetRedis())
	eventService.Start(context.Background())
	defer eventService.Stop()

	// 初始化 handlers
	handler.InitHandlers()

	// 初始化 HTTP 服务
	r := gin.Default()

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API 路由组
	api := r.Group("/api/v1")
	{
		// 认证相关
		api.POST("/auth/verify", handler.VerifyUser)

		// 需要认证的路由
		auth := api.Group("")
		auth.Use(middleware.AuthMiddleware())
		{
			// 余额查询
			auth.GET("/balance", handler.GetBalance)

			// 扣款
			auth.POST("/deduct", handler.DeductBalance)

			// 回退扣款
			auth.POST("/refund", handler.RefundDeduction)

			// 订阅管理（管理员功能）
			auth.POST("/subscriptions", middleware.AdminOnly(), handler.AddSubscription)
			auth.DELETE("/subscriptions/:id", middleware.AdminOnly(), handler.RemoveSubscription)
			auth.GET("/subscriptions", middleware.AdminOnly(), handler.ListSubscriptions)

			// 查询交易记录
			auth.GET("/transactions", handler.GetTransactions)
		}
	}

	// 启动服务
	port := os.Getenv("WALLET_API_PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Wallet API server starting on port %s...", port)
	if err := r.Run(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
