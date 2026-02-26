package dao

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/VennLe/charlotte/internal/model"
)

// PermissionExample 权限使用示例
// 展示如何使用精简版的权限系统

type PermissionExample struct {
	db *gorm.DB
}

func NewPermissionExample(db *gorm.DB) *PermissionExample {
	return &PermissionExample{db: db}
}

// RunPermissionExamples 运行权限使用示例
func (e *PermissionExample) RunPermissionExamples(ctx context.Context) error {
	// 创建统一权限DAO实例
	permissionDAO := NewUnifiedPermissionDAO(e.db)

	// 1. 初始化默认权限配置
	fmt.Println("=== 1. 初始化默认权限配置 ===")
	if err := permissionDAO.InitializeDefaultPermissions(ctx); err != nil {
		log.Printf("初始化权限配置失败: %v", err)
	} else {
		fmt.Println("✓ 默认权限配置初始化成功")
	}

	// 2. 设置用户角色示例
	fmt.Println("\n=== 2. 设置用户角色示例 ===")
	userID := uint(1)
	
	// 设置用户为普通用户
	if err := permissionDAO.SetUserRole(ctx, userID, model.RoleUser); err != nil {
		log.Printf("设置用户角色失败: %v", err)
	} else {
		fmt.Printf("✓ 用户 %d 角色设置为: %s\n", userID, model.RoleUser)
	}

	// 3. 检查权限示例
	fmt.Println("\n=== 3. 权限检查示例 ===")
	
	// 检查用户是否能查看内容
	hasPermission, err := permissionDAO.CheckPermission(ctx, userID, "content", model.PermissionRead)
	if err != nil {
		log.Printf("权限检查失败: %v", err)
	} else {
		fmt.Printf("用户 %d 查看内容权限: %v\n", userID, hasPermission)
	}

	// 检查用户是否能修改内容
	hasPermission, err = permissionDAO.CheckPermission(ctx, userID, "content", model.PermissionWrite)
	if err != nil {
		log.Printf("权限检查失败: %v", err)
	} else {
		fmt.Printf("用户 %d 修改内容权限: %v\n", userID, hasPermission)
	}

	// 4. 获取用户权限信息
	fmt.Println("\n=== 4. 获取用户权限信息 ===")
	permissions, err := permissionDAO.GetUserPermissions(ctx, userID)
	if err != nil {
		log.Printf("获取用户权限失败: %v", err)
	} else {
		fmt.Printf("用户 %d 权限信息: %+v\n", userID, permissions)
	}

	// 5. 临时角色授予示例
	fmt.Println("\n=== 5. 临时角色授予示例 ===")
	vipUserID := uint(2)
	if err := permissionDAO.GrantTemporaryRole(ctx, vipUserID, model.RoleVIP, 24*time.Hour); err != nil {
		log.Printf("授予临时角色失败: %v", err)
	} else {
		fmt.Printf("✓ 用户 %d 被授予临时VIP角色（24小时）\n", vipUserID)
	}

	// 6. 角色权限管理示例
	fmt.Println("\n=== 6. 角色权限管理示例 ===")
	
	// 为VIP角色添加特殊权限
	if err := permissionDAO.AddRolePermission(ctx, model.RoleVIP, "special", "read,write", "all"); err != nil {
		log.Printf("添加角色权限失败: %v", err)
	} else {
		fmt.Printf("✓ VIP角色添加特殊权限成功\n")
	}

	// 7. 不同角色权限对比示例
	fmt.Println("\n=== 7. 不同角色权限对比 ===")
	e.compareRolePermissions(ctx, permissionDAO)

	return nil
}

