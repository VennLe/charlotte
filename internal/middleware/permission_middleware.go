package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/VennLe/charlotte/internal/model"
	"github.com/VennLe/charlotte/internal/service"
	"github.com/VennLe/charlotte/pkg/logger"
	"github.com/VennLe/charlotte/pkg/utils"
)

// PermissionMiddleware 权限中间件
type PermissionMiddleware struct {
	permissionService *service.PermissionService
}

// NewPermissionMiddleware 创建权限中间件实例
func NewPermissionMiddleware(permissionService *service.PermissionService) *PermissionMiddleware {
	return &PermissionMiddleware{
		permissionService: permissionService,
	}
}

// CheckPermission 权限检查中间件
func (m *PermissionMiddleware) CheckPermission(resourceType, operation string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			// 未登录用户视为游客
			userID = uint(0)
		}

		// 构建权限检查请求
		req := &model.PermissionCheckRequest{
			UserID:       userID.(uint),
			ResourceType: resourceType,
			Operation:    operation,
			ResourceID:   m.extractResourceID(c),
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
				zap.Uint("user_id", userID.(uint)),
				zap.String("resource_type", resourceType),
				zap.String("operation", operation),
				zap.String("reason", result.Reason),
			)
			utils.Error(c, http.StatusForbidden, "权限不足: "+result.Reason)
			c.Abort()
			return
		}

		// 将权限信息存储到上下文中
		c.Set("permission_result", result)
		c.Set("user_role", result.UserRole)
		c.Set("allowed_operations", result.AllowedOps)

		logger.Debug("权限验证通过",
			zap.Uint("user_id", userID.(uint)),
			zap.String("role", result.UserRole),
			zap.String("resource_type", resourceType),
			zap.String("operation", operation),
		)

		c.Next()
	}
}

// RequireLogin 要求登录中间件
func (m *PermissionMiddleware) RequireLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists || userID.(uint) == 0 {
			utils.Error(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireAdmin 要求管理员权限
func (m *PermissionMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			utils.Error(c, http.StatusForbidden, "权限信息缺失")
			c.Abort()
			return
		}

		role := userRole.(string)
		if role != model.RoleAdmin && role != model.RoleSuperAdmin {
			utils.Error(c, http.StatusForbidden, "需要管理员权限")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireSuperAdmin 要求超级管理员权限
func (m *PermissionMiddleware) RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			utils.Error(c, http.StatusForbidden, "权限信息缺失")
			c.Abort()
			return
		}

		role := userRole.(string)
		if role != model.RoleSuperAdmin {
			utils.Error(c, http.StatusForbidden, "需要超级管理员权限")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireVIP 要求VIP用户权限
func (m *PermissionMiddleware) RequireVIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			utils.Error(c, http.StatusForbidden, "权限信息缺失")
			c.Abort()
			return
		}

		role := userRole.(string)
		if role != model.RoleVIP && role != model.RoleAdmin && role != model.RoleSuperAdmin {
			utils.Error(c, http.StatusForbidden, "需要VIP用户权限")
			c.Abort()
			return
		}

		c.Next()
	}
}

// CheckResourceOwnership 检查资源所有权
func (m *PermissionMiddleware) CheckResourceOwnership(resourceType string, ownerIDExtractor func(c *gin.Context) uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists || userID.(uint) == 0 {
			utils.Error(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}

		ownerID := ownerIDExtractor(c)
		if ownerID == 0 {
			utils.Error(c, http.StatusNotFound, "资源不存在")
			c.Abort()
			return
		}

		// 检查是否是资源所有者
		if userID.(uint) != ownerID {
			// 如果不是所有者，检查是否有权限操作他人的资源
			req := &model.PermissionCheckRequest{
				UserID:       userID.(uint),
				ResourceType: resourceType,
				Operation:    model.PermissionWrite,
				ResourceID:   ownerID,
			}

			result, err := m.permissionService.CheckPermission(c.Request.Context(), req)
			if err != nil || !result.HasPermission {
				utils.Error(c, http.StatusForbidden, "无权操作他人资源")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// GetUserPermissions 获取用户权限信息（用于前端展示）
func (m *PermissionMiddleware) GetUserPermissions() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists || userID.(uint) == 0 {
			// 游客权限
			c.JSON(http.StatusOK, gin.H{
				"role":               model.RoleGuest,
				"is_super_admin":     false,
				"allowed_operations": model.PermissionRead,
				"groups":             []interface{}{},
			})
			c.Abort()
			return
		}

		permissions, err := m.permissionService.GetUserPermissions(c.Request.Context(), userID.(uint))
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

// extractResourceID 从请求中提取资源ID
func (m *PermissionMiddleware) extractResourceID(c *gin.Context) uint {
	// 从URL参数中提取ID
	if id := c.Param("id"); id != "" {
		// 这里需要根据实际情况解析ID
		// 简化处理，实际使用时需要具体实现
		return 1 // 示例值
	}

	// 从查询参数中提取ID
	if id := c.Query("id"); id != "" {
		// 这里需要根据实际情况解析ID
		return 1 // 示例值
	}

	return 0
}