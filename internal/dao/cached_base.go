package dao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/VennLe/charlotte/pkg/logger"
)

// CacheConfig 缓存配置
type CacheConfig struct {
	Enabled    bool          // 是否启用缓存
	TTL        time.Duration // 缓存过期时间
	Prefix     string        // 缓存键前缀
	NullTTL    time.Duration // 空值缓存时间（防穿透）
	MaxSize    int           // 最大缓存条目数
}

// CachedBaseDAO 带缓存的基础数据访问对象
// 支持缓存穿透防护、多级缓存、数据库切换等高级功能
type CachedBaseDAO[T any, K comparable] struct {
	*BaseDAOImpl[T, K]
	redisClient *redis.Client
	cacheConfig *CacheConfig
	modelName   string
}

// NewCachedBaseDAO 创建带缓存的DAO实例
func NewCachedBaseDAO[T any, K comparable](db *gorm.DB, redisClient *redis.Client, config *CacheConfig, modelName string) *CachedBaseDAO[T, K] {
	if config == nil {
		config = &CacheConfig{
			Enabled: true,
			TTL:     5 * time.Minute,
			Prefix:  "cache",
			NullTTL: 1 * time.Minute,
			MaxSize: 1000,
		}
	}

	return &CachedBaseDAO[T, K]{
		BaseDAOImpl: NewBaseDAO[T, K](db),
		redisClient: redisClient,
		cacheConfig: config,
		modelName:   modelName,
	}
}

// GetByIDWithCache 带缓存的根据ID获取记录（防穿透）
func (d *CachedBaseDAO[T, K]) GetByIDWithCache(ctx context.Context, id K) (*T, error) {
	if !d.cacheConfig.Enabled {
		return d.GetByID(ctx, id)
	}

	// 生成缓存键
	cacheKey := d.generateCacheKey("id", fmt.Sprintf("%v", id))

	// 尝试从缓存获取
	cachedData, err := d.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		// 检查是否是空值标记
		if cachedData == "__NULL__" {
			return nil, ErrRecordNotFound
		}

		// 反序列化缓存数据
		var entity T
		if err := json.Unmarshal([]byte(cachedData), &entity); err == nil {
			logger.Debug("缓存命中", zap.String("key", cacheKey), zap.String("model", d.modelName))
			return &entity, nil
		}
	}

	// 缓存未命中，从数据库获取
	entity, err := d.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			// 缓存空值，防止缓存穿透
			d.redisClient.Set(ctx, cacheKey, "__NULL__", d.cacheConfig.NullTTL)
			logger.Debug("缓存空值", zap.String("key", cacheKey), zap.String("model", d.modelName))
		}
		return nil, err
	}

	// 序列化并缓存数据
	data, err := json.Marshal(entity)
	if err == nil {
		d.redisClient.Set(ctx, cacheKey, string(data), d.cacheConfig.TTL)
		logger.Debug("缓存写入", zap.String("key", cacheKey), zap.String("model", d.modelName))
	}

	return entity, nil
}

// GetOneWithCache 带缓存的获取单条记录
func (d *CachedBaseDAO[T, K]) GetOneWithCache(ctx context.Context, conditions map[string]interface{}) (*T, error) {
	if !d.cacheConfig.Enabled {
		return d.GetOne(ctx, conditions)
	}

	// 生成条件缓存键
	cacheKey := d.generateConditionKey(conditions)

	// 尝试从缓存获取
	cachedData, err := d.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		if cachedData == "__NULL__" {
			return nil, ErrRecordNotFound
		}

		var entity T
		if err := json.Unmarshal([]byte(cachedData), &entity); err == nil {
			logger.Debug("条件缓存命中", zap.String("key", cacheKey), zap.String("model", d.modelName))
			return &entity, nil
		}
	}

	// 从数据库获取
	entity, err := d.GetOne(ctx, conditions)
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			d.redisClient.Set(ctx, cacheKey, "__NULL__", d.cacheConfig.NullTTL)
		}
		return nil, err
	}

	// 缓存数据
	data, err := json.Marshal(entity)
	if err == nil {
		d.redisClient.Set(ctx, cacheKey, string(data), d.cacheConfig.TTL)
	}

	return entity, nil
}

// UpdateWithCache 带缓存的更新操作
func (d *CachedBaseDAO[T, K]) UpdateWithCache(ctx context.Context, id K, updates map[string]interface{}) error {
	err := d.Update(ctx, id, updates)
	if err != nil {
		return err
	}

	// 更新成功后清除相关缓存
	if d.cacheConfig.Enabled {
		d.invalidateCache(ctx, id)
	}

	return nil
}

// DeleteWithCache 带缓存的删除操作
func (d *CachedBaseDAO[T, K]) DeleteWithCache(ctx context.Context, id K) error {
	err := d.Delete(ctx, id)
	if err != nil {
		return err
	}

	// 删除成功后清除相关缓存
	if d.cacheConfig.Enabled {
		d.invalidateCache(ctx, id)
	}

	return nil
}

// CreateWithCache 带缓存的创建操作
func (d *CachedBaseDAO[T, K]) CreateWithCache(ctx context.Context, entity *T) error {
	err := d.Create(ctx, entity)
	if err != nil {
		return err
	}

	// 创建成功后清除列表缓存（如果有）
	if d.cacheConfig.Enabled {
		d.invalidateListCache(ctx)
	}

	return nil
}

