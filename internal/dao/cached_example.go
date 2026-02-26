package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/VennLe/charlotte/internal/model"
)

// CachedUserDAO 带缓存的用户DAO示例
type CachedUserDAO struct {
	*CachedBaseDAO[model.User, uint]
}

// NewCachedUserDAO 创建带缓存的用户DAO
func NewCachedUserDAO(db *gorm.DB, redisClient *redis.Client) *CachedUserDAO {
	cacheConfig := &CacheConfig{
		Enabled:    true,
		TTL:        10 * time.Minute, // 缓存10分钟
		Prefix:     "charlotte",
		NullTTL:    2 * time.Minute,  // 空值缓存2分钟
		MaxSize:    1000,
	}

	return &CachedUserDAO{
		CachedBaseDAO: NewCachedBaseDAO[model.User, uint](db, redisClient, cacheConfig, "user"),
	}
}

// GetUserByEmailWithCache 带缓存的根据邮箱获取用户
func (d *CachedUserDAO) GetUserByEmailWithCache(ctx context.Context, email string) (*model.User, error) {
	return d.GetOneWithCache(ctx, map[string]interface{}{"email": email})
}

// GetActiveUsersWithCache 带缓存的获取活跃用户
func (d *CachedUserDAO) GetActiveUsersWithCache(ctx context.Context) ([]*model.User, error) {
	return d.GetMany(ctx, map[string]interface{}{"status": 1})
}

// CachedExampleUsage 带缓存的使用示例
func CachedExampleUsage() {
	// 假设已经有了数据库连接和Redis连接
	var db *gorm.DB
	var redisClient *redis.Client

	// 创建带缓存的用户DAO
	cachedUserDAO := NewCachedUserDAO(db, redisClient)

	// 1. 带缓存获取用户（防穿透）
	user, err := cachedUserDAO.GetByIDWithCache(context.Background(), 1)
	if err != nil {
		fmt.Printf("获取用户失败: %v\n", err)
	} else {
		fmt.Printf("获取用户成功: %s\n", user.Username)
	}

	// 2. 带缓存根据邮箱获取用户
	user, err = cachedUserDAO.GetUserByEmailWithCache(context.Background(), "test@example.com")
	if err != nil {
		fmt.Printf("获取用户失败: %v\n", err)
	} else {
		fmt.Printf("获取用户成功: %s\n", user.Username)
	}

	// 3. 批量获取用户（防穿透）
	userIDs := []uint{1, 2, 3, 4, 5}
	users, err := cachedUserDAO.BatchGetWithCache(context.Background(), userIDs)
	if err != nil {
		fmt.Printf("批量获取用户失败: %v\n", err)
	} else {
		fmt.Printf("批量获取到 %d 个用户\n", len(users))
	}

	// 4. 带缓存的更新操作
	err = cachedUserDAO.UpdateWithCache(context.Background(), 1, map[string]interface{}{
		"nickname": "新昵称",
		"avatar":   "new_avatar.jpg",
	})
	if err != nil {
		fmt.Printf("更新用户失败: %v\n", err)
	} else {
		fmt.Printf("更新用户成功，缓存已自动失效\n")
	}

	// 5. 带缓存的删除操作
	err = cachedUserDAO.DeleteWithCache(context.Background(), 1)
	if err != nil {
		fmt.Printf("删除用户失败: %v\n", err)
	} else {
		fmt.Printf("删除用户成功，缓存已自动失效\n")
	}

	// 6. 健康检查
	err = cachedUserDAO.HealthCheck(context.Background())
	if err != nil {
		fmt.Printf("健康检查失败: %v\n", err)
	} else {
		fmt.Printf("健康检查通过\n")
	}

	// 7. 获取缓存统计
	stats := cachedUserDAO.GetCacheStats(context.Background())
	fmt.Printf("缓存统计: %+v\n", stats)
}

// 缓存穿透防护示例
func CachePenetrationExample() {
	var db *gorm.DB
	var redisClient *redis.Client
	cachedUserDAO := NewCachedUserDAO(db, redisClient)

	// 模拟缓存穿透攻击场景
	ctx := context.Background()
	
	// 攻击者尝试获取不存在的用户ID
	nonExistentIDs := []uint{99999, 88888, 77777, 66666, 55555}

	fmt.Println("=== 缓存穿透防护测试 ===")
	
	for _, id := range nonExistentIDs {
		// 第一次请求（缓存未命中，会查询数据库）
		user, err := cachedUserDAO.GetByIDWithCache(ctx, id)
		if err != nil {
			fmt.Printf("ID %d: %v\n", id, err)
		} else {
			fmt.Printf("ID %d: 找到用户 %s\n", id, user.Username)
		}

		// 第二次请求（缓存命中，直接返回空值）
		user, err = cachedUserDAO.GetByIDWithCache(ctx, id)
		if err != nil {
			fmt.Printf("ID %d (缓存): %v\n", id, err)
		} else {
			fmt.Printf("ID %d (缓存): 找到用户 %s\n", id, user.Username)
		}
	}

	fmt.Println("=== 批量防穿透测试 ===")
	
	// 批量获取，包含存在和不存在的ID
	mixedIDs := []uint{1, 99999, 2, 88888, 3}
	users, err := cachedUserDAO.BatchGetWithCache(ctx, mixedIDs)
	if err != nil {
		fmt.Printf("批量获取失败: %v\n", err)
	} else {
		fmt.Printf("批量获取结果: %d 个用户\n", len(users))
		for id, user := range users {
			if user != nil {
				fmt.Printf("  ID %d: %s\n", id, user.Username)
			} else {
				fmt.Printf("  ID %d: 不存在\n", id)
			}
		}
	}
}

// MultiDatabaseExample 多数据库支持示例
func MultiDatabaseExample() {
	// 示例展示如何支持多种数据库
	fmt.Println("=== 多数据库支持 ===")
	
	// PostgreSQL配置示例
	fmt.Println("PostgreSQL DSN示例:")
	fmt.Println("host=localhost port=5432 user=postgres password=password dbname=charlotte sslmode=disable")
	
	// MySQL配置示例
	fmt.Println("MySQL DSN示例:")
	fmt.Println("user:password@tcp(localhost:3306)/charlotte?charset=utf8mb4&parseTime=True&loc=Local")
	
	// SQLite配置示例
	fmt.Println("SQLite DSN示例:")
	fmt.Println("./data/charlotte.db")
	
	fmt.Println("注意：实际使用时请参考internal/initialize/gorm.go中的generateDSN函数")
}

// 性能优化建议
func PerformanceTips() {
	fmt.Println("=== 性能优化建议 ===")
	
	tips := []string{
		"1. 合理设置缓存TTL：热点数据可设置较长TTL，冷数据设置较短TTL",
		"2. 空值缓存时间不宜过长：建议1-5分钟，防止缓存污染",
		"3. 批量操作优先：使用BatchGetWithCache减少数据库查询次数",
		"4. 监控缓存命中率：定期检查缓存效果，调整缓存策略",
		"5. 使用连接池：合理配置数据库和Redis连接池参数",
		"6. 避免大Key：单个缓存值不宜过大，可考虑分片存储",
		"7. 设置缓存上限：防止缓存占用过多内存",
		"8. 定期清理：设置缓存清理策略，防止内存泄漏",
	}
	
	for _, tip := range tips {
		fmt.Println(tip)
	}
}