package dao

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// Common errors
var (
	ErrRecordNotFound = errors.New("记录不存在")
	ErrRecordExists   = errors.New("记录已存在")
)

// QueryOptions 查询选项
type QueryOptions struct {
	Page     int                    // 页码
	Size     int                    // 每页大小
	Keyword  string                 // 关键词搜索
	Filters  map[string]interface{} // 精确过滤条件
	OrderBy  string                 // 排序字段
	OrderDir string                 // 排序方向 (asc/desc)
	Preloads []string              // 预加载关联
}

// BaseDAO 基础数据访问接口
// T 为具体的模型类型，K为模型主键类型
//go:generate mockgen -source=base.go -destination=mocks/base_mock.go -package=mocks
type BaseDAO[T any, K comparable] interface {
	// Create 创建记录
	Create(ctx context.Context, entity *T) error

	// CreateBatch 批量创建记录
	CreateBatch(ctx context.Context, entities []*T) error

	// GetByID 根据主键获取记录
	GetByID(ctx context.Context, id K) (*T, error)

	// GetOne 获取单条记录（带条件）
	GetOne(ctx context.Context, conditions map[string]interface{}) (*T, error)

	// GetMany 获取多条记录（带条件）
	GetMany(ctx context.Context, conditions map[string]interface{}) ([]*T, error)

	// List 分页列表查询
	List(ctx context.Context, options *QueryOptions) ([]*T, int64, error)

	// Update 更新记录
	Update(ctx context.Context, id K, updates map[string]interface{}) error

	// UpdateWhere 条件更新
	UpdateWhere(ctx context.Context, conditions map[string]interface{}, updates map[string]interface{}) error

	// Delete 删除记录（软删除）
	Delete(ctx context.Context, id K) error

	// DeleteWhere 条件删除
	DeleteWhere(ctx context.Context, conditions map[string]interface{}) error

	// HardDelete 硬删除
	HardDelete(ctx context.Context, id K) error

	// HardDeleteWhere 条件硬删除
	HardDeleteWhere(ctx context.Context, conditions map[string]interface{}) error

	// Count 统计记录数量
	Count(ctx context.Context, conditions map[string]interface{}) (int64, error)

	// Exists 检查记录是否存在
	Exists(ctx context.Context, conditions map[string]interface{}) (bool, error)

	// Transaction 执行事务操作
	Transaction(ctx context.Context, fn func(txDAO BaseDAO[T, K]) error) error
}

// BaseDAOImpl 基础数据访问实现
// 这是一个基础实现，具体模型可以继承并扩展
//go:generate mockgen -source=base.go -destination=mocks/base_mock.go -package=mocks
type BaseDAOImpl[T any, K comparable] struct {
	DB *gorm.DB
}

// NewBaseDAO 创建基础DAO实例
func NewBaseDAO[T any, K comparable](db *gorm.DB) *BaseDAOImpl[T, K] {
	return &BaseDAOImpl[T, K]{DB: db}
}

// Create 创建记录
func (d *BaseDAOImpl[T, K]) Create(ctx context.Context, entity *T) error {
	return d.DB.WithContext(ctx).Create(entity).Error
}

// CreateBatch 批量创建记录
func (d *BaseDAOImpl[T, K]) CreateBatch(ctx context.Context, entities []*T) error {
	return d.DB.WithContext(ctx).CreateInBatches(entities, 100).Error
}

// GetByID 根据主键获取记录
func (d *BaseDAOImpl[T, K]) GetByID(ctx context.Context, id K) (*T, error) {
	var entity T
	err := d.DB.WithContext(ctx).First(&entity, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRecordNotFound
	}
	return &entity, err
}

// GetOne 获取单条记录（带条件）
func (d *BaseDAOImpl[T, K]) GetOne(ctx context.Context, conditions map[string]interface{}) (*T, error) {
	var entity T
	query := d.DB.WithContext(ctx).Model(&entity)
	
	for field, value := range conditions {
		query = query.Where(field, value)
	}
	
	err := query.First(&entity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRecordNotFound
	}
	return &entity, err
}

