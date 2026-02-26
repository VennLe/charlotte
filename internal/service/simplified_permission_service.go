package service

import (
	"context"
	"errors"

	"github.com/VennLe/charlotte/internal/dao"
	"github.com/VennLe/charlotte/internal/model"
)

// SimplifiedPermissionService 简化版权限服务
// 使用统一权限DAO，简化权限检查逻辑

type SimplifiedPermissionService struct {
	userDAO            *dao.UserDAO
	permissionDAO      *dao.UnifiedPermissionDAO
}

// NewSimplifiedPermissionService 创建简化版权限服务实例
func NewSimplifiedPermissionService(
	userDAO *dao.UserDAO,
	permissionDAO *dao.UnifiedPermissionDAO,
) *SimplifiedPermissionService {
	return &SimplifiedPermissionService{
		userDAO:       userDAO,
		permissionDAO: permissionDAO,
	}
}

// PermissionCheckRequest 权限检查请求（简化版）
type PermissionCheckRequest struct {
	UserID       uint   `json:"user_id"`
	ResourceType string `json:"resource_type"`
	Operation    string `json:"operation"`
}

// PermissionCheckResult 权限检查结果（简化版）
type PermissionCheckResult struct {
	HasPermission bool   `json:"has_permission"`
	Reason       string `json:"reason,omitempty"`
	UserRole     string `json:"user_role"`
}

// CheckPermission 检查用户权限（简化版）
func (s *SimplifiedPermissionService) CheckPermission(ctx context.Context, req *PermissionCheckRequest) (*PermissionCheckResult, error) {
	// 获取用户信息
	user, err := s.userDAO.GetByID(ctx, req.UserID)
	if err != nil {
		// 用户不存在，视为游客
		return &PermissionCheckResult{
			HasPermission: s.checkGuestPermission(req.ResourceType, req.Operation),
			Reason:       "用户不存在，按游客权限处理",
			UserRole:     model.RoleGuest,
		}, nil
	}

	// 检查用户状态
	if user.Status != 1 {
		return &PermissionCheckResult{
			HasPermission: false,
			Reason:       "用户已被禁用",
			UserRole:     user.Role,
		}, nil
	}

	// 使用统一权限DAO检查权限
	hasPermission, err := s.permissionDAO.CheckPermission(ctx, req.UserID, req.ResourceType, req.Operation)
	if err != nil {
		return nil, err
	}

	userRole, err := s.permissionDAO.GetUserRole(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	if hasPermission {
		return &PermissionCheckResult{
			HasPermission: true,
			Reason:       "权限验证通过",
			UserRole:     userRole,
		}, nil
	}

	return &PermissionCheckResult{
		HasPermission: false,
		Reason:       "权限不足",
		UserRole:     userRole,
	}, nil
}

// SetUserRole 设置用户角色
func (s *SimplifiedPermissionService) SetUserRole(ctx context.Context, userID uint, role string) error {
	// 验证角色是否有效
	if !s.isValidRole(role) {
		return errors.New("无效的用户角色: " + role)
	}

	return s.permissionDAO.SetUserRole(ctx, userID, role)
}

// GetUserPermissions 获取用户权限信息
func (s *SimplifiedPermissionService) GetUserPermissions(ctx context.Context, userID uint) (map[string]interface{}, error) {
	user, err := s.userDAO.GetByID(ctx, userID)
	if err != nil {
		// 返回游客权限信息
		return map[string]interface{}{
			"user_id":   0,
			"role":      model.RoleGuest,
			"is_active": false,
			"permissions": []map[string]interface{}{
				{
					"resource_type": "content",
					"operations":    model.PermissionRead,
					"scope":         "public",
				},
			},
		}, nil
	}

	// 获取用户角色
	role, err := s.permissionDAO.GetUserRole(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 获取角色权限
	permissions, err := s.permissionDAO.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"user_id":     user.ID,
		"username":    user.Username,
		"role":        role,
		"is_active":   user.Status == 1,
		"is_super_admin": user.IsSuperAdmin,
		"permissions": permissions["permissions"],
	}

	return result, nil
}

// InitializeDefaultPermissions 初始化默认权限配置
func (s *SimplifiedPermissionService) InitializeDefaultPermissions(ctx context.Context) error {
	return s.permissionDAO.InitializeDefaultPermissions(ctx)
}

// AddRolePermission 添加角色权限
func (s *SimplifiedPermissionService) AddRolePermission(ctx context.Context, role, resourceType, operations, scope string) error {
	if !s.isValidRole(role) {
		return errors.New("无效的用户角色: " + role)
	}

	return s.permissionDAO.AddRolePermission(ctx, role, resourceType, operations, scope)
}

// GetRolePermissions 获取角色权限
func (s *SimplifiedPermissionService) GetRolePermissions(ctx context.Context, role string) ([]map[string]interface{}, error) {
	if !s.isValidRole(role) {
		return nil, errors.New("无效的用户角色: " + role)
	}

	permissions, err := s.permissionDAO.GetRolePermissions(ctx, role)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, perm := range permissions {
		result = append(result, map[string]interface{}{
			"resource_type": perm.ResourceType,
			"operations":    perm.Operations,
			"scope":         perm.Scope,
		})
	}

	return result, nil
}

// GetAvailableRoles 获取可用角色列表
func (s *SimplifiedPermissionService) GetAvailableRoles() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        model.RoleSuperAdmin,
			"description": "超级管理员，拥有所有权限",
			"level":       model.PermissionLevelHigh,
		},
		{
			"name":        model.RoleAdmin,
			"description": "普通管理员，可以查看和修改但不能删除",
			"level":       model.PermissionLevelHigh,
		},
		{
			"name":        model.RoleVIP,
			"description": "VIP用户，可以查看所有内容",
			"level":       model.PermissionLevelMedium,
		},
		{
			"name":        model.RoleUser,
			"description": "普通用户，只能查看自己的内容",
			"level":       model.PermissionLevelLow,
		},
		{
			"name":        model.RoleGuest,
			"description": "游客，只能查看公开内容",
			"level":       model.PermissionLevelLow,
		},
	}
}

