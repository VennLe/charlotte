package service

import (
	"context"
	"strings"

	"github.com/VennLe/charlotte/internal/dao"
	"github.com/VennLe/charlotte/internal/model"
)

// PermissionService 权限管理服务
type PermissionService struct {
	userDAO          *dao.UserDAO
	userGroupDAO      *dao.UserGroupDAO
	permissionTagDAO  *dao.PermissionTagDAO
	groupMemberDAO    *dao.UserGroupMemberDAO
	groupPermissionDAO *dao.UserGroupPermissionDAO
	userPermissionDAO *dao.UserPermissionDAO
}

// NewPermissionService 创建权限服务实例
func NewPermissionService(
	userDAO *dao.UserDAO,
	userGroupDAO *dao.UserGroupDAO,
	permissionTagDAO *dao.PermissionTagDAO,
	groupMemberDAO *dao.UserGroupMemberDAO,
	groupPermissionDAO *dao.UserGroupPermissionDAO,
	userPermissionDAO *dao.UserPermissionDAO,
) *PermissionService {
	return &PermissionService{
		userDAO:          userDAO,
		userGroupDAO:      userGroupDAO,
		permissionTagDAO:  permissionTagDAO,
		groupMemberDAO:    groupMemberDAO,
		groupPermissionDAO: groupPermissionDAO,
		userPermissionDAO: userPermissionDAO,
	}
}

