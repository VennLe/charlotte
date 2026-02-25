package initialize

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/VennLe/charlotte/internal/config"
	"github.com/VennLe/charlotte/pkg/logger"
)

var Redis *redis.Client

func InitRedis() error {
	cfg := config.Global.Redis

	// 尝试连接 Redis
	Redis = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
		MaxRetries: 0, // 禁用自动重试
		PoolTimeout: 2 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := Redis.Ping(ctx).Err(); err != nil {
		// 开发模式下只警告，不阻止启动
		if config.Global.Server.Mode == "debug" {
			logger.Warn("Redis 连接失败，将以无缓存模式运行", zap.Error(err))
			Redis = nil // 清空连接，避免后续使用
			return nil
		}
		return fmt.Errorf("连接 Redis 失败: %w", err)
	}

	logger.Info("Redis 连接成功",
		zap.String("addr", Redis.Options().Addr),
		zap.Int("pool_size", Redis.Options().PoolSize))
	return nil
}
