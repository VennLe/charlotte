package dao

import (
	"context"

	"gorm.io/gorm"

	"github.com/VennLe/charlotte/internal/model"
)

// PermissionTagDAO 权限标签数据访问对象
type PermissionTagDAO struct {
	*BaseDAOImpl[model.PermissionTag, uint]
}

// NewPermissionTagDAO 创建权限标签DAO实例
func NewPermissionTagDAO(db *gorm.DB) *PermissionTagDAO {
	return &PermissionTagDAO{
		BaseDAOImpl: NewBaseDAO[model.PermissionTag, uint](db),
	}
}

// GetTagByTagName 根据标签名获取权限标签
func (d *PermissionTagDAO) GetTagByTagName(ctx context.Context, tag string) (*model.PermissionTag, error) {
	return d.GetOne(ctx, map[string]interface{}{"tag": tag})
}

// GetTagsByCategory 根据分类获取权限标签
func (d *PermissionTagDAO) GetTagsByCategory(ctx context.Context, category string) ([]*model.PermissionTag, error) {
	return d.GetMany(ctx, map[string]interface{}{"category": category})
}

// GetTagsByLevel 根据权限级别获取标签
func (d *PermissionTagDAO) GetTagsByLevel(ctx context.Context, level int) ([]*model.PermissionTag, error) {
	return d.GetMany(ctx, map[string]interface{}{"level": level})
}

// CreateDefaultTags 创建默认权限标签（系统初始化时调用）
func (d *PermissionTagDAO) CreateDefaultTags(ctx context.Context) error {
	defaultTags := []*model.PermissionTag{
		// 用户管理权限
		{
			Tag:         "user_read",
			Name:        "用户查看权限",
			Description: "允许查看用户信息",
			Category:    "user",
			Level:       model.PermissionLevelLow,
		},
		{
			Tag:         "user_write",
			Name:        "用户修改权限",
			Description: "允许修改用户信息",
			Category:    "user",
			Level:       model.PermissionLevelMedium,
		},
		{
			Tag:         "user_delete",
			Name:        "用户删除权限",
			Description: "允许删除用户",
			Category:    "user",
			Level:       model.PermissionLevelHigh,
		},
		{
			Tag:         "user_create",
			Name:        "用户创建权限",
			Description: "允许创建新用户",
			Category:    "user",
			Level:       model.PermissionLevelMedium,
		},

		// 系统管理权限
		{
			Tag:         "system_config",
			Name:        "系统配置权限",
			Description: "允许修改系统配置",
			Category:    "system",
			Level:       model.PermissionLevelHigh,
		},
		{
			Tag:         "system_backup",
			Name:        "系统备份权限",
			Description: "允许执行系统备份",
			Category:    "system",
			Level:       model.PermissionLevelHigh,
		},

		// 内容管理权限
		{
			Tag:         "content_read",
			Name:        "内容查看权限",
			Description: "允许查看内容",
			Category:    "content",
			Level:       model.PermissionLevelLow,
		},
		{
			Tag:         "content_write",
			Name:        "内容修改权限",
			Description: "允许修改内容",
			Category:    "content",
			Level:       model.PermissionLevelMedium,
		},
		{
			Tag:         "content_delete",
			Name:        "内容删除权限",
			Description: "允许删除内容",
			Category:    "content",
			Level:       model.PermissionLevelHigh,
		},
		{
			Tag:         "content_create",
			Name:        "内容创建权限",
			Description: "允许创建新内容",
			Category:    "content",
			Level:       model.PermissionLevelMedium,
		},

		// 权限管理权限
		{
			Tag:         "permission_manage",
			Name:        "权限管理权限",
			Description: "允许管理用户权限",
			Category:    "permission",
			Level:       model.PermissionLevelHigh,
		},
		{
			Tag:         "user_group_manage",
			Name:        "用户组管理权限",
			Description: "允许管理用户组",
			Category:    "permission",
			Level:       model.PermissionLevelHigh,
		},
	}

	for _, tag := range defaultTags {
		exists, err := d.Exists(ctx, map[string]interface{}{"tag": tag.Tag})
		if err != nil {
			return err
		}
		if !exists {
			if err := d.Create(ctx, tag); err != nil {
				return err
			}
		}
	}

	return nil
}

// SearchTags 搜索权限标签
func (d *PermissionTagDAO) SearchTags(ctx context.Context, keyword string) ([]*model.PermissionTag, error) {
	options := &QueryOptions{
		Filters: map[string]interface{}{
			"tag LIKE ? OR name LIKE ? OR description LIKE ?": "%" + keyword + "%",
		},
		OrderBy:  "category, level",
		OrderDir: "asc",
	}

	tags, _, err := d.List(ctx, options)
	return tags, err
}

// GetTagsWithCategory 获取带分类的权限标签
func (d *PermissionTagDAO) GetTagsWithCategory(ctx context.Context) (map[string][]*model.PermissionTag, error) {
	tags, err := d.GetMany(ctx, nil)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]*model.PermissionTag)
	for _, tag := range tags {
		result[tag.Category] = append(result[tag.Category], tag)
	}

	return result, nil
}