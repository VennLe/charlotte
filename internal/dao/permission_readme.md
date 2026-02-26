# 精简权限系统使用说明

## 概述

本系统将原有的复杂权限标签系统简化为基于角色的权限管理，将所有权限相关的DAO操作整合到单一文件中，大大减少了代码冗余和维护复杂度。

## 主要改进

### 1. 简化权限模型
- 移除了复杂的权限标签系统
- 使用简单的角色-权限映射
- 减少了数据库表数量

### 2. 统一DAO接口
- 将5个权限相关的DAO文件整合为1个
- 提供统一的权限检查接口
- 简化了权限管理逻辑

### 3. 角色定义

系统预定义了5种角色：

| 角色 | 权限级别 | 描述 |
|------|----------|------|
| `guest` | 低 | 游客，只能查看公开内容 |
| `user` | 低 | 普通用户，只能查看自己的内容 |
| `vip` | 中 | VIP用户，可以查看所有内容 |
| `admin` | 高 | 普通管理员，可以查看和修改但不能删除 |
| `superadmin` | 最高 | 超级管理员，拥有所有权限 |

## 快速开始

### 1. 初始化权限系统

```go
import "github.com/VennLe/charlotte/internal/dao"

// 创建统一权限DAO
permissionDAO := dao.NewUnifiedPermissionDAO(db)

// 初始化默认权限配置
if err := permissionDAO.InitializeDefaultPermissions(ctx); err != nil {
    log.Fatal("初始化权限配置失败:", err)
}
```

### 2. 设置用户角色

```go
// 设置用户为普通用户
if err := permissionDAO.SetUserRole(ctx, userID, "user"); err != nil {
    return err
}

// 设置用户为管理员
if err := permissionDAO.SetUserRole(ctx, userID, "admin"); err != nil {
    return err
}
```

### 3. 检查权限

```go
// 检查用户是否有查看内容的权限
hasPermission, err := permissionDAO.CheckPermission(ctx, userID, "content", "read")
if err != nil {
    return err
}

if hasPermission {
    // 允许访问
} else {
    // 拒绝访问
}
```

### 4. 使用权限服务

```go
import "github.com/VennLe/charlotte/internal/service"

// 创建权限服务
permissionService := service.NewSimplifiedPermissionService(userDAO, permissionDAO)

// 检查权限
req := &service.PermissionCheckRequest{
    UserID:       userID,
    ResourceType: "content",
    Operation:    "read",
}

result, err := permissionService.CheckPermission(ctx, req)
if err != nil {
    return err
}

if result.HasPermission {
    // 权限验证通过
}
```

### 5. 使用权限中间件

```go
import "github.com/VennLe/charlotte/internal/middleware"

// 创建权限中间件
permissionMiddleware := middleware.NewSimplifiedPermissionMiddleware(permissionService)

// 在路由中使用权限检查
router.GET("/api/content", 
    permissionMiddleware.CheckPermission("content", "read"),
    contentHandler,
)

// 要求管理员权限
router.POST("/api/admin/users", 
    permissionMiddleware.RequireAdmin(),
    createUserHandler,
)
```

## API参考

### UnifiedPermissionDAO 主要方法

#### 权限管理
- `InitializeDefaultPermissions(ctx) error` - 初始化默认权限配置
- `SetUserRole(ctx, userID, role) error` - 设置用户角色
- `CheckPermission(ctx, userID, resourceType, operation) (bool, error)` - 检查权限

#### 角色权限管理
- `AddRolePermission(ctx, role, resourceType, operations, scope) error` - 添加角色权限
- `RemoveRolePermission(ctx, role, resourceType) error` - 移除角色权限
- `GetRolePermissions(ctx, role) ([]RolePermission, error)` - 获取角色权限

#### 用户权限信息
- `GetUserRole(ctx, userID) (string, error)` - 获取用户角色
- `GetUserPermissions(ctx, userID) (map[string]interface{}, error)` - 获取用户权限信息

### SimplifiedPermissionService 主要方法

- `CheckPermission(ctx, req) (*PermissionCheckResult, error)` - 检查权限
- `SetUserRole(ctx, userID, role) error` - 设置用户角色
- `GetUserPermissions(ctx, userID) (map[string]interface{}, error)` - 获取用户权限信息
- `GetAvailableRoles() []map[string]interface{}` - 获取可用角色列表

### SimplifiedPermissionMiddleware 主要方法

