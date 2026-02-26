# 通用数据库操作接口设计

## 概述

本项目实现了一个通用的数据库操作接口，封装了所有常见的增删改查操作，提供了高度可扩展的DAO层架构。

## 设计理念

### 1. 接口统一化
- 所有DAO都实现相同的`BaseDAO`接口
- 提供一致的API调用方式
- 便于维护和扩展

### 2. 泛型支持
- 使用Go泛型确保类型安全
- 支持任意模型类型和主键类型
- 编译时类型检查

### 3. 分层架构
```
BaseDAO (接口层)
    ↓
BaseDAOImpl (实现层)
    ↓
具体DAO (业务层)
```

## 核心组件

### BaseDAO 接口

定义了所有基础数据库操作：

```go
type BaseDAO[T any, K comparable] interface {
    // CRUD操作
    Create(ctx context.Context, entity *T) error
    GetByID(ctx context.Context, id K) (*T, error)
    Update(ctx context.Context, id K, updates map[string]interface{}) error
    Delete(ctx context.Context, id K) error
    
    // 批量操作
    CreateBatch(ctx context.Context, entities []*T) error
    
    // 条件查询
    GetOne(ctx context.Context, conditions map[string]interface{}) (*T, error)
    GetMany(ctx context.Context, conditions map[string]interface{}) ([]*T, error)
    
    // 分页查询
    List(ctx context.Context, options *QueryOptions) ([]*T, int64, error)
    
    // 条件操作
    UpdateWhere(ctx context.Context, conditions map[string]interface{}, updates map[string]interface{}) error
    DeleteWhere(ctx context.Context, conditions map[string]interface{}) error
    
    // 硬删除
    HardDelete(ctx context.Context, id K) error
    HardDeleteWhere(ctx context.Context, conditions map[string]interface{}) error
    
    // 统计操作
    Count(ctx context.Context, conditions map[string]interface{}) (int64, error)
    Exists(ctx context.Context, conditions map[string]interface{}) (bool, error)
    
    // 事务支持
    Transaction(ctx context.Context, fn func(txDAO BaseDAO[T, K]) error) error
}
```

### QueryOptions 结构体

支持复杂的查询选项：

```go
type QueryOptions struct {
    Page     int                    // 页码
    Size     int                    // 每页大小
    Keyword  string                 // 关键词搜索
    Filters  map[string]interface{} // 精确过滤条件
    OrderBy  string                 // 排序字段
    OrderDir string                 // 排序方向
    Preloads []string              // 预加载关联
}
```

## 使用方法

### 1. 创建新的DAO

```go
// 定义模型
type Product struct {
    ID    uint   `gorm:"primarykey"`
    Name  string `gorm:"size:100"`
    Price float64
}

// 创建DAO
type ProductDAO struct {
    *BaseDAOImpl[Product, uint]
}

func NewProductDAO(db *gorm.DB) *ProductDAO {
    return &ProductDAO{
        BaseDAOImpl: NewBaseDAO[Product, uint](db),
    }
}

// 添加自定义方法
func (d *ProductDAO) GetByCategory(ctx context.Context, category string) ([]*Product, error) {
    return d.GetMany(ctx, map[string]interface{}{"category": category})
}
```

### 2. 使用基础操作

```go
// 创建
product := &Product{Name: "iPhone", Price: 999.99}
err := productDAO.Create(ctx, product)

// 查询
product, err := productDAO.GetByID(ctx, 1)

// 更新
err = productDAO.Update(ctx, 1, map[string]interface{}{"price": 899.99})

// 删除
err = productDAO.Delete(ctx, 1)
```

### 3. 复杂查询

```go
// 分页查询
options := &QueryOptions{
    Page:    1,
    Size:    10,
    Keyword: "phone",
    Filters: map[string]interface{}{"status": "active"},
    OrderBy: "price",
    OrderDir: "desc",
}
products, total, err := productDAO.List(ctx, options)

// 条件查询
products, err := productDAO.GetMany(ctx, map[string]interface{}{
    "price > ?": 500,
    "category": "electronics",
})
```

### 4. 事务操作

```go
err := productDAO.Transaction(ctx, func(txDAO BaseDAO[Product, uint]) error {
    // 创建产品
    product := &Product{Name: "New Product", Price: 100}
    if err := txDAO.Create(ctx, product); err != nil {
        return err
    }
    
    // 更新库存
    if err := txDAO.Update(ctx, product.ID, map[string]interface{}{"stock": 50}); err != nil {
        return err
    }
    
    return nil
})
```

## 优势

### 1. 代码复用性
- 所有基础CRUD操作无需重复实现
- 统一错误处理机制
- 一致的API设计

### 2. 类型安全性
- 编译时类型检查
- 避免运行时类型错误
- 更好的IDE支持

### 3. 可维护性
- 集中管理数据库操作逻辑
- 易于添加新功能
- 便于单元测试

### 4. 扩展性
- 支持自定义查询方法
- 可以重写基础方法添加业务逻辑
- 支持复杂的查询场景

## 最佳实践

### 1. 错误处理
```go
// 使用预定义的错误类型
if errors.Is(err, dao.ErrRecordNotFound) {
    return nil, errors.New("记录不存在")
}
```

### 2. 事务管理
```go
// 使用Transaction方法确保事务一致性
err := dao.Transaction(ctx, func(txDAO BaseDAO[T, K]) error {
    // 多个数据库操作
    return nil
})
```

### 3. 查询优化
```go
// 使用QueryOptions进行复杂查询
options := &QueryOptions{
    Preloads: []string{"Category", "Tags"}, // 预加载关联
    Filters: map[string]interface{}{
        "status": "active",
        "created_at > ?": "2023-01-01",
    },
}
```

## 扩展指南

### 添加新的基础方法
1. 在`BaseDAO`接口中添加方法定义
2. 在`BaseDAOImpl`中实现该方法
3. 所有继承的DAO自动获得该方法

### 自定义业务逻辑
1. 在具体DAO中重写基础方法
2. 添加业务特定的验证逻辑
3. 保持接口一致性

## 示例

参考 `example.go` 文件查看完整的使用示例。