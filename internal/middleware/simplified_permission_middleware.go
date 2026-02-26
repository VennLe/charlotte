package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/VennLe/charlotte/internal/service"
	"github.com/VennLe/charlotte/pkg/logger"
	"github.com/VennLe/charlotte/pkg/utils"
)

// SimplifiedPermissionMiddleware 简化版权限中间件
type SimplifiedPermissionMiddleware struct {
	permissionService *service.SimplifiedPermissionService
}

// NewSimplifiedPermissionMiddleware 创建简化版权限中间件实例
func NewSimplifiedPermissionMiddleware(permissionService *service.SimplifiedPermissionService) *SimplifiedPermissionMiddleware {
	return &SimplifiedPermissionMiddleware{
		permissionService: permissionService,
	}
}

// CheckPermission 权限检查中间件（简化版）
func (m *SimplifiedPermissionMiddleware) CheckPermission(resourceType, operation string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前用户ID
		userID := m.getUserID(c)

		// 构建权限检查请求
		req := &service.PermissionCheckRequest{
			UserID:       userID,
			ResourceType: resourceType,
			Operation:    operation,
		}

		// 检查权限
		result, err := m.permissionService.CheckPermission(c.Request.Context(), req)
		if err != nil {
			logger.Error("权限检查失败", zap.Error(err))
			utils.Error(c, http.StatusInternalServerError, "权限检查失败")
			c.Abort()
			return
		}

		if !result.HasPermission {
			logger.Warn("权限不足", 
				zap.Uint("user_id", userID),
				zap.String("resource_type", resourceType),
				zap.String("operation", operation),
				zap.String("reason", result.Reason),
			)
			utils.Error(c, http.StatusForbidden, "权限不足: "+result.Reason)
			c.Abort()
			return
		}

		// 将权限信息存储到上下文中
		c.Set("user_role", result.UserRole)
		c.Set("has_permission", true)

		logger.Debug("权限验证通过",
			zap.Uint("user_id", userID),
			zap.String("role", result.UserRole),
			zap.String("resource_type", resourceType),
			zap.String("operation", operation),
		)

		c.Next()
	}
}

// RequireLogin 要求登录中间件
func (m *SimplifiedPermissionMiddleware) RequireLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := m.getUserID(c)
		if userID == 0 {
			utils.Error(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireRole 要求特定角色权限
func (m *SimplifiedPermissionMiddleware) RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := m.getUserID(c)
		if userID == 0 {
			utils.Error(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}

		userRole, exists := c.Get("user_role")
		if !exists {
			utils.Error(c, http.StatusForbidden, "权限信息缺失")
			c.Abort()
			return
		}

		role := userRole.(string)
		if !m.hasRequiredRole(role, requiredRole) {
			utils.Error(c, http.StatusForbidden, "需要"+requiredRole+"权限")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin 要求管理员权限
func (m *SimplifiedPermissionMiddleware) RequireAdmin() gin.HandlerFunc {
	return m.RequireRole("admin")
}

// RequireSuperAdmin 要求超级管理员权限
func (m *SimplifiedPermissionMiddleware) RequireSuperAdmin() gin.HandlerFunc {
	return m.RequireRole("superadmin")
}

// RequireVIP 要求VIP用户权限
func (m *SimplifiedPermissionMiddleware) RequireVIP() gin.HandlerFunc {
	return m.RequireRole("vip")
}

// GetUserPermissions 获取用户权限信息（用于前端展示）
func (m *SimplifiedPermissionMiddleware) GetUserPermissions() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := m.getUserID(c)
		
		if userID == 0 {
			// 游客权限
			c.JSON(http.StatusOK, gin.H{
				"role":              "guest",
				"is_super_admin":    false,
				"allowed_operations": "read",
				"permissions":       []map[string]interface{}{
					{
						"resource_type": "content",
						"operations":    "read",
						"scope":        "public",
					},
				},
			})
			c.Abort()
			return
		}

		permissions, err := m.permissionService.GetUserPermissions(c.Request.Context(), userID)
		if err != nil {
			logger.Error("获取用户权限失败", zap.Error(err))
			utils.Error(c, http.StatusInternalServerError, "获取权限信息失败")
			c.Abort()
			return
		}

		c.JSON(http.StatusOK, permissions)
		c.Abort()
	}
}

// GetPermissionSummary 获取权限摘要
func (m *SimplifiedPermissionMiddleware) GetPermissionSummary() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := m.getUserID(c)
		
		summary, err := m.permissionService.GetPermissionSummary(c.Request.Context(), userID)
		if err != nil {
			logger.Error("获取权限摘要失败", zap.Error(err))
			utils.Error(c, http.StatusInternalServerError, "获取权限摘要失败")
			c.Abort()
			return
		}

		c.JSON(http.StatusOK, summary)
		c.Abort()
	}
}

