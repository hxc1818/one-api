package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/songquanpeng/one-api/wallet-api/config"
)

type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     int    `json:"role"`
	jwt.RegisteredClaims
}

// AuthMiddleware JWT 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		// 移除 "Bearer " 前缀
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		// 解析 token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.GetConfig().JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// 提取 claims
		if claims, ok := token.Claims.(*Claims); ok {
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("role", claims.Role)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminOnly 管理员权限中间件
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "role not found"})
			c.Abort()
			return
		}

		if role.(int) < 10 { // 10 是管理员权限
			c.JSON(http.StatusForbidden, gin.H{"error": "admin permission required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) (int, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	return userID.(int), true
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) (string, bool) {
	username, exists := c.Get("username")
	if !exists {
		return "", false
	}
	return username.(string), true
}