// 辅助方法
func (s *SimplifiedPermissionService) checkGuestPermission(resourceType, operation string) bool {
	// 游客只能查看公开内容
	return resourceType == "content" && operation == model.PermissionRead
}

func (s *SimplifiedPermissionService) isValidRole(role string) bool {
	validRoles := []string{
		model.RoleSuperAdmin,
		model.RoleAdmin,
		model.RoleVIP,
		model.RoleUser,
		model.RoleGuest,
	}

	for _, validRole := range validRoles {
		if role == validRole {
			return true
		}
	}

	return false
}

// GetPermissionSummary 获取权限摘要
func (s *SimplifiedPermissionService) GetPermissionSummary(ctx context.Context, userID uint) (map[string]interface{}, error) {
	_, err := s.userDAO.GetByID(ctx, userID)
	if err != nil {
		return map[string]interface{}{
			"role":        model.RoleGuest,
			"can_read":    true,
			"can_write":   false,
			"can_delete":  false,
			"description": "游客权限",
		}, nil
	}

	role, err := s.permissionDAO.GetUserRole(ctx, userID)
	if err != nil {
		return nil, err
	}

	var canRead, canWrite, canDelete bool

	switch role {
	case model.RoleSuperAdmin:
		canRead, canWrite, canDelete = true, true, true
	case model.RoleAdmin:
		canRead, canWrite, canDelete = true, true, false
	case model.RoleVIP:
		canRead, canWrite, canDelete = true, false, false
	case model.RoleUser:
		canRead, canWrite, canDelete = true, false, false
	case model.RoleGuest:
		canRead, canWrite, canDelete = true, false, false
	}

	return map[string]interface{}{
		"role":        role,
		"can_read":    canRead,
		"can_write":   canWrite,
		"can_delete":  canDelete,
		"description": s.getRoleDescription(role),
	}, nil
}

func (s *SimplifiedPermissionService) getRoleDescription(role string) string {
	switch role {
	case model.RoleSuperAdmin:
		return "超级管理员，拥有所有权限"
	case model.RoleAdmin:
		return "普通管理员，可以查看和修改但不能删除"
	case model.RoleVIP:
		return "VIP用户，可以查看所有内容"
	case model.RoleUser:
		return "普通用户，只能查看自己的内容"
	case model.RoleGuest:
		return "游客，只能查看公开内容"
	default:
		return "未知角色"
	}
}