package initialize

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/VennLe/charlotte/internal/config"
	"github.com/VennLe/charlotte/pkg/logger"
)

var Redis *redis.Client

func InitRedis() error {
	logger.Debug("开始初始化Redis连接")
	cfg := config.Global.Redis

	logger.Debug("Redis配置", 
		zap.String("host", cfg.Host), 
		zap.String("port", cfg.Port),
		zap.String("mode", config.Global.Server.Mode))

	// 在开发模式下，如果Redis连接失败，直接返回nil
	if config.Global.Server.Mode == "debug" {
		// 使用更短的超时时间快速检测连接
		testConn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", cfg.Host, cfg.Port), 1*time.Second)
		if err != nil {
			logger.Warn("Redis 连接失败，将以无缓存模式运行", zap.Error(err))
			Redis = nil
			return nil
		}
		testConn.Close()
	}

	// 如果连接成功或生产模式，创建正式连接
	Redis = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
		MaxRetries: 0, // 禁用自动重试
		PoolTimeout: 2 * time.Second,
		// 禁用连接池的自动重连
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		// 禁用身份信息
		DisableIdentity: true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := Redis.Ping(ctx).Err(); err != nil {
		// 生产模式下返回错误
		logger.Debug("生产模式下返回错误，阻止启动")
		return fmt.Errorf("连接 Redis 失败: %w", err)
	}

	logger.Info("Redis 连接成功",
		zap.String("addr", Redis.Options().Addr),
		zap.Int("pool_size", Redis.Options().PoolSize))
	return nil
}
