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
type UserDAO struct {
	db *gorm.DB
}

// NewUserDAO 创建 DAO 实例
func NewUserDAO(db *gorm.DB) *UserDAO {
	return &UserDAO{
		db: db,
	}
}

// Create 创建用户
func (d *UserDAO) Create(ctx context.Context, user *model.User) error {
	// 检查用户名是否存在
	var count int64
	if err := d.db.WithContext(ctx).Model(&model.User{}).Where("username = ?", user.Username).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrUserExists
	}

	// 检查邮箱是否存在
	if err := d.db.WithContext(ctx).Model(&model.User{}).Where("email = ?", user.Email).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("邮箱已被注册")
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("密码加密失败", zap.Error(err))
		return err
	}
	user.Password = string(hashedPassword)

	return d.db.WithContext(ctx).Create(user).Error
}

// GetByID 根据 ID 获取用户
func (d *UserDAO) GetByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	if err := d.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (d *UserDAO) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	if err := d.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail 根据邮箱获取用户
func (d *UserDAO) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := d.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// List 获取用户列表
func (d *UserDAO) List(ctx context.Context, page, size int, keyword string) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	query := d.db.WithContext(ctx).Model(&model.User{})
	if keyword != "" {
		query = query.Where("username LIKE ? OR email LIKE ? OR nickname LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Offset((page - 1) * size).Limit(size).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// Update 更新用户
func (d *UserDAO) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	// 不允许直接更新密码，使用单独的方法
	delete(updates, "password")
	delete(updates, "id")

	result := d.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdatePassword 更新密码
func (d *UserDAO) UpdatePassword(ctx context.Context, id uint, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	result := d.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("password", string(hashedPassword))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// Delete 删除用户 (软删除)
func (d *UserDAO) Delete(ctx context.Context, id uint) error {
	result := d.db.WithContext(ctx).Delete(&model.User{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// HardDelete 硬删除
func (d *UserDAO) HardDelete(ctx context.Context, id uint) error {
	result := d.db.WithContext(ctx).Unscoped().Delete(&model.User{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdateLastLogin 更新最后登录时间
func (d *UserDAO) UpdateLastLogin(ctx context.Context, id uint) error {
	return d.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("last_login", gorm.Expr("NOW()")).Error
}

// CheckPassword 验证密码
func (d *UserDAO) CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}