// compareRolePermissions 比较不同角色的权限
func (e *PermissionExample) compareRolePermissions(ctx context.Context, permissionDAO *UnifiedPermissionDAO) {
	roles := []string{
		model.RoleGuest,
		model.RoleUser,
		model.RoleVIP,
		model.RoleAdmin,
		model.RoleSuperAdmin,
	}

	operations := []string{
		model.PermissionRead,
		model.PermissionWrite,
		model.PermissionDelete,
	}

	resourceTypes := []string{
		"content",
		"user",
		"system",
	}

	fmt.Println("角色权限矩阵:")
	fmt.Printf("%-15s", "角色/操作")
	for _, op := range operations {
		fmt.Printf("%-10s", op)
	}
	fmt.Println()

	for _, resourceType := range resourceTypes {
		fmt.Printf("\n资源类型: %s\n", resourceType)
		for _, role := range roles {
			fmt.Printf("%-15s", role)
			
			for _, op := range operations {
				testUserID := uint(999) // 测试用户ID
				
				// 设置测试用户角色
				if err := permissionDAO.SetUserRole(ctx, testUserID, role); err != nil {
					fmt.Printf("%-10s", "ERROR")
					continue
				}
				
				hasPermission, err := permissionDAO.CheckPermission(ctx, testUserID, resourceType, op)
				if err != nil {
					fmt.Printf("%-10s", "ERROR")
				} else {
					if hasPermission {
						fmt.Printf("%-10s", "✓")
					} else {
						fmt.Printf("%-10s", "✗")
					}
				}
			}
			fmt.Println()
		}
	}
}

// DemoUserManagement 用户管理权限演示
func (e *PermissionExample) DemoUserManagement(ctx context.Context) error {
	permissionDAO := NewUnifiedPermissionDAO(e.db)

	fmt.Println("\n=== 用户管理权限演示 ===")

	// 创建测试用户
	testUsers := []struct {
		ID   uint
		Role string
	}{
		{1, model.RoleUser},
		{2, model.RoleVIP},
		{3, model.RoleAdmin},
		{4, model.RoleSuperAdmin},
	}

	for _, user := range testUsers {
		if err := permissionDAO.SetUserRole(ctx, user.ID, user.Role); err != nil {
			log.Printf("设置用户 %d 角色失败: %v", user.ID, err)
			continue
		}

		fmt.Printf("\n用户 %d (%s) 权限测试:\n", user.ID, user.Role)

		// 测试不同操作权限
		operations := map[string]string{
			"查看用户信息": model.PermissionRead,
			"修改用户信息": model.PermissionWrite,
			"删除用户":    model.PermissionDelete,
		}

		for desc, op := range operations {
			hasPermission, err := permissionDAO.CheckPermission(ctx, user.ID, "user", op)
			if err != nil {
				fmt.Printf("  %s: ERROR\n", desc)
			} else {
				if hasPermission {
					fmt.Printf("  %s: ✓ 允许\n", desc)
				} else {
					fmt.Printf("  %s: ✗ 拒绝\n", desc)
				}
			}
		}
	}

	return nil
}

// DemoContentManagement 内容管理权限演示
func (e *PermissionExample) DemoContentManagement(ctx context.Context) error {
	permissionDAO := NewUnifiedPermissionDAO(e.db)

	fmt.Println("\n=== 内容管理权限演示 ===")

	testUsers := []struct {
		ID   uint
		Role string
	}{
		{1, model.RoleGuest},
		{2, model.RoleUser},
		{3, model.RoleVIP},
		{4, model.RoleAdmin},
	}

	for _, user := range testUsers {
		if err := permissionDAO.SetUserRole(ctx, user.ID, user.Role); err != nil {
			log.Printf("设置用户 %d 角色失败: %v", user.ID, err)
			continue
		}

		fmt.Printf("\n用户 %d (%s) 内容权限测试:\n", user.ID, user.Role)

		operations := map[string]string{
			"查看内容": model.PermissionRead,
			"创建内容": model.PermissionWrite,
			"修改内容": model.PermissionWrite,
			"删除内容": model.PermissionDelete,
		}

		for desc, op := range operations {
			hasPermission, err := permissionDAO.CheckPermission(ctx, user.ID, "content", op)
			if err != nil {
				fmt.Printf("  %s: ERROR\n", desc)
			} else {
				if hasPermission {
					fmt.Printf("  %s: ✓ 允许\n", desc)
				} else {
					fmt.Printf("  %s: ✗ 拒绝\n", desc)
				}
			}
		}
	}

	return nil
}