// CheckPermission 检查用户权限
func (s *PermissionService) CheckPermission(ctx context.Context, req *model.PermissionCheckRequest) (*model.PermissionCheckResult, error) {
	// 获取用户信息
	user, err := s.userDAO.GetByID(ctx, req.UserID)
	if err != nil {
		return &model.PermissionCheckResult{
			HasPermission: false,
			Reason:       "用户不存在",
			UserRole:     model.RoleGuest,
		}, nil
	}

	// 检查用户状态
	if user.Status != 1 {
		return &model.PermissionCheckResult{
			HasPermission: false,
			Reason:       "用户已被禁用",
			UserRole:     user.Role,
		}, nil
	}

	// 超级管理员拥有所有权限
	if user.IsSuperAdmin {
		return &model.PermissionCheckResult{
			HasPermission: true,
			Reason:       "超级管理员拥有所有权限",
			UserRole:     model.RoleSuperAdmin,
			AllowedOps:   model.PermissionAll,
		}, nil
	}

	// 根据角色检查基础权限
	if !s.hasBasePermission(user.Role, req.Operation) {
		return &model.PermissionCheckResult{
			HasPermission: false,
			Reason:       "角色权限不足",
			UserRole:     user.Role,
		}, nil
	}

	// 检查用户组权限
	groupPermissions, err := s.getUserGroupPermissions(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	// 检查特殊权限
	specialPermissions, err := s.getUserSpecialPermissions(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	// 合并权限检查
	if s.checkPermissions(groupPermissions, specialPermissions, req) {
		return &model.PermissionCheckResult{
			HasPermission: true,
			Reason:       "权限验证通过",
			UserRole:     user.Role,
			AllowedOps:   s.getAllowedOperations(user.Role, groupPermissions, specialPermissions),
		}, nil
	}

	return &model.PermissionCheckResult{
		HasPermission: false,
		Reason:       "权限不足",
		UserRole:     user.Role,
	}, nil
}

// hasBasePermission 检查基础角色权限
func (s *PermissionService) hasBasePermission(role, operation string) bool {
	switch role {
	case model.RoleSuperAdmin:
		return true // 超级管理员拥有所有权限
	case model.RoleAdmin:
		// 普通管理员不能删除
		return operation != model.PermissionDelete
	case model.RoleVIP, model.RoleUser:
		// VIP用户和普通用户只能查看
		return operation == model.PermissionRead
	case model.RoleGuest:
		// 游客没有权限
		return false
	default:
		return false
	}
}

// getUserGroupPermissions 获取用户组权限
func (s *PermissionService) getUserGroupPermissions(ctx context.Context, userID uint) ([]*model.UserGroupPermission, error) {
	// 获取用户所在的用户组
	groupMembers, err := s.groupMemberDAO.GetUserGroups(ctx, userID)
	if err != nil {
		return nil, err
	}

	var allPermissions []*model.UserGroupPermission
	for _, member := range groupMembers {
		if member.Status == 1 { // 只处理正常状态的组成员
			permissions, err := s.groupPermissionDAO.GetGroupPermissions(ctx, member.UserGroupID)
			if err != nil {
				return nil, err
			}
			allPermissions = append(allPermissions, permissions...)
		}
	}

	return allPermissions, nil
}

// getUserSpecialPermissions 获取用户特殊权限
func (s *PermissionService) getUserSpecialPermissions(ctx context.Context, userID uint) ([]*model.UserPermission, error) {
	return s.userPermissionDAO.GetActiveUserPermissions(ctx, userID)
}

// checkPermissions 检查权限是否满足要求
func (s *PermissionService) checkPermissions(
	groupPermissions []*model.UserGroupPermission,
	specialPermissions []*model.UserPermission,
	req *model.PermissionCheckRequest,
) bool {
	// 首先检查特殊权限（优先级最高）
	for _, perm := range specialPermissions {
		if s.matchPermission(perm, req) {
			return perm.IsGrant
		}
	}

	// 然后检查组权限
	for _, perm := range groupPermissions {
		if s.matchPermissionFromGroup(perm, req) {
			return true
		}
	}

	return false
}

// matchPermission 匹配权限
func (s *PermissionService) matchPermission(perm *model.UserPermission, req *model.PermissionCheckRequest) bool {
	if perm.ResourceType != req.ResourceType {
		return false
	}

	if !s.containsOperation(perm.Operations, req.Operation) {
		return false
	}

	// 检查资源范围
	return s.checkResourceScope(perm.ResourceScope, req)
}

// matchPermissionFromGroup 匹配组权限
func (s *PermissionService) matchPermissionFromGroup(perm *model.UserGroupPermission, req *model.PermissionCheckRequest) bool {
	if perm.ResourceType != req.ResourceType {
		return false
	}

	if !s.containsOperation(perm.Operations, req.Operation) {
		return false
	}

	// 检查资源范围
	return s.checkResourceScope(perm.ResourceScope, req)
}

// containsOperation 检查是否包含指定操作
func (s *PermissionService) containsOperation(operations, targetOp string) bool {
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

// checkResourceScope 检查资源范围
func (s *PermissionService) checkResourceScope(scope string, req *model.PermissionCheckRequest) bool {
	switch scope {
	case "all":
		return true // 所有资源
	case "own":
		// 需要检查资源所有者
		// 这里需要根据具体业务逻辑实现
		return true
	case "system":
		// 系统级资源
		return true
	default:
		return false
	}
}

// getAllowedOperations 获取允许的操作列表
func (s *PermissionService) getAllowedOperations(
	role string,
	groupPermissions []*model.UserGroupPermission,
	specialPermissions []*model.UserPermission,
) string {
	if role == model.RoleSuperAdmin {
		return model.PermissionAll
	}

	var allowedOps []string

	// 添加基础角色权限
	switch role {
	case model.RoleAdmin:
		allowedOps = append(allowedOps, model.PermissionRead, model.PermissionWrite)
	case model.RoleVIP, model.RoleUser:
		allowedOps = append(allowedOps, model.PermissionRead)
	}

	// 添加组权限
	for _, perm := range groupPermissions {
		if perm.Operations == model.PermissionAll {
			return model.PermissionAll
		}
		ops := strings.Split(perm.Operations, ",")
		allowedOps = append(allowedOps, ops...)
	}

	// 添加特殊权限
	for _, perm := range specialPermissions {
		if perm.IsGrant {
			if perm.Operations == model.PermissionAll {
				return model.PermissionAll
			}
			ops := strings.Split(perm.Operations, ",")
			allowedOps = append(allowedOps, ops...)
		}
	}

	// 去重并返回
	return s.removeDuplicates(allowedOps)
}

// removeDuplicates 去除重复项
func (s *PermissionService) removeDuplicates(items []string) string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return strings.Join(result, ",")
}

// GetUserPermissions 获取用户完整权限信息
func (s *PermissionService) GetUserPermissions(ctx context.Context, userID uint) (map[string]interface{}, error) {
	user, err := s.userDAO.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"user_id":    user.ID,
		"username":   user.Username,
		"role":       user.Role,
		"is_super_admin": user.IsSuperAdmin,
		"permission_level": user.PermissionLevel,
	}

	// 获取用户组信息
	groupMembers, err := s.groupMemberDAO.GetUserGroups(ctx, userID)
	if err == nil {
		var groups []map[string]interface{}
		for _, member := range groupMembers {
			groups = append(groups, map[string]interface{}{
				"group_id":   member.UserGroupID,
				"group_name": member.UserGroup.Name,
				"joined_at":  member.JoinedAt,
				"status":     member.Status,
			})
		}
		result["groups"] = groups
	}

	// 获取权限信息
	allowedOps := s.getAllowedOperations(user.Role, nil, nil)
	result["allowed_operations"] = allowedOps

	return result, nil
}