- `CheckPermission(resourceType, operation) gin.HandlerFunc` - 权限检查中间件
- `RequireLogin() gin.HandlerFunc` - 要求登录中间件
- `RequireRole(role) gin.HandlerFunc` - 要求特定角色权限
- `RequireAdmin() gin.HandlerFunc` - 要求管理员权限
- `GetUserPermissions() gin.HandlerFunc` - 获取用户权限信息

## 默认权限配置

系统预定义了以下默认权限：

### 超级管理员 (`superadmin`)
- 所有资源类型：所有操作权限

### 普通管理员 (`admin`)
- 用户管理：查看、修改、创建
- 内容管理：查看、修改、创建
- 系统管理：查看、修改
- **不能删除**任何资源

### VIP用户 (`vip`)
- 用户管理：查看自己的信息
- 内容管理：查看所有内容

### 普通用户 (`user`)
- 用户管理：查看自己的信息
- 内容管理：查看自己的内容

### 游客 (`guest`)
- 内容管理：查看公开内容

## 扩展自定义权限

### 添加新的资源类型权限

```go
// 为管理员添加新的资源类型权限
if err := permissionDAO.AddRolePermission(ctx, "admin", "report", "read,write", "all"); err != nil {
    return err
}

// 检查权限
hasPermission, err := permissionDAO.CheckPermission(ctx, userID, "report", "read")
```

### 创建自定义角色

```go
// 添加新的角色权限配置
if err := permissionDAO.AddRolePermission(ctx, "moderator", "content", "read,write,delete", "all"); err != nil {
    return err
}

// 设置用户为新角色
if err := permissionDAO.SetUserRole(ctx, userID, "moderator"); err != nil {
    return err
}
```

## 性能优化

### 缓存策略
- 用户角色信息可以缓存，减少数据库查询
- 角色权限配置在系统启动时加载到内存
- 频繁的权限检查可以使用本地缓存

### 数据库优化
- 为常用查询字段建立索引
- 定期清理过期的临时权限
- 使用连接池减少数据库连接开销

## 迁移指南

### 从旧系统迁移

1. **备份数据**：备份原有的权限相关数据
2. **运行迁移脚本**：将旧数据转换为新格式
3. **测试验证**：确保权限检查功能正常
4. **逐步切换**：先在新系统测试，再逐步切换

### 数据转换示例

```sql
-- 将旧权限标签转换为新角色权限
INSERT INTO role_permissions (role, resource_type, operations, scope)
SELECT 
    CASE 
        WHEN ug.name = '超级管理员组' THEN 'superadmin'
        WHEN ug.name = '管理员组' THEN 'admin'
        WHEN ug.name = 'VIP用户组' THEN 'vip'
        WHEN ug.name = '普通用户组' THEN 'user'
        ELSE 'guest'
    END as role,
    pt.category as resource_type,
    ugp.operations,
    ugp.resource_scope as scope
FROM user_group_permissions ugp
JOIN user_groups ug ON ugp.user_group_id = ug.id
JOIN permission_tags pt ON ugp.permission_tag_id = pt.id;
```

## 故障排除

### 常见问题

1. **权限检查返回错误**
   - 检查数据库连接是否正常
   - 确认权限配置已初始化
   - 验证用户角色设置是否正确

2. **角色权限不生效**
   - 检查角色名称拼写是否正确
   - 确认权限配置包含该角色
   - 查看数据库中的权限记录

3. **性能问题**
   - 检查数据库索引
   - 考虑添加缓存层
   - 优化频繁的权限检查调用

### 调试方法

```go
// 启用调试日志
logger.Debug("权限检查详情", 
    zap.Uint("user_id", userID),
    zap.String("resource_type", resourceType),
    zap.String("operation", operation),
)

// 获取详细的权限信息
permissions, err := permissionDAO.GetUserPermissions(ctx, userID)
if err != nil {
    logger.Error("获取权限信息失败", zap.Error(err))
}
```

## 总结

新的精简权限系统相比原有系统具有以下优势：

- ✅ **代码简化**：从多个DAO文件整合为1个
- ✅ **维护容易**：权限逻辑更清晰
- ✅ **性能更好**：减少数据库查询
- ✅ **扩展性强**：易于添加新角色和权限
- ✅ **使用简单**：API接口简洁明了

推荐在新项目中使用此精简权限系统，或在现有项目中逐步迁移到此系统。