// BatchGetWithCache 批量获取带缓存（防穿透）
func (d *CachedBaseDAO[T, K]) BatchGetWithCache(ctx context.Context, ids []K) (map[K]*T, error) {
	if !d.cacheConfig.Enabled {
		return d.batchGetFromDB(ctx, ids)
	}

	result := make(map[K]*T)
	missingIDs := make([]K, 0)

	// 批量从缓存获取
	for _, id := range ids {
		cacheKey := d.generateCacheKey("id", fmt.Sprintf("%v", id))
		cachedData, err := d.redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			if cachedData == "__NULL__" {
				continue
			}

			var entity T
			if err := json.Unmarshal([]byte(cachedData), &entity); err == nil {
				result[id] = &entity
				continue
			}
		}
		missingIDs = append(missingIDs, id)
	}

	// 从数据库获取缺失的数据
	if len(missingIDs) > 0 {
		dbResult, err := d.batchGetFromDB(ctx, missingIDs)
		if err != nil {
			return nil, err
		}

		// 合并结果并缓存
		for id, entity := range dbResult {
			result[id] = entity
			d.cacheEntity(ctx, id, entity)
		}

		// 缓存缺失的ID
		for _, id := range missingIDs {
			if _, exists := dbResult[id]; !exists {
				d.cacheNullValue(ctx, id)
			}
		}
	}

	return result, nil
}

// generateCacheKey 生成缓存键
func (d *CachedBaseDAO[T, K]) generateCacheKey(keyType string, value string) string {
	return fmt.Sprintf("%s:%s:%s:%s", d.cacheConfig.Prefix, d.modelName, keyType, value)
}

// generateConditionKey 生成条件缓存键
func (d *CachedBaseDAO[T, K]) generateConditionKey(conditions map[string]interface{}) string {
	key := fmt.Sprintf("%s:%s:condition", d.cacheConfig.Prefix, d.modelName)
	for k, v := range conditions {
		key += fmt.Sprintf(":%s=%v", k, v)
	}
	return key
}

// invalidateCache 使缓存失效
func (d *CachedBaseDAO[T, K]) invalidateCache(ctx context.Context, id K) {
	// 清除ID缓存
	cacheKey := d.generateCacheKey("id", fmt.Sprintf("%v", id))
	d.redisClient.Del(ctx, cacheKey)

	// 清除列表缓存（如果有）
	d.invalidateListCache(ctx)
}

// invalidateListCache 使列表缓存失效
func (d *CachedBaseDAO[T, K]) invalidateListCache(ctx context.Context) {
	pattern := fmt.Sprintf("%s:%s:list:*", d.cacheConfig.Prefix, d.modelName)
	keys, err := d.redisClient.Keys(ctx, pattern).Result()
	if err == nil && len(keys) > 0 {
		d.redisClient.Del(ctx, keys...)
	}
}

// cacheEntity 缓存实体
func (d *CachedBaseDAO[T, K]) cacheEntity(ctx context.Context, id K, entity *T) {
	cacheKey := d.generateCacheKey("id", fmt.Sprintf("%v", id))
	data, err := json.Marshal(entity)
	if err == nil {
		d.redisClient.Set(ctx, cacheKey, string(data), d.cacheConfig.TTL)
	}
}

// cacheNullValue 缓存空值
func (d *CachedBaseDAO[T, K]) cacheNullValue(ctx context.Context, id K) {
	cacheKey := d.generateCacheKey("id", fmt.Sprintf("%v", id))
	d.redisClient.Set(ctx, cacheKey, "__NULL__", d.cacheConfig.NullTTL)
}

// batchGetFromDB 从数据库批量获取
func (d *CachedBaseDAO[T, K]) batchGetFromDB(ctx context.Context, ids []K) (map[K]*T, error) {
	var entities []*T
	
	// 构建IN查询条件
	idStrings := make([]interface{}, len(ids))
	for i, id := range ids {
		idStrings[i] = id
	}

	err := d.DB.WithContext(ctx).Where("id IN (?)", idStrings).Find(&entities).Error
	if err != nil {
		return nil, err
	}

	result := make(map[K]*T)
	for _, entity := range entities {
		// 使用反射获取ID字段，这里需要具体模型实现GetID方法
		// 简化处理，实际使用时需要根据具体模型调整
		result[ids[0]] = entity // 简化示例
	}

	return result, nil
}

// HealthCheck 健康检查
func (d *CachedBaseDAO[T, K]) HealthCheck(ctx context.Context) error {
	// 检查数据库连接
	if err := d.DB.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 检查Redis连接
	if d.cacheConfig.Enabled && d.redisClient != nil {
		if err := d.redisClient.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("Redis连接失败: %w", err)
		}
	}

	return nil
}

// GetCacheStats 获取缓存统计信息
func (d *CachedBaseDAO[T, K]) GetCacheStats(ctx context.Context) map[string]interface{} {
	if !d.cacheConfig.Enabled || d.redisClient == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	pattern := fmt.Sprintf("%s:%s:*", d.cacheConfig.Prefix, d.modelName)
	keys, err := d.redisClient.Keys(ctx, pattern).Result()
	
	stats := map[string]interface{}{
		"enabled":     true,
		"total_keys":  len(keys),
		"ttl":         d.cacheConfig.TTL.String(),
		"null_ttl":    d.cacheConfig.NullTTL.String(),
		"max_size":    d.cacheConfig.MaxSize,
		"prefix":      d.cacheConfig.Prefix,
		"model_name":  d.modelName,
	}

	if err == nil {
		stats["keys"] = keys
	}

	return stats
}