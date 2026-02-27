package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/VennLe/charlotte/internal/middleware"
	"github.com/VennLe/charlotte/internal/model"
	"github.com/VennLe/charlotte/pkg/utils"
)

// PermissionExampleHandler 权限示例处理器
type PermissionExampleHandler struct {
	permissionMiddleware *middleware.PermissionMiddleware
}

// RegisterRoutes 注册权限示例路由
func (h *PermissionExampleHandler) RegisterRoutes(router *gin.RouterGroup) {
	// 权限配置信息
	router.GET("/permissions/config", h.permissionMiddleware.GetUserPermissions())

	// 用户权限信息
	router.GET("/permissions/user", h.permissionMiddleware.GetUserPermissions())

	// 权限示例接口
	exampleGroup := router.Group("/examples")
	{
		// 公开接口（无需登录）
		exampleGroup.GET("/public", h.PublicExample)

		// 需要登录的接口
		exampleGroup.GET("/user-only", h.permissionMiddleware.RequireLogin(), h.UserOnlyExample)

		// 需要VIP权限的接口
		exampleGroup.GET("/vip-only", h.permissionMiddleware.RequireVIP(), h.VIPOnlyExample)

		// 需要管理员权限的接口
		exampleGroup.GET("/admin-only", h.permissionMiddleware.RequireAdmin(), h.AdminOnlyExample)

		// 需要超级管理员权限的接口
		exampleGroup.GET("/super-admin-only", h.permissionMiddleware.RequireSuperAdmin(), h.SuperAdminOnlyExample)

		// 具体权限检查示例
		exampleGroup.GET("/user/:id", h.permissionMiddleware.CheckPermission("user", model.PermissionRead), h.GetUserExample)
		exampleGroup.PUT("/user/:id", h.permissionMiddleware.CheckPermission("user", model.PermissionWrite), h.UpdateUserExample)
		exampleGroup.DELETE("/user/:id", h.permissionMiddleware.CheckPermission("user", model.PermissionDelete), h.DeleteUserExample)

		// 内容权限示例
		exampleGroup.GET("/content/:id", h.permissionMiddleware.CheckPermission("content", model.PermissionRead), h.GetContentExample)
		exampleGroup.POST("/content", h.permissionMiddleware.CheckPermission("content", model.PermissionWrite), h.CreateContentExample)
		exampleGroup.PUT("/content/:id", h.permissionMiddleware.CheckPermission("content", model.PermissionWrite), h.UpdateContentExample)
		exampleGroup.DELETE("/content/:id", h.permissionMiddleware.CheckPermission("content", model.PermissionDelete), h.DeleteContentExample)

		// 系统权限示例
		exampleGroup.GET("/system/info", h.permissionMiddleware.CheckPermission("system", model.PermissionRead), h.GetSystemInfoExample)
		exampleGroup.PUT("/system/settings", h.permissionMiddleware.CheckPermission("system", model.PermissionWrite), h.UpdateSystemSettingsExample)

		// 管理权限示例
		exampleGroup.GET("/admin/dashboard", h.permissionMiddleware.CheckPermission("admin", model.PermissionRead), h.AdminDashboardExample)
		exampleGroup.POST("/admin/users", h.permissionMiddleware.CheckPermission("admin", model.PermissionWrite), h.CreateAdminUserExample)
	}
}

// PublicExample 公开接口示例（无需权限）
func (h *PermissionExampleHandler) PublicExample(c *gin.Context) {
	utils.Success(c, gin.H{
		"message": "这是一个公开接口，无需登录即可访问",
		"data":    "公开数据",
	})
}

// UserOnlyExample 仅限登录用户访问的示例
func (h *PermissionExampleHandler) UserOnlyExample(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")

	utils.Success(c, gin.H{
		"message": "仅限登录用户访问",
		"user_id": userID,
		"role":    userRole,
		"data":    "用户专属数据",
	})
}

// VIPOnlyExample VIP用户专属接口示例
func (h *PermissionExampleHandler) VIPOnlyExample(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")

	utils.Success(c, gin.H{
		"message": "VIP用户专属接口",
		"user_id": userID,
		"role":    userRole,
		"data":    "VIP专属内容",
	})
}

// AdminOnlyExample 管理员专属接口示例
func (h *PermissionExampleHandler) AdminOnlyExample(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")

	utils.Success(c, gin.H{
		"message": "管理员专属接口",
		"user_id": userID,
		"role":    userRole,
		"data":    "管理面板数据",
	})
}

// SuperAdminOnlyExample 超级管理员专属接口示例
func (h *PermissionExampleHandler) SuperAdminOnlyExample(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")

	utils.Success(c, gin.H{
		"message": "超级管理员专属接口",
		"user_id": userID,
		"role":    userRole,
		"data":    "系统级配置数据",
	})
}

// GetUserExample 获取用户信息（需要读取权限）
func (h *PermissionExampleHandler) GetUserExample(c *gin.Context) {
	userID := c.Param("id")
	permissionResult, _ := c.Get("permission_result")

	utils.Success(c, gin.H{
		"message":    "获取用户信息成功",
		"user_id":    userID,
		"permission": permissionResult,
		"data": gin.H{
			"username": "示例用户",
			"email":    "user@example.com",
			"role":     "user",
		},
	})
}

