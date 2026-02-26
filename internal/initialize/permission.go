package initialize

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/VennLe/charlotte/internal/dao"
	"github.com/VennLe/charlotte/internal/model"
	"github.com/VennLe/charlotte/pkg/logger"
)

// InitPermissionSystem 初始化权限系统
func InitPermissionSystem(db interface{}) error {
	gormDB, ok := db.(*gorm.DB)
	if !ok {
		return fmt.Errorf("无效的数据库连接")
	}

	logger.Info("开始初始化权限系统...")

	// 创建DAO实例
	userGroupDAO := dao.NewUserGroupDAO(gormDB)
	permissionTagDAO := dao.NewPermissionTagDAO(gormDB)
	groupPermissionDAO := dao.NewUserGroupPermissionDAO(gormDB)

	ctx := context.Background()

	// 1. 创建默认用户组
	if err := createDefaultUserGroups(ctx, userGroupDAO); err != nil {
		return fmt.Errorf("创建默认用户组失败: %w", err)
	}

	// 2. 创建默认权限标签
	if err := createDefaultPermissionTags(ctx, permissionTagDAO); err != nil {
		return fmt.Errorf("创建默认权限标签失败: %w", err)
	}

	// 3. 配置默认权限
	if err := configureDefaultPermissions(ctx, groupPermissionDAO); err != nil {
		return fmt.Errorf("配置默认权限失败: %w", err)
	}

	logger.Info("权限系统初始化完成")
	return nil
}

// createDefaultUserGroups 创建默认用户组
func createDefaultUserGroups(ctx context.Context, userGroupDAO *dao.UserGroupDAO) error {
	defaultGroups := []*model.UserGroup{
		{
			Name:        "超级管理员组",
			Description: "系统超级管理员，拥有所有权限",
			Level:       model.PermissionLevelSuperAdmin,
			Status:      1,
			IsDefault:   true,
		},
		{
			Name:        "管理员组",
			Description: "系统管理员，可以管理大部分内容但不能删除",
			Level:       model.PermissionLevelAdmin,
			Status:      1,
			IsDefault:   true,
		},
		{
			Name:        "VIP用户组",
			Description: "VIP用户，可以查看所有内容",
			Level:       model.PermissionLevelVIP,
			Status:      1,
			IsDefault:   true,
		},
		{
			Name:        "普通用户组",
			Description: "普通注册用户，只能查看和操作自己的内容",
			Level:       model.PermissionLevelUser,
			Status:      1,
			IsDefault:   true,
		},
		{
			Name:        "游客组",
			Description: "未登录用户，只能查看公开内容",
			Level:       model.PermissionLevelGuest,
			Status:      1,
			IsDefault:   true,
		},
	}

	for _, group := range defaultGroups {
		exists, err := userGroupDAO.ExistsByName(ctx, group.Name)
		if err != nil {
			return err
		}

		if !exists {
			if err := userGroupDAO.Create(ctx, group); err != nil {
				return err
			}
			logger.Info("创建用户组成功", zap.String("group", group.Name))
		} else {
			logger.Debug("用户组已存在", zap.String("group", group.Name))
		}
	}

	return nil
}

// createDefaultPermissionTags 创建默认权限标签
func createDefaultPermissionTags(ctx context.Context, permissionTagDAO *dao.PermissionTagDAO) error {
	defaultTags := []*model.PermissionTag{
		// 用户相关权限
		{
			Tag:         "user_read",
			Name:        "用户读取权限",
			Description: "读取用户信息的权限",
			Category:    "user",
			Type:        model.PermissionTypeRead,
			Status:      1,
		},
		{
			Tag:         "user_write",
			Name:        "用户写入权限",
			Description: "修改用户信息的权限",
			Category:    "user",
			Type:        model.PermissionTypeWrite,
			Status:      1,
		},
		{
			Tag:         "user_create",
			Name:        "用户创建权限",
			Description: "创建新用户的权限",
			Category:    "user",
			Type:        model.PermissionTypeCreate,
			Status:      1,
		},
		{
			Tag:         "user_delete",
			Name:        "用户删除权限",
			Description: "删除用户的权限",
			Category:    "user",
			Type:        model.PermissionTypeDelete,
			Status:      1,
		},

		// 内容相关权限
		{
			Tag:         "content_read",
			Name:        "内容读取权限",
			Description: "读取内容的权限",
			Category:    "content",
			Type:        model.PermissionTypeRead,
			Status:      1,
		},
		{
			Tag:         "content_write",
			Name:        "内容写入权限",
			Description: "修改内容的权限",
			Category:    "content",
			Type:        model.PermissionTypeWrite,
			Status:      1,
		},
		{
			Tag:         "content_create",
			Name:        "内容创建权限",
			Description: "创建新内容的权限",
			Category:    "content",
			Type:        model.PermissionTypeCreate,
			Status:      1,
		},
		{
			Tag:         "content_delete",
			Name:        "内容删除权限",
			Description: "删除内容的权限",
			Category:    "content",
			Type:        model.PermissionTypeDelete,
			Status:      1,
		},

		// 系统相关权限
		{
			Tag:         "system_read",
			Name:        "系统读取权限",
			Description: "读取系统信息的权限",
			Category:    "system",
			Type:        model.PermissionTypeRead,
			Status:      1,
		},
		{
			Tag:         "system_write",
			Name:        "系统写入权限",
			Description: "修改系统设置的权限",
			Category:    "system",
			Type:        model.PermissionTypeWrite,
			Status:      1,
		},

		// 管理相关权限
		{
			Tag:         "admin_read",
			Name:        "管理读取权限",
			Description: "读取管理面板的权限",
			Category:    "admin",
			Type:        model.PermissionTypeRead,
			Status:      1,
		},
		{
			Tag:         "admin_write",
			Name:        "管理写入权限",
			Description: "执行管理操作的权限",
			Category:    "admin",
			Type:        model.PermissionTypeWrite,
			Status:      1,
		},
		{
			Tag:         "admin_create",
			Name:        "管理创建权限",
			Description: "创建管理资源的权限",
			Category:    "admin",
			Type:        model.PermissionTypeCreate,
			Status:      1,
		},
		{
			Tag:         "admin_delete",
			Name:        "管理删除权限",
			Description: "删除管理资源的权限",
			Category:    "admin",
			Type:        model.PermissionTypeDelete,
			Status:      1,
		},
	}

	for _, tag := range defaultTags {
		exists, err := permissionTagDAO.ExistsByTag(ctx, tag.Tag)
		if err != nil {
			return err
		}

		if !exists {
			if err := permissionTagDAO.Create(ctx, tag); err != nil {
				return err
			}
			logger.Info("创建权限标签成功", zap.String("tag", tag.Tag))
		} else {
			logger.Debug("权限标签已存在", zap.String("tag", tag.Tag))
		}
	}

	return nil
}

