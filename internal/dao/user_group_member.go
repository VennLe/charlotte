package dao

import (
	"context"

	"gorm.io/gorm"

	"github.com/VennLe/charlotte/internal/model"
)

// UserGroupMemberDAO 用户组成员关联数据访问对象
type UserGroupMemberDAO struct {
	*BaseDAOImpl[model.UserGroupMember, uint]
}

// NewUserGroupMemberDAO 创建用户组成员关联DAO实例
func NewUserGroupMemberDAO(db *gorm.DB) *UserGroupMemberDAO {
	return &UserGroupMemberDAO{
		BaseDAOImpl: NewBaseDAO[model.UserGroupMember, uint](db),
	}
}

// AddUserToGroup 添加用户到用户组
func (d *UserGroupMemberDAO) AddUserToGroup(ctx context.Context, userID, groupID uint) error {
	// 检查是否已经存在
	exists, err := d.Exists(ctx, map[string]interface{}{
		"user_id":       userID,
		"user_group_id": groupID,
	})
	if err != nil {
		return err
	}
	if exists {
		return nil // 已经存在，无需重复添加
	}

	member := &model.UserGroupMember{
		UserID:      userID,
		UserGroupID: groupID,
		Status:      1, // 正常状态
	}
	return d.Create(ctx, member)
}

// RemoveUserFromGroup 从用户组移除用户
func (d *UserGroupMemberDAO) RemoveUserFromGroup(ctx context.Context, userID, groupID uint) error {
	member, err := d.GetOne(ctx, map[string]interface{}{
		"user_id":       userID,
		"user_group_id": groupID,
	})
	if err != nil {
		return err
	}
	return d.Delete(ctx, member.ID)
}

// GetUserGroups 获取用户所在的用户组
func (d *UserGroupMemberDAO) GetUserGroups(ctx context.Context, userID uint) ([]*model.UserGroupMember, error) {
	return d.GetMany(ctx, map[string]interface{}{"user_id": userID, "status": 1})
}

// GetGroupMembers 获取用户组的成员
func (d *UserGroupMemberDAO) GetGroupMembers(ctx context.Context, groupID uint) ([]*model.UserGroupMember, error) {
	return d.GetMany(ctx, map[string]interface{}{"user_group_id": groupID, "status": 1})
}

// IsUserInGroup 检查用户是否在指定用户组中
func (d *UserGroupMemberDAO) IsUserInGroup(ctx context.Context, userID, groupID uint) (bool, error) {
	return d.Exists(ctx, map[string]interface{}{
		"user_id":       userID,
		"user_group_id": groupID,
		"status":        1,
	})
}

// UpdateMemberStatus 更新成员状态
func (d *UserGroupMemberDAO) UpdateMemberStatus(ctx context.Context, userID, groupID uint, status int) error {
	member, err := d.GetOne(ctx, map[string]interface{}{
		"user_id":       userID,
		"user_group_id": groupID,
	})
	if err != nil {
		return err
	}
	return d.Update(ctx, member.ID, map[string]interface{}{"status": status})
}

// GetActiveGroupMembers 获取活跃的组成员
func (d *UserGroupMemberDAO) GetActiveGroupMembers(ctx context.Context, groupID uint) ([]*model.UserGroupMember, error) {
	return d.GetMany(ctx, map[string]interface{}{
		"user_group_id": groupID,
		"status":        1,
	})
}

// GetUserGroupCount 获取用户的用户组数量
func (d *UserGroupMemberDAO) GetUserGroupCount(ctx context.Context, userID uint) (int64, error) {
	return d.Count(ctx, map[string]interface{}{"user_id": userID, "status": 1})
}

// GetGroupMemberCount 获取用户组的成员数量
func (d *UserGroupMemberDAO) GetGroupMemberCount(ctx context.Context, groupID uint) (int64, error) {
	return d.Count(ctx, map[string]interface{}{"user_group_id": groupID, "status": 1})
}

// GetUsersByGroup 获取用户组的所有用户ID
func (d *UserGroupMemberDAO) GetUsersByGroup(ctx context.Context, groupID uint) ([]uint, error) {
	members, err := d.GetGroupMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}

	userIDs := make([]uint, len(members))
	for i, member := range members {
		userIDs[i] = member.UserID
	}

	return userIDs, nil
}

// GetGroupsByUser 获取用户的所有用户组ID
func (d *UserGroupMemberDAO) GetGroupsByUser(ctx context.Context, userID uint) ([]uint, error) {
	members, err := d.GetUserGroups(ctx, userID)
	if err != nil {
		return nil, err
	}

	groupIDs := make([]uint, len(members))
	for i, member := range members {
		groupIDs[i] = member.UserGroupID
	}

	return groupIDs, nil
}

// AssignDefaultGroups 为用户分配默认用户组（新用户注册时调用）
func (d *UserGroupMemberDAO) AssignDefaultGroups(ctx context.Context, userID uint, userRole string) error {
	groupDAO := NewUserGroupDAO(d.DB)
	defaultGroups, err := groupDAO.GetDefaultGroups(ctx)
	if err != nil {
		return err
	}

	// 根据用户角色分配默认组
	for _, group := range defaultGroups {
		shouldAssign := false

		switch userRole {
		case model.RoleSuperAdmin:
			shouldAssign = group.Name == "超级管理员组"
		case model.RoleAdmin:
			shouldAssign = group.Name == "管理员组"
		case model.RoleVIP:
			shouldAssign = group.Name == "VIP用户组"
		case model.RoleUser:
			shouldAssign = group.Name == "普通用户组"
		case model.RoleGuest:
			shouldAssign = group.Name == "游客组"
		}

		if shouldAssign {
			if err := d.AddUserToGroup(ctx, userID, group.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

// RemoveUserFromAllGroups 从所有用户组中移除用户
func (d *UserGroupMemberDAO) RemoveUserFromAllGroups(ctx context.Context, userID uint) error {
	members, err := d.GetUserGroups(ctx, userID)
	if err != nil {
		return err
	}

	for _, member := range members {
		if err := d.Delete(ctx, member.ID); err != nil {
			return err
		}
	}

	return nil
}

// TransferUserGroups 转移用户的用户组（用户角色变更时调用）
func (d *UserGroupMemberDAO) TransferUserGroups(ctx context.Context, userID uint, newRole string) error {
	// 先移除所有现有组
	if err := d.RemoveUserFromAllGroups(ctx, userID); err != nil {
		return err
	}

	// 分配新的默认组
	return d.AssignDefaultGroups(ctx, userID, newRole)
}