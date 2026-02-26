package dao

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/VennLe/charlotte/internal/model"
)

// UnifiedPermissionDAO 统一权限数据访问对象
// 整合了所有权限相关的操作，简化权限标签为角色标记

type UnifiedPermissionDAO struct {
	db *gorm.DB
}

// NewUnifiedPermissionDAO 创建统一权限DAO实例
func NewUnifiedPermissionDAO(db *gorm.DB) *UnifiedPermissionDAO {
	return &UnifiedPermissionDAO{db: db}
}

// UserRole 用户角色模型（简化版）
type UserRole struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	UserID      uint   `gorm:"not null;uniqueIndex" json:"user_id"`
	Role       string `gorm:"size:20;not null;index" json:"role"`
	IsActive   bool   `gorm:"default:true" json:"is_active"`
	ExpiredAt  time.Time `json:"expired_at"`
}

func (UserRole) TableName() string {
	return "user_roles"
}

// RolePermission 角色权限模型（简化版）
type RolePermission struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Role       string `gorm:"size:20;not null;index" json:"role"`
	ResourceType string `gorm:"size:50;not null" json:"resource_type"`
	Operations  string `gorm:"size:100;not null" json:"operations"`
	Scope      string `gorm:"size:20;default:'own'" json:"scope"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}

// InitializeDefaultPermissions 初始化默认权限配置
func (d *UnifiedPermissionDAO) InitializeDefaultPermissions(ctx context.Context) error {
	// 创建默认角色权限配置
	defaultPermissions := []RolePermission{
		// 超级管理员权限
		{Role: model.RoleSuperAdmin, ResourceType: "*", Operations: model.PermissionAll, Scope: "all"},

		// 普通管理员权限（不能删除）
		{Role: model.RoleAdmin, ResourceType: "user", Operations: "read,write,create", Scope: "all"},
		{Role: model.RoleAdmin, ResourceType: "content", Operations: "read,write,create", Scope: "all"},
		{Role: model.RoleAdmin, ResourceType: "system", Operations: "read,write", Scope: "all"},

		// VIP用户权限（只能查看）
		{Role: model.RoleVIP, ResourceType: "user", Operations: model.PermissionRead, Scope: "own"},
		{Role: model.RoleVIP, ResourceType: "content", Operations: model.PermissionRead, Scope: "all"},

		// 普通用户权限（只能查看自己的）
		{Role: model.RoleUser, ResourceType: "user", Operations: model.PermissionRead, Scope: "own"},
		{Role: model.RoleUser, ResourceType: "content", Operations: model.PermissionRead, Scope: "own"},

		// 游客权限（只能查看公开内容）
		{Role: model.RoleGuest, ResourceType: "content", Operations: model.PermissionRead, Scope: "public"},
	}

	for _, perm := range defaultPermissions {
		exists, err := d.checkPermissionExists(ctx, perm.Role, perm.ResourceType)
		if err != nil {
			return err
		}
		if !exists {
			if err := d.db.WithContext(ctx).Create(&perm).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// SetUserRole 设置用户角色
func (d *UnifiedPermissionDAO) SetUserRole(ctx context.Context, userID uint, role string) error {
	// 先检查是否已有角色记录
	var existingRole UserRole
	err := d.db.WithContext(ctx).Where("user_id = ? AND is_active = ?", userID, true).First(&existingRole).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新角色记录
		userRole := UserRole{
			UserID: userID,
			Role:    role,
			IsActive: true,
		}
		return d.db.WithContext(ctx).Create(&userRole).Error
	} else if err != nil {
		return err
	}

	// 更新现有角色记录
	return d.db.WithContext(ctx).Model(&UserRole{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Update("role", role).Error
}

// GetUserRole 获取用户角色
func (d *UnifiedPermissionDAO) GetUserRole(ctx context.Context, userID uint) (string, error) {
	var userRole UserRole
	err := d.db.WithContext(ctx).
		Where("user_id = ? AND is_active = ?", userID, true).
		First(&userRole).Error

	if err == gorm.ErrRecordNotFound {
		// 默认返回游客角色
		return model.RoleGuest, nil
	}
	if err != nil {
		return "", err
	}

	return userRole.Role, nil
}

// CheckPermission 检查用户权限
func (d *UnifiedPermissionDAO) CheckPermission(ctx context.Context, userID uint, resourceType, operation string) (bool, error) {
	// 获取用户角色
	role, err := d.GetUserRole(ctx, userID)
	if err != nil {
		return false, err
	}

	// 超级管理员拥有所有权限
	if role == model.RoleSuperAdmin {
		return true, nil
	}

	// 检查角色权限
	var permissions []RolePermission
	err = d.db.WithContext(ctx).
		Where("(role = ? AND resource_type = ?) OR (role = ? AND resource_type = '*')", role, resourceType, role).
		Find(&permissions).Error
	if err != nil {
		return false, err
	}

	for _, perm := range permissions {
		if d.containsOperation(perm.Operations, operation) {
			return true, nil
		}
	}

	return false, nil
}

// GetUserPermissions 获取用户的所有权限
func (d *UnifiedPermissionDAO) GetUserPermissions(ctx context.Context, userID uint) (map[string]interface{}, error) {
	role, err := d.GetUserRole(ctx, userID)
	if err != nil {
		return nil, err
	}

	var permissions []RolePermission
	err = d.db.WithContext(ctx).Where("role = ?", role).Find(&permissions).Error
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"user_id": userID,
		"role":    role,
		"permissions": permissions,
	}

	return result, nil
}

// GrantTemporaryRole 授予临时角色
func (d *UnifiedPermissionDAO) GrantTemporaryRole(ctx context.Context, userID uint, role string, duration time.Duration) error {
	expiredAt := time.Now().Add(duration)
	
	userRole := UserRole{
		UserID:    userID,
		Role:      role,
		IsActive:  true,
		ExpiredAt: expiredAt,
	}
	
	return d.db.WithContext(ctx).Create(&userRole).Error
}

// RevokeRole 撤销用户角色
func (d *UnifiedPermissionDAO) RevokeRole(ctx context.Context, userID uint) error {
	return d.db.WithContext(ctx).
		Model(&UserRole{}).
		Where("user_id = ?", userID).
		Update("is_active", false).Error
}

// AddRolePermission 添加角色权限
func (d *UnifiedPermissionDAO) AddRolePermission(ctx context.Context, role, resourceType, operations, scope string) error {
	permission := RolePermission{
		Role:         role,
		ResourceType: resourceType,
		Operations:   operations,
		Scope:        scope,
	}
	
	return d.db.WithContext(ctx).Create(&permission).Error
}

// RemoveRolePermission 移除角色权限
func (d *UnifiedPermissionDAO) RemoveRolePermission(ctx context.Context, role, resourceType string) error {
	return d.db.WithContext(ctx).
		Where("role = ? AND resource_type = ?", role, resourceType).
		Delete(&RolePermission{}).Error
}

// GetRolePermissions 获取角色的所有权限
func (d *UnifiedPermissionDAO) GetRolePermissions(ctx context.Context, role string) ([]RolePermission, error) {
	var permissions []RolePermission
	err := d.db.WithContext(ctx).Where("role = ?", role).Find(&permissions).Error
	return permissions, err
}

// 辅助方法
func (d *UnifiedPermissionDAO) checkPermissionExists(ctx context.Context, role, resourceType string) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&RolePermission{}).
		Where("role = ? AND resource_type = ?", role, resourceType).
		Count(&count).Error
	return count > 0, err
}

func (d *UnifiedPermissionDAO) containsOperation(operations, targetOp string) bool {
	if operations == model.PermissionAll {
		return true
	}
	
	ops := strings.Split(operations, ",")
	for _, op := range ops {
		if strings.TrimSpace(op) == targetOp {
			return true
		}
	}
	
	return false
}