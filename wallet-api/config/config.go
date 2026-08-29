package config

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	DB     *gorm.DB
	LOG_DB *gorm.DB
	RDB    *redis.Client
)

type Config struct {
	DatabaseDSN      string
	LogDatabaseDSN   string
	RedisConnString  string
	RedisPassword    string
	JWTSecret        string
	StreamKey        string
	StreamMaxLen     int64
}

var cfg *Config

func InitConfig() error {
	cfg = &Config{
		DatabaseDSN:     os.Getenv("SQL_DSN"),
		LogDatabaseDSN:  os.Getenv("LOG_SQL_DSN"),
		RedisConnString: os.Getenv("REDIS_CONN_STRING"),
		RedisPassword:   os.Getenv("REDIS_PASSWORD"),
		JWTSecret:       getEnvOrDefault("JWT_SECRET", "wallet-api-secret-change-me"),
		StreamKey:       getEnvOrDefault("REDIS_STREAM_KEY", "wallet:balance:updates"),
		StreamMaxLen:    getEnvInt64("REDIS_STREAM_MAXLEN", 10000),
	}

	if cfg.DatabaseDSN == "" {
		cfg.DatabaseDSN = "./one-api.db"
	}

	if cfg.RedisConnString == "" {
		return fmt.Errorf("REDIS_CONN_STRING is required")
	}

	return nil
}

func GetConfig() *Config {
	return cfg
}

func InitDB() error {
	var err error
	dsn := cfg.DatabaseDSN

	switch {
	case strings.HasPrefix(dsn, "postgres://"):
		DB, err = gorm.Open(postgres.New(postgres.Config{
			DSN:                  dsn,
			PreferSimpleProtocol: true,
		}), &gorm.Config{
			PrepareStmt: true,
		})
	case strings.Contains(dsn, "@tcp("):
		DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			PrepareStmt: true,
		})
	default:
		// SQLite
		if !strings.HasSuffix(dsn, ".db") {
			dsn = "./one-api.db"
		}
		dbPath := dsn + "?_busy_timeout=5000"
		DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			PrepareStmt: true,
		})
	}

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移交易表
	if err := DB.AutoMigrate(&Transaction{}); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// 自动迁移订阅表
	if err := DB.AutoMigrate(&Subscription{}); err != nil {
		return fmt.Errorf("failed to migrate subscription table: %w", err)
	}

	// 初始化日志数据库（如果配置了独立的日志数据库）
	if cfg.LogDatabaseDSN != "" {
		if err := initLogDB(); err != nil {
			return err
		}
	} else {
		// 使用主数据库
		LOG_DB = DB
		// 迁移 Log 表
		if err := DB.AutoMigrate(&Log{}); err != nil {
			return fmt.Errorf("failed to migrate log table: %w", err)
		}
	}

	return nil
}

func InitRedis() error {
	opt, err := redis.ParseURL(cfg.RedisConnString)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	if cfg.RedisPassword != "" {
		opt.Password = cfg.RedisPassword
	}

	RDB = redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RDB.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("failed to ping Redis: %w", err)
	}

	return nil
}

func GetDB() *gorm.DB {
	return DB
}

func GetRedis() *redis.Client {
	return RDB
}

func GetLogDB() *gorm.DB {
	return LOG_DB
}

func CloseDB() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		sqlDB.Close()
	}
	if LOG_DB != nil && LOG_DB != DB {
		sqlDB, err := LOG_DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

func CloseRedis() error {
	if RDB != nil {
		return RDB.Close()
	}
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		var result int64
		fmt.Sscanf(value, "%d", &result)
		return result
	}
	return defaultValue
}

// initLogDB 初始化日志数据库
func initLogDB() error {
	var err error
	dsn := cfg.LogDatabaseDSN

	switch {
	case strings.HasPrefix(dsn, "postgres://"):
		LOG_DB, err = gorm.Open(postgres.New(postgres.Config{
			DSN:                  dsn,
			PreferSimpleProtocol: true,
		}), &gorm.Config{
			PrepareStmt: true,
		})
	case strings.Contains(dsn, "@tcp("):
		LOG_DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			PrepareStmt: true,
		})
	default:
		// SQLite
		if !strings.HasSuffix(dsn, ".db") {
			dsn = "./one-api-log.db"
		}
		dbPath := dsn + "?_busy_timeout=5000"
		LOG_DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			PrepareStmt: true,
		})
	}

	if err != nil {
		return fmt.Errorf("failed to connect to log database: %w", err)
	}

	sqlDB, err := LOG_DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get log database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 迁移 Log 表
	if err := LOG_DB.AutoMigrate(&Log{}); err != nil {
		return fmt.Errorf("failed to migrate log table: %w", err)
	}

	return nil
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

// Transaction 交易记录表
type Transaction struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	UserID      int       `gorm:"index;not null" json:"user_id"`
	Username    string    `gorm:"index" json:"username"`
	Type        string    `gorm:"type:varchar(20);not null" json:"type"` // deduct, refund
	Amount      int64     `gorm:"not null" json:"amount"`
	BalanceBefore int64   `gorm:"not null" json:"balance_before"`
	BalanceAfter  int64   `gorm:"not null" json:"balance_after"`
	Description string    `gorm:"type:varchar(500)" json:"description"`
	RefTxnID    *uint     `gorm:"index" json:"ref_txn_id,omitempty"` // 关联的交易ID（用于回退）
	CreatedAt   time.Time `json:"created_at"`
}

// Subscription 订阅域名表
type Subscription struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	WebhookURL  string    `gorm:"type:varchar(500);not null;uniqueIndex" json:"webhook_url"`
	Description string    `gorm:"type:varchar(200)" json:"description"`
	Active      bool      `gorm:"default:true" json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
