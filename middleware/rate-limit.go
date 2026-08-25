package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
)

var timeFormat = "2006-01-02T15:04:05.000Z"

var inMemoryRateLimiter common.InMemoryRateLimiter

func redisRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
	ctx := context.Background()
	rdb := common.RDB
	key := "rateLimit:" + mark + c.ClientIP()
	listLength, err := rdb.LLen(ctx, key).Result()
	if err != nil {
		fmt.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		c.Abort()
		return
	}
	if listLength < int64(maxRequestNum) {
		rdb.LPush(ctx, key, time.Now().Format(timeFormat))
		rdb.Expire(ctx, key, config.RateLimitKeyExpirationDuration)
	} else {
		oldTimeStr, _ := rdb.LIndex(ctx, key, -1).Result()
		oldTime, err := time.Parse(timeFormat, oldTimeStr)
		if err != nil {
			fmt.Println(err)
			c.Status(http.StatusInternalServerError)
			c.Abort()
			return
		}
		nowTimeStr := time.Now().Format(timeFormat)
		nowTime, err := time.Parse(timeFormat, nowTimeStr)
		if err != nil {
			fmt.Println(err)
			c.Status(http.StatusInternalServerError)
			c.Abort()
			return
		}
		// time.Since will return negative number!
		// See: https://stackoverflow.com/questions/50970900/why-is-time-since-returning-negative-durations-on-windows
		if int64(nowTime.Sub(oldTime).Seconds()) < duration {
			rdb.Expire(ctx, key, config.RateLimitKeyExpirationDuration)
			c.Status(http.StatusTooManyRequests)
			c.Abort()
			return
		} else {
			rdb.LPush(ctx, key, time.Now().Format(timeFormat))
			rdb.LTrim(ctx, key, 0, int64(maxRequestNum-1))
			rdb.Expire(ctx, key, config.RateLimitKeyExpirationDuration)
		}
	}
}

func memoryRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
	key := mark + c.ClientIP()
	if !inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
		c.Status(http.StatusTooManyRequests)
		c.Abort()
		return
	}
}

func rateLimitFactory(maxRequestNum int, duration int64, mark string) func(c *gin.Context) {
	if maxRequestNum == 0 || config.DebugEnabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}
	if common.RedisEnabled {
		return func(c *gin.Context) {
			redisRateLimiter(c, maxRequestNum, duration, mark)
		}
	} else {
		// It's safe to call multi times.
		inMemoryRateLimiter.Init(config.RateLimitKeyExpirationDuration)
		return func(c *gin.Context) {
			memoryRateLimiter(c, maxRequestNum, duration, mark)
		}
	}
}

func GlobalWebRateLimit() func(c *gin.Context) {
	return rateLimitFactory(config.GlobalWebRateLimitNum, config.GlobalWebRateLimitDuration, "GW")
}

func GlobalAPIRateLimit() func(c *gin.Context) {
	return rateLimitFactory(config.GlobalApiRateLimitNum, config.GlobalApiRateLimitDuration, "GA")
}

func CriticalRateLimit() func(c *gin.Context) {
	return rateLimitFactory(config.CriticalRateLimitNum, config.CriticalRateLimitDuration, "CT")
}

func DownloadRateLimit() func(c *gin.Context) {
	return rateLimitFactory(config.DownloadRateLimitNum, config.DownloadRateLimitDuration, "DW")
}

func UploadRateLimit() func(c *gin.Context) {
	return rateLimitFactory(config.UploadRateLimitNum, config.UploadRateLimitDuration, "UP")
}

func UserRateLimit() func(c *gin.Context) {
	return func(c *gin.Context) {
		ctx := context.Background()
		userId := c.GetInt(ctxkey.Id)
		if userId == 0 {
			c.Next()
			return
		}

		// Get user's rate limits
		rpm, rpd, rpw, err := model.GetUserRateLimits(userId)
		if err != nil {
			// If error getting limits, just continue
			c.Next()
			return
		}

		// Check RPM (requests per minute)
		if rpm > 0 {
			if !checkUserRateLimit(ctx, userId, rpm, 60, "RPM", c) {
				return
			}
		}

		// Check RPD (requests per day)
		if rpd > 0 {
			if !checkUserRateLimit(ctx, userId, rpd, 86400, "RPD", c) {
				return
			}
		}

		// Check RPW (requests per week)
		if rpw > 0 {
			if !checkUserRateLimit(ctx, userId, rpw, 604800, "RPW", c) {
				return
			}
		}

		c.Next()
	}
}

func checkUserRateLimit(ctx context.Context, userId int, maxRequestNum int, duration int64, limitType string, c *gin.Context) bool {
	key := fmt.Sprintf("userRateLimit:%s:%d", limitType, userId)

	if common.RedisEnabled {
		rdb := common.RDB
		listLength, err := rdb.LLen(ctx, key).Result()
		if err != nil {
			// On error, allow the request
			return true
		}

		if listLength < int64(maxRequestNum) {
			rdb.LPush(ctx, key, time.Now().Format(timeFormat))
			rdb.Expire(ctx, key, time.Duration(duration)*time.Second)
			return true
		} else {
			oldTimeStr, _ := rdb.LIndex(ctx, key, -1).Result()
			oldTime, err := time.Parse(timeFormat, oldTimeStr)
			if err != nil {
				return true
			}
			nowTime := time.Now()
			if int64(nowTime.Sub(oldTime).Seconds()) < duration {
				rdb.Expire(ctx, key, time.Duration(duration)*time.Second)
				abortWithMessage(c, http.StatusTooManyRequests, "上游渠道可能还有配额，但是我们无法分配")
				return false
			} else {
				rdb.LPush(ctx, key, time.Now().Format(timeFormat))
				rdb.LTrim(ctx, key, 0, int64(maxRequestNum-1))
				rdb.Expire(ctx, key, time.Duration(duration)*time.Second)
				return true
			}
		}
	} else {
		// Use in-memory rate limiter
		inMemoryRateLimiter.Init(config.RateLimitKeyExpirationDuration)
		if !inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
			abortWithMessage(c, http.StatusTooManyRequests, "上游渠道可能还有配额，但是我们无法分配")
			return false
		}
		return true
	}
}
