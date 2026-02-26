package dao

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/VennLe/charlotte/internal/model"
)

// ExampleDAO 示例DAO，展示如何使用通用接口
// 使用现有的User模型作为示例

type UserExampleDAO struct {
	*BaseDAOImpl[model.User, uint]
}

// NewUserExampleDAO 创建用户示例DAO实例
func NewUserExampleDAO(db *gorm.DB) *UserExampleDAO {
	return &UserExampleDAO{
		BaseDAOImpl: NewBaseDAO[model.User, uint](db),
	}
}

// GetByRole 根据角色获取用户（自定义方法）
func (d *UserExampleDAO) GetByRole(ctx context.Context, role string) ([]*model.User, error) {
	return d.GetMany(ctx, map[string]interface{}{"role": role})
}

// GetActiveUsers 获取活跃用户（自定义方法）
func (d *UserExampleDAO) GetActiveUsers(ctx context.Context) ([]*model.User, error) {
	return d.GetMany(ctx, map[string]interface{}{"status": 1})
}

// UpdateRole 更新用户角色（自定义方法）
func (d *UserExampleDAO) UpdateRole(ctx context.Context, id uint, role string) error {
	return d.Update(ctx, id, map[string]interface{}{"role": role})
}

// ExampleUsage 示例使用方法
func ExampleUsage() {
	// 假设我们已经有了数据库连接
	var db *gorm.DB

	// 创建UserDAO实例
	userDAO := NewUserDAO(db)

	// 1. 创建用户
	user := &model.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	err := userDAO.Create(context.Background(), user)
	if err != nil {
		fmt.Printf("创建用户失败: %v\n", err)
	}

	// 2. 根据ID获取用户
	user, err = userDAO.GetByID(context.Background(), 1)
	if err != nil {
		fmt.Printf("获取用户失败: %v\n", err)
	}

	// 3. 根据用户名获取用户
	user, err = userDAO.GetByUsername(context.Background(), "testuser")
	if err != nil {
		fmt.Printf("获取用户失败: %v\n", err)
	}

	// 4. 分页查询用户列表
	options := &QueryOptions{
		Page:    1,
		Size:    10,
		Keyword: "test",
		OrderBy: "created_at",
		OrderDir: "desc",
	}
	users, total, err := userDAO.List(context.Background(), options)
	if err != nil {
		fmt.Printf("查询用户列表失败: %v\n", err)
	}
	fmt.Printf("查询到 %d 条记录，共 %d 条\n", len(users), total)

	// 5. 更新用户信息
	err = userDAO.Update(context.Background(), 1, map[string]interface{}{
		"nickname": "测试用户",
		"avatar":   "avatar.jpg",
	})
	if err != nil {
		fmt.Printf("更新用户失败: %v\n", err)
	}

	// 6. 删除用户
	err = userDAO.Delete(context.Background(), 1)
	if err != nil {
		fmt.Printf("删除用户失败: %v\n", err)
	}

	// 7. 事务操作
	err = userDAO.Transaction(context.Background(), func(txDAO BaseDAO[model.User, uint]) error {
		// 在事务中执行多个操作
		user := &model.User{
			Username: "transaction_user",
			Email:    "tx@example.com",
			Password: "password123",
		}
		
		if err := txDAO.Create(context.Background(), user); err != nil {
			return err
		}

		// 更新用户信息
		if err := txDAO.Update(context.Background(), user.ID, map[string]interface{}{
			"nickname": "事务用户",
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		fmt.Printf("事务操作失败: %v\n", err)
	}

	// 8. 使用示例DAO
	exampleDAO := NewUserExampleDAO(db)
	
	// 根据角色获取用户
	adminUsers, err := exampleDAO.GetByRole(context.Background(), "admin")
	if err != nil {
		fmt.Printf("获取管理员用户失败: %v\n", err)
	}
	fmt.Printf("找到 %d 个管理员用户\n", len(adminUsers))

	// 更新用户角色
	err = exampleDAO.UpdateRole(context.Background(), 1, "admin")
	if err != nil {
		fmt.Printf("更新用户角色失败: %v\n", err)
	}
}

// 扩展接口的优势：
// 1. 代码复用：所有基础CRUD操作都已在BaseDAO中实现
// 2. 类型安全：使用泛型确保类型正确性
// 3. 易于测试：可以轻松创建Mock对象进行单元测试
// 4. 一致性：所有DAO都遵循相同的接口规范
// 5. 可扩展性：可以轻松添加新的自定义方法

// 当需要添加新的模型时，只需要：
// 1. 创建模型结构体
// 2. 创建对应的DAO结构体，继承BaseDAOImpl
// 3. 添加必要的自定义方法
// 4. 享受所有基础CRUD操作的支持