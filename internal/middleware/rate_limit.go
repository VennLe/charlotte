package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/VennLe/charlotte/pkg/utils"
)

// RateLimiterConfig 限流器配置
type RateLimiterConfig struct {
	RedisClient *redis.Client
	MaxRequests int64
	WindowSize  time.Duration
}

// NewRateLimiter 创建限流中间件
func NewRateLimiter(config RateLimiterConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.RedisClient == nil {
			c.Next()
			return
		}

		// IP 限流
		key := "rate_limit:" + c.ClientIP()

		ctx := c.Request.Context()
		pipe := config.RedisClient.Pipeline()
		incr := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, config.WindowSize)
		_, _ = pipe.Exec(ctx)

		count := incr.Val()
		if count > config.MaxRequests {
			utils.Error(c, http.StatusTooManyRequests, "请求过于频繁")
			c.Abort()
			return
		}

		c.Next()
	}
}

// DefaultRateLimiter 默认限流中间件（每分钟100请求）
func DefaultRateLimiter(redisClient *redis.Client) gin.HandlerFunc {
	return NewRateLimiter(RateLimiterConfig{
		RedisClient: redisClient,
		MaxRequests: 100,
		WindowSize:  time.Minute,
	})
}
