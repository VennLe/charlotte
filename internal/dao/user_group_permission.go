package dao

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/VennLe/charlotte/internal/model"
)

// UserGroupPermissionDAO 用户组权限关联数据访问对象
type UserGroupPermissionDAO struct {
	*BaseDAOImpl[model.UserGroupPermission, uint]
}

// NewUserGroupPermissionDAO 创建用户组权限关联DAO实例
func NewUserGroupPermissionDAO(db *gorm.DB) *UserGroupPermissionDAO {
	return &UserGroupPermissionDAO{
		BaseDAOImpl: NewBaseDAO[model.UserGroupPermission, uint](db),
	}
}

// GetGroupPermissions 获取用户组的权限列表
func (d *UserGroupPermissionDAO) GetGroupPermissions(ctx context.Context, groupID uint) ([]*model.UserGroupPermission, error) {
	return d.GetMany(ctx, map[string]interface{}{"user_group_id": groupID})
}

// GetPermissionGroups 获取拥有指定权限的用户组
func (d *UserGroupPermissionDAO) GetPermissionGroups(ctx context.Context, permissionTagID uint) ([]*model.UserGroupPermission, error) {
	return d.GetMany(ctx, map[string]interface{}{"permission_tag_id": permissionTagID})
}

// GetGroupPermissionByTag 获取用户组对特定标签的权限
func (d *UserGroupPermissionDAO) GetGroupPermissionByTag(ctx context.Context, groupID uint, tag string) (*model.UserGroupPermission, error) {
	permissionTagID, err := d.getPermissionTagIDByTag(ctx, tag)
	if err != nil {
		return nil, err
	}
	if permissionTagID == 0 {
		return nil, nil
	}

	return d.GetOne(ctx, map[string]interface{}{
		"user_group_id":     groupID,
		"permission_tag_id": permissionTagID,
	})
}

// AddPermissionToGroup 为用户组添加权限
func (d *UserGroupPermissionDAO) AddPermissionToGroup(ctx context.Context, groupID uint, permissionTagID uint, operations, resourceType, resourceScope string) error {
	permission := &model.UserGroupPermission{
		UserGroupID:     groupID,
		PermissionTagID: permissionTagID,
		Operations:      operations,
		ResourceType:    resourceType,
		ResourceScope:   resourceScope,
	}
	return d.Create(ctx, permission)
}

// RemovePermissionFromGroup 从用户组移除权限
func (d *UserGroupPermissionDAO) RemovePermissionFromGroup(ctx context.Context, groupID uint, permissionTagID uint) error {
	permission, err := d.GetOne(ctx, map[string]interface{}{
		"user_group_id":     groupID,
		"permission_tag_id": permissionTagID,
	})
	if err != nil {
		return err
	}
	return d.Delete(ctx, permission.ID)
}

// UpdateGroupPermission 更新用户组权限
func (d *UserGroupPermissionDAO) UpdateGroupPermission(ctx context.Context, groupID uint, permissionTagID uint, operations, resourceType, resourceScope string) error {
	permission, err := d.GetOne(ctx, map[string]interface{}{
		"user_group_id":     groupID,
		"permission_tag_id": permissionTagID,
	})
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"operations":     operations,
		"resource_type":  resourceType,
		"resource_scope": resourceScope,
	}
	return d.Update(ctx, permission.ID, updates)
}

// GetPermissionsByResourceType 根据资源类型获取权限
func (d *UserGroupPermissionDAO) GetPermissionsByResourceType(ctx context.Context, resourceType string) ([]*model.UserGroupPermission, error) {
	return d.GetMany(ctx, map[string]interface{}{"resource_type": resourceType})
}

// GetGroupsWithPermission 获取拥有特定权限的用户组
func (d *UserGroupPermissionDAO) GetGroupsWithPermission(ctx context.Context, tag string, operation string) ([]*model.UserGroupPermission, error) {
	permissionTagID, err := d.getPermissionTagIDByTag(ctx, tag)
	if err != nil || permissionTagID == 0 {
		return nil, nil
	}

	permissions, err := d.GetMany(ctx, map[string]interface{}{"permission_tag_id": permissionTagID})
	if err != nil {
		return nil, err
	}

	var result []*model.UserGroupPermission
	for _, perm := range permissions {
		if d.containsOperation(perm.Operations, operation) {
			result = append(result, perm)
		}
	}

	return result, nil
}

// CreateDefaultPermissions 创建默认权限配置（系统初始化时调用）
func (d *UserGroupPermissionDAO) CreateDefaultPermissions(ctx context.Context) error {
	// 获取默认用户组
	groupDAO := NewUserGroupDAO(d.DB)
	groups, err := groupDAO.GetDefaultGroups(ctx)
	if err != nil {
		return err
	}

	// 获取权限标签
	tagDAO := NewPermissionTagDAO(d.DB)
	tags, err := tagDAO.GetMany(ctx, nil)
	if err != nil {
		return err
	}

	// 为每个用户组配置默认权限
	for _, group := range groups {
		switch group.Name {
		case "超级管理员组":
			// 超级管理员拥有所有权限
			for _, tag := range tags {
				d.AddPermissionToGroup(ctx, group.ID, tag.ID, model.PermissionAll, tag.Category, "all")
			}
		case "管理员组":
			// 管理员拥有除删除外的所有权限
			for _, tag := range tags {
				if tag.Tag != "user_delete" && tag.Tag != "content_delete" {
					d.AddPermissionToGroup(ctx, group.ID, tag.ID, "read,write,create", tag.Category, "all")
				}
			}
		case "VIP用户组":
			// VIP用户拥有查看权限
			for _, tag := range tags {
				if strings.Contains(tag.Tag, "_read") {
					d.AddPermissionToGroup(ctx, group.ID, tag.ID, model.PermissionRead, tag.Category, "own")
				}
			}
		case "普通用户组":
			// 普通用户拥有基础查看权限
			for _, tag := range tags {
				if tag.Tag == "user_read" || tag.Tag == "content_read" {
					d.AddPermissionToGroup(ctx, group.ID, tag.ID, model.PermissionRead, tag.Category, "own")
				}
			}
		case "游客组":
			// 游客只有极少的查看权限
			for _, tag := range tags {
				if tag.Tag == "content_read" {
					d.AddPermissionToGroup(ctx, group.ID, tag.ID, model.PermissionRead, tag.Category, "all")
				}
			}
		}
	}

	return nil
}

// 辅助方法
func (d *UserGroupPermissionDAO) getPermissionTagIDByTag(ctx context.Context, tag string) (uint, error) {
	tagDAO := NewPermissionTagDAO(d.DB)
	permissionTag, err := tagDAO.GetTagByTagName(ctx, tag)
	if err != nil {
		return 0, err
	}
	return permissionTag.ID, nil
}

func (d *UserGroupPermissionDAO) containsOperation(operations, targetOp string) bool {
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