// GetMany 获取多条记录（带条件）
func (d *BaseDAOImpl[T, K]) GetMany(ctx context.Context, conditions map[string]interface{}) ([]*T, error) {
	var entities []*T
	query := d.DB.WithContext(ctx)
	
	for field, value := range conditions {
		query = query.Where(field, value)
	}
	
	err := query.Find(&entities).Error
	return entities, err
}

// List 分页列表查询
func (d *BaseDAOImpl[T, K]) List(ctx context.Context, options *QueryOptions) ([]*T, int64, error) {
	var entities []*T
	var total int64

	query := d.DB.WithContext(ctx).Model(new(T))

	// 应用过滤条件
	if options != nil {
		// 关键词搜索
		if options.Keyword != "" {
			// 具体实现由子类重写，这里提供框架
		}

		// 精确过滤
		if options.Filters != nil {
			for field, value := range options.Filters {
				query = query.Where(field, value)
			}
		}

		// 预加载关联
		for _, preload := range options.Preloads {
			query = query.Preload(preload)
		}

		// 统计总数
		if err := query.Count(&total).Error; err != nil {
			return nil, 0, err
		}

		// 分页
		if options.Page > 0 && options.Size > 0 {
			query = query.Offset((options.Page - 1) * options.Size).Limit(options.Size)
		}

		// 排序
		if options.OrderBy != "" {
			order := options.OrderBy
			if options.OrderDir != "" {
				order = order + " " + options.OrderDir
			}
			query = query.Order(order)
		}
	}

	err := query.Find(&entities).Error
	return entities, total, err
}

// Update 更新记录
func (d *BaseDAOImpl[T, K]) Update(ctx context.Context, id K, updates map[string]interface{}) error {
	result := d.DB.WithContext(ctx).Model(new(T)).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

// UpdateWhere 条件更新
func (d *BaseDAOImpl[T, K]) UpdateWhere(ctx context.Context, conditions map[string]interface{}, updates map[string]interface{}) error {
	query := d.DB.WithContext(ctx).Model(new(T))
	
	for field, value := range conditions {
		query = query.Where(field, value)
	}
	
	return query.Updates(updates).Error
}

// Delete 删除记录（软删除）
func (d *BaseDAOImpl[T, K]) Delete(ctx context.Context, id K) error {
	result := d.DB.WithContext(ctx).Delete(new(T), id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

// DeleteWhere 条件删除
func (d *BaseDAOImpl[T, K]) DeleteWhere(ctx context.Context, conditions map[string]interface{}) error {
	query := d.DB.WithContext(ctx).Model(new(T))
	
	for field, value := range conditions {
		query = query.Where(field, value)
	}
	
	return query.Delete(new(T)).Error
}

// HardDelete 硬删除
func (d *BaseDAOImpl[T, K]) HardDelete(ctx context.Context, id K) error {
	result := d.DB.WithContext(ctx).Unscoped().Delete(new(T), id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

// HardDeleteWhere 条件硬删除
func (d *BaseDAOImpl[T, K]) HardDeleteWhere(ctx context.Context, conditions map[string]interface{}) error {
	query := d.DB.WithContext(ctx).Unscoped().Model(new(T))
	
	for field, value := range conditions {
		query = query.Where(field, value)
	}
	
	return query.Delete(new(T)).Error
}

// Count 统计记录数量
func (d *BaseDAOImpl[T, K]) Count(ctx context.Context, conditions map[string]interface{}) (int64, error) {
	var count int64
	query := d.DB.WithContext(ctx).Model(new(T))
	
	for field, value := range conditions {
		query = query.Where(field, value)
	}
	
	err := query.Count(&count).Error
	return count, err
}

// Exists 检查记录是否存在
func (d *BaseDAOImpl[T, K]) Exists(ctx context.Context, conditions map[string]interface{}) (bool, error) {
	count, err := d.Count(ctx, conditions)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Transaction 执行事务操作
func (d *BaseDAOImpl[T, K]) Transaction(ctx context.Context, fn func(txDAO BaseDAO[T, K]) error) error {
	return d.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txDAO := NewBaseDAO[T, K](tx)
		return fn(txDAO)
	})
}