// configureDefaultPermissions 配置默认权限
func configureDefaultPermissions(ctx context.Context, groupPermissionDAO *dao.UserGroupPermissionDAO) error {
	logger.Info("开始配置默认权限...")

	// 调用DAO中的默认权限配置方法
	if err := groupPermissionDAO.CreateDefaultPermissions(ctx); err != nil {
		return err
	}

	logger.Info("默认权限配置完成")
	return nil
}

// CreateSuperAdminUser 创建超级管理员用户
func CreateSuperAdminUser(ctx context.Context, userDAO *dao.UserDAO, username, email, password string) error {
	// 检查是否已存在超级管理员
	exists, err := userDAO.ExistsByRole(ctx, model.RoleSuperAdmin)
	if err != nil {
		return err
	}

	if exists {
		logger.Info("超级管理员已存在，跳过创建")
		return nil
	}

	// 创建超级管理员用户
	superAdmin := &model.User{
		Username:      username,
		Email:         email,
		PasswordHash:  hashPassword(password),
		Role:          model.RoleSuperAdmin,
		IsSuperAdmin:  true,
		PermissionLevel: model.PermissionLevelSuperAdmin,
		Status:        1,
		EmailVerified: true,
	}

	if err := userDAO.Create(ctx, superAdmin); err != nil {
		return fmt.Errorf("创建超级管理员失败: %w", err)
	}

	logger.Info("超级管理员创建成功", 
		zap.String("username", username),
		zap.String("email", email),
	)

	return nil
}

// hashPassword 密码哈希函数（简化示例）
func hashPassword(password string) string {
	// 实际项目中应该使用安全的密码哈希算法
	// 这里使用简化实现，实际使用时需要替换为bcrypt等安全算法
	return fmt.Sprintf("hashed_%s", password)
}

// GetPermissionSummary 获取权限系统摘要信息
func GetPermissionSummary(ctx context.Context, db interface{}) (map[string]interface{}, error) {
	gormDB, ok := db.(*gorm.DB)
	if !ok {
		return nil, fmt.Errorf("无效的数据库连接")
	}

	userGroupDAO := dao.NewUserGroupDAO(gormDB)
	permissionTagDAO := dao.NewPermissionTagDAO(gormDB)
	groupPermissionDAO := dao.NewUserGroupPermissionDAO(gormDB)
	userDAO := dao.NewUserDAO(gormDB)

	summary := make(map[string]interface{})

	// 统计用户组
	groups, err := userGroupDAO.GetMany(ctx, nil)
	if err == nil {
		summary["user_groups"] = len(groups)
	}

	// 统计权限标签
	tags, err := permissionTagDAO.GetMany(ctx, nil)
	if err == nil {
		summary["permission_tags"] = len(tags)
	}

	// 统计权限配置
	permissions, err := groupPermissionDAO.GetMany(ctx, nil)
	if err == nil {
		summary["group_permissions"] = len(permissions)
	}

	// 统计用户分布
	for _, role := range []string{
		model.RoleSuperAdmin,
		model.RoleAdmin,
		model.RoleVIP,
		model.RoleUser,
		model.RoleGuest,
	} {
		count, err := userDAO.CountByRole(ctx, role)
		if err == nil {
			summary["users_"+role] = count
		}
	}

	return summary, nil
}

// CleanupExpiredPermissions 清理过期权限
func CleanupExpiredPermissions(ctx context.Context, db interface{}) error {
	gormDB, ok := db.(*gorm.DB)
	if !ok {
		return fmt.Errorf("无效的数据库连接")
	}

	userPermissionDAO := dao.NewUserPermissionDAO(gormDB)

	// 清理过期权限
	if err := userPermissionDAO.CleanExpiredPermissions(ctx); err != nil {
		return fmt.Errorf("清理过期权限失败: %w", err)
	}

	logger.Info("过期权限清理完成")
	return nil
}

// SchedulePermissionCleanup 定时清理过期权限
func SchedulePermissionCleanup(ctx context.Context, db interface{}) {
	ticker := time.NewTicker(24 * time.Hour) // 每天清理一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := CleanupExpiredPermissions(ctx, db); err != nil {
				logger.Error("定时清理过期权限失败", zap.Error(err))
			} else {
				logger.Info("定时清理过期权限成功")
			}
		case <-ctx.Done():
			logger.Info("权限清理定时器已停止")
			return
		}
	}
}