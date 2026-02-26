package dao

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/VennLe/charlotte/internal/model"
	"github.com/VennLe/charlotte/pkg/logger"
)

var (
	ErrUserNotFound = errors.New("用户不存在")
	ErrUserExists   = errors.New("用户已存在")
)

// UserDAO 用户数据访问对象
// 继承基础DAO接口，同时保持原有的特殊方法
type UserDAO struct {
	*BaseDAOImpl[model.User, uint]
}

// NewUserDAO 创建 DAO 实例
func NewUserDAO(db *gorm.DB) *UserDAO {
	return &UserDAO{
		BaseDAOImpl: NewBaseDAO[model.User, uint](db),
	}
}

// Create 创建用户（重写基础方法，添加业务逻辑）
func (d *UserDAO) Create(ctx context.Context, user *model.User) error {
	// 检查用户名是否存在
	exists, err := d.Exists(ctx, map[string]interface{}{"username": user.Username})
	if err != nil {
		return err
	}
	if exists {
		return ErrUserExists
	}

	// 检查邮箱是否存在
	exists, err = d.Exists(ctx, map[string]interface{}{"email": user.Email})
	if err != nil {
		return err
	}
	if exists {
		return errors.New("邮箱已被注册")
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("密码加密失败", zap.Error(err))
		return err
	}
	user.Password = string(hashedPassword)

	// 调用基础创建方法
	return d.BaseDAOImpl.Create(ctx, user)
}

// GetByUsername 根据用户名获取用户
func (d *UserDAO) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	return d.GetOne(ctx, map[string]interface{}{"username": username})
}

// GetByEmail 根据邮箱获取用户
func (d *UserDAO) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return d.GetOne(ctx, map[string]interface{}{"email": email})
}

// List 获取用户列表（重写基础方法，支持关键词搜索）
func (d *UserDAO) List(ctx context.Context, options *QueryOptions) ([]*model.User, int64, error) {
	// 如果options为空，创建默认选项
	if options == nil {
		options = &QueryOptions{
			OrderBy:  "created_at",
			OrderDir: "desc",
		}
	}

	// 处理关键词搜索
	if options.Keyword != "" {
		if options.Filters == nil {
			options.Filters = make(map[string]interface{})
		}
		options.Filters["username LIKE ? OR email LIKE ? OR nickname LIKE ?"] = "%" + options.Keyword + "%"
	}

	return d.BaseDAOImpl.List(ctx, options)
}

// UpdatePassword 更新密码（特殊方法）
func (d *UserDAO) UpdatePassword(ctx context.Context, id uint, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return d.Update(ctx, id, map[string]interface{}{"password": string(hashedPassword)})
}

// UpdateLastLogin 更新最后登录时间（特殊方法）
func (d *UserDAO) UpdateLastLogin(ctx context.Context, id uint) error {
	return d.Update(ctx, id, map[string]interface{}{"last_login": gorm.Expr("NOW()")})
}

// CheckPassword 验证密码（特殊方法）
func (d *UserDAO) CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// 以下方法现在通过基础接口提供，无需重复实现：
// - GetByID: 通过基础接口的 GetByID 方法
// - Update: 通过基础接口的 Update 方法（已过滤密码字段）
// - Delete: 通过基础接口的 Delete 方法
// - HardDelete: 通过基础接口的 HardDelete 方法