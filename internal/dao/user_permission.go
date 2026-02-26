package dao

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/VennLe/charlotte/internal/model"
)

// UserPermissionDAO 用户特殊权限数据访问对象
type UserPermissionDAO struct {
	*BaseDAOImpl[model.UserPermission, uint]
}

// NewUserPermissionDAO 创建用户特殊权限DAO实例
func NewUserPermissionDAO(db *gorm.DB) *UserPermissionDAO {
	return &UserPermissionDAO{
		BaseDAOImpl: NewBaseDAO[model.UserPermission, uint](db),
	}
}

// GetUserPermissions 获取用户的所有特殊权限
func (d *UserPermissionDAO) GetUserPermissions(ctx context.Context, userID uint) ([]*model.UserPermission, error) {
	return d.GetMany(ctx, map[string]interface{}{"user_id": userID})
}

// GetActiveUserPermissions 获取用户的有效特殊权限（未过期）
func (d *UserPermissionDAO) GetActiveUserPermissions(ctx context.Context, userID uint) ([]*model.UserPermission, error) {
	now := time.Now()
	return d.GetMany(ctx, map[string]interface{}{
		"user_id":                              userID,
		"is_grant":                             true,
		"expired_at > ? OR expired_at IS NULL": now,
	})
}

// GetPermissionByTag 获取用户对特定标签的特殊权限
func (d *UserPermissionDAO) GetPermissionByTag(ctx context.Context, userID uint, tag string) (*model.UserPermission, error) {
	permissionTagID, err := d.getPermissionTagIDByTag(ctx, tag)
	if err != nil {
		return nil, err
	}
	if permissionTagID == 0 {
		return nil, nil
	}

	return d.GetOne(ctx, map[string]interface{}{
		"user_id":           userID,
		"permission_tag_id": permissionTagID,
	})
}

// GrantPermission 授予用户特殊权限
func (d *UserPermissionDAO) GrantPermission(ctx context.Context, userID uint, permissionTagID uint, operations, resourceType, resourceScope string, expiredAt time.Time) error {
	permission := &model.UserPermission{
		UserID:          userID,
		PermissionTagID: permissionTagID,
		Operations:      operations,
		ResourceType:    resourceType,
		ResourceScope:   resourceScope,
		IsGrant:         true,
		ExpiredAt:       expiredAt,
	}
	return d.Create(ctx, permission)
}

// RevokePermission 撤销用户特殊权限
func (d *UserPermissionDAO) RevokePermission(ctx context.Context, userID uint, permissionTagID uint) error {
	permission, err := d.GetOne(ctx, map[string]interface{}{
		"user_id":           userID,
		"permission_tag_id": permissionTagID,
	})
	if err != nil {
		return err
	}

	// 软删除权限记录
	return d.Delete(ctx, permission.ID)
}

// UpdatePermission 更新用户特殊权限
func (d *UserPermissionDAO) UpdatePermission(ctx context.Context, userID uint, permissionTagID uint, operations, resourceType, resourceScope string, expiredAt time.Time) error {
	permission, err := d.GetOne(ctx, map[string]interface{}{
		"user_id":           userID,
		"permission_tag_id": permissionTagID,
	})
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"operations":     operations,
		"resource_type":  resourceType,
		"resource_scope": resourceScope,
		"expired_at":     expiredAt,
	}
	return d.Update(ctx, permission.ID, updates)
}

// GrantPermissionByTag 通过标签名授予权限
func (d *UserPermissionDAO) GrantPermissionByTag(ctx context.Context, userID uint, tag string, operations, resourceType, resourceScope string, expiredAt time.Time) error {
	permissionTagID, err := d.getPermissionTagIDByTag(ctx, tag)
	if err != nil {
		return err
	}
	if permissionTagID == 0 {
		return nil // 权限标签不存在
	}

	return d.GrantPermission(ctx, userID, permissionTagID, operations, resourceType, resourceScope, expiredAt)
}

// RevokePermissionByTag 通过标签名撤销权限
func (d *UserPermissionDAO) RevokePermissionByTag(ctx context.Context, userID uint, tag string) error {
	permissionTagID, err := d.getPermissionTagIDByTag(ctx, tag)
	if err != nil {
		return err
	}
	if permissionTagID == 0 {
		return nil // 权限标签不存在
	}

	return d.RevokePermission(ctx, userID, permissionTagID)
}

// HasPermission 检查用户是否拥有特定权限
func (d *UserPermissionDAO) HasPermission(ctx context.Context, userID uint, tag string, operation string) (bool, error) {
	permissions, err := d.GetActiveUserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, perm := range permissions {
		if d.getPermissionTagByID(ctx, perm.PermissionTagID) == tag {
			if d.containsOperation(perm.Operations, operation) {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetExpiredPermissions 获取已过期的权限
func (d *UserPermissionDAO) GetExpiredPermissions(ctx context.Context) ([]*model.UserPermission, error) {
	now := time.Now()
	return d.GetMany(ctx, map[string]interface{}{
		"expired_at <= ?":        now,
		"expired_at IS NOT NULL": nil,
	})
}

// CleanExpiredPermissions 清理已过期的权限
func (d *UserPermissionDAO) CleanExpiredPermissions(ctx context.Context) error {
	expiredPermissions, err := d.GetExpiredPermissions(ctx)
	if err != nil {
		return err
	}

	for _, perm := range expiredPermissions {
		if err := d.Delete(ctx, perm.ID); err != nil {
			return err
		}
	}

	return nil
}

// GetPermissionsByResourceType 根据资源类型获取权限
func (d *UserPermissionDAO) GetPermissionsByResourceType(ctx context.Context, resourceType string) ([]*model.UserPermission, error) {
	return d.GetMany(ctx, map[string]interface{}{"resource_type": resourceType})
}

// GetUsersWithPermission 获取拥有特定权限的用户
func (d *UserPermissionDAO) GetUsersWithPermission(ctx context.Context, tag string, operation string) ([]*model.UserPermission, error) {
	permissionTagID, err := d.getPermissionTagIDByTag(ctx, tag)
	if err != nil {
		return nil, err
	}
	if permissionTagID == 0 {
		return nil, nil
	}

	permissions, err := d.GetMany(ctx, map[string]interface{}{
		"permission_tag_id": permissionTagID,
		"is_grant":          true,
	})
	if err != nil {
		return nil, err
	}

	var result []*model.UserPermission
	for _, perm := range permissions {
		if d.containsOperation(perm.Operations, operation) {
			result = append(result, perm)
		}
	}

	return result, nil
}

// GrantTemporaryPermission 授予临时权限
func (d *UserPermissionDAO) GrantTemporaryPermission(ctx context.Context, userID uint, tag string, operations string, duration time.Duration) error {
	expiredAt := time.Now().Add(duration)
	return d.GrantPermissionByTag(ctx, userID, tag, operations, "", "", expiredAt)
}

// 辅助方法
func (d *UserPermissionDAO) getPermissionTagIDByTag(ctx context.Context, tag string) (uint, error) {
	tagDAO := NewPermissionTagDAO(d.DB)
	permissionTag, err := tagDAO.GetTagByTagName(ctx, tag)
	if err != nil {
		return 0, err
	}
	return permissionTag.ID, nil
}

func (d *UserPermissionDAO) getPermissionTagByID(ctx context.Context, tagID uint) string {
	tagDAO := NewPermissionTagDAO(d.DB)
	permissionTag, err := tagDAO.GetByID(ctx, tagID)
	if err != nil {
		return ""
	}
	return permissionTag.Tag
}

func (d *UserPermissionDAO) containsOperation(operations, targetOp string) bool {
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
