package service

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// HealthChecker 健康检查接口
type HealthChecker struct {
	db       *gorm.DB
	redis    *redis.Client
	producer sarama.SyncProducer
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(db *gorm.DB, redis *redis.Client, producer sarama.SyncProducer) *HealthChecker {
	return &HealthChecker{
		db:       db,
		redis:    redis,
		producer: producer,
	}
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp int64                  `json:"timestamp"`
	Service   string                 `json:"service"`
	Version   string                 `json:"version"`
	Checks    map[string]interface{} `json:"checks"`
}

// Check 执行健康检查
func (h *HealthChecker) Check(ctx context.Context) *HealthStatus {
	status := &HealthStatus{
		Status:    "ok",
		Timestamp: time.Now().Unix(),
		Service:   "enterprise-api",
		Version:   "1.0.0",
		Checks:    make(map[string]interface{}),
	}

	// 检查数据库
	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err == nil && sqlDB.Ping() == nil {
			status.Checks["database"] = "connected"
		} else {
			status.Checks["database"] = "disconnected"
			status.Status = "degraded"
		}
	} else {
		status.Checks["database"] = "not_initialized"
	}

	// 检查 Redis
	if h.redis != nil {
		if err := h.redis.Ping(ctx).Err(); err == nil {
			status.Checks["redis"] = "connected"
		} else {
			status.Checks["redis"] = "disconnected"
			status.Status = "degraded"
		}
	} else {
		status.Checks["redis"] = "not_initialized"
	}

	// 检查 Kafka
	if h.producer != nil {
		status.Checks["kafka"] = "connected"
	} else {
		status.Checks["kafka"] = "not_initialized"
	}

	return status
}