// GetAvailableRoles 获取可用角色列表
func (m *SimplifiedPermissionMiddleware) GetAvailableRoles() gin.HandlerFunc {
	return func(c *gin.Context) {
		roles := m.permissionService.GetAvailableRoles()
		c.JSON(http.StatusOK, gin.H{
			"roles": roles,
		})
		c.Abort()
	}
}

// SetUserRole 设置用户角色（管理员专用）
func (m *SimplifiedPermissionMiddleware) SetUserRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只有管理员可以设置用户角色
		userRole, exists := c.Get("user_role")
		if !exists || (userRole != "admin" && userRole != "superadmin") {
			utils.Error(c, http.StatusForbidden, "需要管理员权限")
			c.Abort()
			return
		}

		var req struct {
			UserID uint   `json:"user_id" binding:"required"`
			Role   string `json:"role" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			utils.Error(c, http.StatusBadRequest, "请求参数错误")
			c.Abort()
			return
		}

		err := m.permissionService.SetUserRole(c.Request.Context(), req.UserID, req.Role)
		if err != nil {
			logger.Error("设置用户角色失败", zap.Error(err))
			utils.Error(c, http.StatusInternalServerError, "设置角色失败")
			c.Abort()
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "用户角色设置成功",
			"user_id": req.UserID,
			"role":    req.Role,
		})
		c.Abort()
	}
}

// 辅助方法
func (m *SimplifiedPermissionMiddleware) getUserID(c *gin.Context) uint {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0 // 游客
	}
	return userID.(uint)
}

func (m *SimplifiedPermissionMiddleware) hasRequiredRole(userRole, requiredRole string) bool {
	// 角色权限等级检查
	roleLevels := map[string]int{
		"guest":      1,
		"user":       2,
		"vip":        3,
		"admin":      4,
		"superadmin": 5,
	}

	userLevel, userOk := roleLevels[userRole]
	reqLevel, reqOk := roleLevels[requiredRole]

	if !userOk || !reqOk {
		return false
	}

	return userLevel >= reqLevel
}

// PermissionConfig 权限配置结构
type PermissionConfig struct {
	ResourceType string `json:"resource_type"`
	Operation    string `json:"operation"`
	Description  string `json:"description"`
	RequiredRole string `json:"required_role"`
}

// DefaultPermissions 默认权限配置
var DefaultPermissions = []PermissionConfig{
	// 用户相关权限
	{"user", "read", "查看用户信息", "user"},
	{"user", "write", "修改用户信息", "admin"},
	{"user", "create", "创建用户", "admin"},
	{"user", "delete", "删除用户", "superadmin"},

	// 内容相关权限
	{"content", "read", "查看内容", "guest"},
	{"content", "write", "修改内容", "admin"},
	{"content", "create", "创建内容", "user"},
	{"content", "delete", "删除内容", "admin"},

	// 系统相关权限
	{"system", "read", "查看系统信息", "admin"},
	{"system", "write", "修改系统设置", "superadmin"},
	{"system", "create", "创建系统资源", "superadmin"},
	{"system", "delete", "删除系统资源", "superadmin"},

	// 管理相关权限
	{"admin", "read", "查看管理面板", "admin"},
	{"admin", "write", "管理操作", "admin"},
	{"admin", "create", "创建管理资源", "admin"},
	{"admin", "delete", "删除管理资源", "superadmin"},
}

// GetPermissionConfig 获取权限配置
func (m *SimplifiedPermissionMiddleware) GetPermissionConfig() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"permissions": DefaultPermissions,
			"roles": m.permissionService.GetAvailableRoles(),
		})
		c.Abort()
	}
}