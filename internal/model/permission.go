package model

import (
	"time"

	"gorm.io/gorm"
)

// 权限常量定义
const (
	// 用户角色
	RoleGuest      = "guest"     // 游客（未登录）
	RoleUser       = "user"      // 普通用户
	RoleVIP        = "vip"       // VIP用户
	RoleAdmin      = "admin"     // 普通管理员
	RoleSuperAdmin = "superadmin" // 超级管理员

	// 权限操作
	PermissionRead   = "read"   // 查看权限
	PermissionWrite  = "write"  // 修改权限
	PermissionDelete = "delete" // 删除权限
	PermissionAll    = "all"    // 所有权限

	// 权限级别
	PermissionLevelLow    = 1 // 低权限级别
	PermissionLevelMedium = 2 // 中权限级别
	PermissionLevelHigh   = 3 // 高权限级别
)

// UserGroup 用户组模型
type UserGroup struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string `gorm:"size:50;not null;uniqueIndex" json:"name"`
	Description string `gorm:"size:255" json:"description"`
	Level       int    `gorm:"default:1;comment:权限级别 1-低 2-中 3-高" json:"level"`
	IsDefault   bool   `gorm:"default:false;comment:是否为默认组" json:"is_default"`
	Status      int    `gorm:"default:1;comment:1启用 2禁用" json:"status"`
}

// PermissionTag 权限标签模型
type PermissionTag struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Tag         string `gorm:"size:50;not null;uniqueIndex" json:"tag"`
	Name        string `gorm:"size:50;not null" json:"name"`
	Description string `gorm:"size:255" json:"description"`
	Category    string `gorm:"size:50;comment:权限分类" json:"category"`
	Level       int    `gorm:"default:1;comment:权限级别" json:"level"`
}

// UserGroupPermission 用户组权限关联表
type UserGroupPermission struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	UserGroupID     uint          `gorm:"not null;index" json:"user_group_id"`
	PermissionTagID uint          `gorm:"not null;index" json:"permission_tag_id"`
	Operations      string        `gorm:"size:100;comment:允许的操作(read,write,delete,all)" json:"operations"`
	ResourceType    string        `gorm:"size:50;comment:资源类型(user,article,comment等)" json:"resource_type"`
	ResourceScope   string        `gorm:"size:100;comment:资源范围(all,own,system等)" json:"resource_scope"`
	Conditions      string        `gorm:"type:text;comment:权限条件(JSON格式)" json:"conditions"`

	UserGroup       UserGroup     `gorm:"foreignKey:UserGroupID" json:"-"`
	PermissionTag   PermissionTag `gorm:"foreignKey:PermissionTagID" json:"-"`
}

// UserGroupMember 用户组成员关联表
type UserGroupMember struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	UserID      uint      `gorm:"not null;uniqueIndex:idx_user_group" json:"user_id"`
	UserGroupID uint      `gorm:"not null;uniqueIndex:idx_user_group" json:"user_group_id"`
	JoinedAt    time.Time `json:"joined_at"`
	ExpiredAt   time.Time `json:"expired_at"`
	Status      int       `gorm:"default:1;comment:1正常 2禁用" json:"status"`

	User        User      `gorm:"foreignKey:UserID" json:"-"`
	UserGroup   UserGroup `gorm:"foreignKey:UserGroupID" json:"-"`
}

// UserPermission 用户特殊权限表（覆盖组权限）
type UserPermission struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	UserID          uint          `gorm:"not null;index" json:"user_id"`
	PermissionTagID uint          `gorm:"not null;index" json:"permission_tag_id"`
	Operations      string        `gorm:"size:100;comment:允许的操作" json:"operations"`
	ResourceType    string        `gorm:"size:50" json:"resource_type"`
	ResourceScope   string        `gorm:"size:100" json:"resource_scope"`
	IsGrant         bool          `gorm:"default:true;comment:true授予权限 false撤销权限" json:"is_grant"`
	ExpiredAt       time.Time     `json:"expired_at"`

	User            User          `gorm:"foreignKey:UserID" json:"-"`
	PermissionTag   PermissionTag `gorm:"foreignKey:PermissionTagID" json:"-"`
}

// PermissionCheckRequest 权限检查请求
type PermissionCheckRequest struct {
	UserID       uint   `json:"user_id"`
	ResourceType string `json:"resource_type"`
	Operation    string `json:"operation"`
	ResourceID   uint   `json:"resource_id,omitempty"`
}

// PermissionCheckResult 权限检查结果
type PermissionCheckResult struct {
	HasPermission bool   `json:"has_permission"`
	Reason       string `json:"reason,omitempty"`
	UserRole     string `json:"user_role"`
	AllowedOps   string `json:"allowed_operations"`
}

// TableName 方法定义
func (UserGroup) TableName() string {
	return "user_groups"
}

func (PermissionTag) TableName() string {
	return "permission_tags"
}

func (UserGroupPermission) TableName() string {
	return "user_group_permissions"
}

func (UserGroupMember) TableName() string {
	return "user_group_members"
}

func (UserPermission) TableName() string {
	return "user_permissions"
}

// BeforeCreate 钩子函数
func (ug *UserGroup) BeforeCreate(tx *gorm.DB) error {
	if ug.Level == 0 {
		ug.Level = PermissionLevelLow
	}
	if ug.Status == 0 {
		ug.Status = 1
	}
	return nil
}

func (pt *PermissionTag) BeforeCreate(tx *gorm.DB) error {
	if pt.Level == 0 {
		pt.Level = PermissionLevelLow
	}
	return nil
}

func (ugm *UserGroupMember) BeforeCreate(tx *gorm.DB) error {
	if ugm.JoinedAt.IsZero() {
		ugm.JoinedAt = time.Now()
	}
	if ugm.Status == 0 {
		ugm.Status = 1
	}
	return nil
}