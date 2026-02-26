package dao

import (
	"context"

	"gorm.io/gorm"

	"github.com/VennLe/charlotte/internal/model"
)

// UserGroupDAO 用户组数据访问对象
type UserGroupDAO struct {
	*BaseDAOImpl[model.UserGroup, uint]
}

// NewUserGroupDAO 创建用户组DAO实例
func NewUserGroupDAO(db *gorm.DB) *UserGroupDAO {
	return &UserGroupDAO{
		BaseDAOImpl: NewBaseDAO[model.UserGroup, uint](db),
	}
}

// GetDefaultGroups 获取默认用户组
func (d *UserGroupDAO) GetDefaultGroups(ctx context.Context) ([]*model.UserGroup, error) {
	return d.GetMany(ctx, map[string]interface{}{"is_default": true, "status": 1})
}

// GetGroupsByLevel 根据权限级别获取用户组
func (d *UserGroupDAO) GetGroupsByLevel(ctx context.Context, level int) ([]*model.UserGroup, error) {
	return d.GetMany(ctx, map[string]interface{}{"level": level, "status": 1})
}

// GetGroupByName 根据名称获取用户组
func (d *UserGroupDAO) GetGroupByName(ctx context.Context, name string) (*model.UserGroup, error) {
	return d.GetOne(ctx, map[string]interface{}{"name": name})
}

// CreateDefaultGroups 创建默认用户组（系统初始化时调用）
func (d *UserGroupDAO) CreateDefaultGroups(ctx context.Context) error {
	defaultGroups := []*model.UserGroup{
		{
			Name:        "游客组",
			Description: "未登录用户的默认组",
			Level:       model.PermissionLevelLow,
			IsDefault:   true,
			Status:      1,
		},
		{
			Name:        "普通用户组",
			Description: "普通注册用户的默认组",
			Level:       model.PermissionLevelLow,
			IsDefault:   true,
			Status:      1,
		},
		{
			Name:        "VIP用户组",
			Description: "VIP用户的默认组",
			Level:       model.PermissionLevelMedium,
			IsDefault:   true,
			Status:      1,
		},
		{
			Name:        "管理员组",
			Description: "普通管理员的默认组",
			Level:       model.PermissionLevelHigh,
			IsDefault:   true,
			Status:      1,
		},
		{
			Name:        "超级管理员组",
			Description: "超级管理员的默认组",
			Level:       model.PermissionLevelHigh,
			IsDefault:   true,
			Status:      1,
		},
	}

	for _, group := range defaultGroups {
		exists, err := d.Exists(ctx, map[string]interface{}{"name": group.Name})
		if err != nil {
			return err
		}
		if !exists {
			if err := d.Create(ctx, group); err != nil {
				return err
			}
		}
	}

	return nil
}

// UpdateGroupStatus 更新用户组状态
func (d *UserGroupDAO) UpdateGroupStatus(ctx context.Context, id uint, status int) error {
	return d.Update(ctx, id, map[string]interface{}{"status": status})
}

// GetActiveGroups 获取活跃的用户组
func (d *UserGroupDAO) GetActiveGroups(ctx context.Context) ([]*model.UserGroup, error) {
	return d.GetMany(ctx, map[string]interface{}{"status": 1})
}