// UpdateUserExample 更新用户信息（需要写入权限）
func (h *PermissionExampleHandler) UpdateUserExample(c *gin.Context) {
	userID := c.Param("id")

	utils.Success(c, gin.H{
		"message": "更新用户信息成功",
		"user_id": userID,
		"data":    "用户信息已更新",
	})
}

// DeleteUserExample 删除用户（需要删除权限）
func (h *PermissionExampleHandler) DeleteUserExample(c *gin.Context) {
	userID := c.Param("id")

	utils.Success(c, gin.H{
		"message": "删除用户成功",
		"user_id": userID,
		"data":    "用户已删除",
	})
}

// GetContentExample 获取内容（需要内容读取权限）
func (h *PermissionExampleHandler) GetContentExample(c *gin.Context) {
	contentID := c.Param("id")

	utils.Success(c, gin.H{
		"message":    "获取内容成功",
		"content_id": contentID,
		"data": gin.H{
			"title":   "示例内容标题",
			"content": "这是示例内容",
			"author":  "示例作者",
		},
	})
}

// CreateContentExample 创建内容（需要内容创建权限）
func (h *PermissionExampleHandler) CreateContentExample(c *gin.Context) {
	utils.Success(c, gin.H{
		"message": "创建内容成功",
		"data": gin.H{
			"id":      1,
			"title":   "新内容标题",
			"content": "新创建的内容",
		},
	})
}

// UpdateContentExample 更新内容（需要内容写入权限）
func (h *PermissionExampleHandler) UpdateContentExample(c *gin.Context) {
	contentID := c.Param("id")

	utils.Success(c, gin.H{
		"message":    "更新内容成功",
		"content_id": contentID,
		"data":       "内容已更新",
	})
}

// DeleteContentExample 删除内容（需要内容删除权限）
func (h *PermissionExampleHandler) DeleteContentExample(c *gin.Context) {
	contentID := c.Param("id")

	utils.Success(c, gin.H{
		"message":    "删除内容成功",
		"content_id": contentID,
		"data":       "内容已删除",
	})
}

// GetSystemInfoExample 获取系统信息（需要系统读取权限）
func (h *PermissionExampleHandler) GetSystemInfoExample(c *gin.Context) {
	utils.Success(c, gin.H{
		"message": "获取系统信息成功",
		"data": gin.H{
			"system_name":   "Charlotte系统",
			"version":       "1.0.0",
			"uptime":        "7天12小时",
			"active_users":  1000,
			"system_status": "正常",
		},
	})
}

// UpdateSystemSettingsExample 更新系统设置（需要系统写入权限）
func (h *PermissionExampleHandler) UpdateSystemSettingsExample(c *gin.Context) {
	utils.Success(c, gin.H{
		"message": "更新系统设置成功",
		"data":    "系统设置已更新",
	})
}

// AdminDashboardExample 管理面板（需要管理读取权限）
func (h *PermissionExampleHandler) AdminDashboardExample(c *gin.Context) {
	utils.Success(c, gin.H{
		"message": "获取管理面板数据成功",
		"data": gin.H{
			"total_users":         1500,
			"active_users":        1200,
			"today_registrations": 25,
			"system_health":       "良好",
			"recent_activities": []string{
				"用户A登录",
				"用户B创建内容",
				"用户C更新信息",
			},
		},
	})
}

// CreateAdminUserExample 创建管理用户（需要管理创建权限）
func (h *PermissionExampleHandler) CreateAdminUserExample(c *gin.Context) {
	utils.Success(c, gin.H{
		"message": "创建管理用户成功",
		"data": gin.H{
			"id":       1001,
			"username": "admin_user",
			"role":     "admin",
			"email":    "admin@example.com",
		},
	})
}

// PermissionUsageExample 权限使用示例说明
func (h *PermissionExampleHandler) PermissionUsageExample(c *gin.Context) {
	examples := gin.H{
		"说明": "以下是权限系统的使用示例",
		"权限级别": []string{
			"游客（Guest）: 只能查看公开内容",
			"普通用户（User）: 可以查看和操作自己的内容",
			"VIP用户（VIP）: 可以查看所有内容，但不能修改他人内容",
			"管理员（Admin）: 可以查看和修改所有内容，但不能删除",
			"超级管理员（SuperAdmin）: 拥有所有权限",
		},
		"权限类型": []string{
			"read: 读取权限",
			"write: 写入权限",
			"create: 创建权限",
			"delete: 删除权限",
		},
		"资源类型": []string{
			"user: 用户资源",
			"content: 内容资源",
			"system: 系统资源",
			"admin: 管理资源",
		},
		"使用方式": []string{
			"1. 在路由中使用中间件进行权限检查",
			"2. 在业务逻辑中调用权限服务进行细粒度检查",
			"3. 通过用户组和标签进行灵活的权限管理",
		},
	}

	utils.Success(c, examples)
}