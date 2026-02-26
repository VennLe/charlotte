# Charlotte API 配置迁移和使用指南

## 概述

本指南介绍如何从旧的配置系统（.env + 多个YAML文件）迁移到新的统一安全配置系统。新系统提供配置加密、环境变量优先级和配置热更新等高级功能。

## 当前配置问题分析

### 1. 配置冗余问题
- **多个配置文件**: .env, config.yaml, config.dev.yaml, config.production.yaml
- **配置分散**: 配置信息分散在不同文件中，管理困难
- **环境变量冲突**: 环境变量和配置文件之间的优先级不明确

### 2. 安全性问题
- **敏感信息明文存储**: 数据库密码、JWT密钥等敏感信息未加密
- **缺乏保护措施**: 配置文件没有加密和访问控制
- **密钥管理缺失**: 缺乏密钥轮换和管理机制

## 新配置系统特性

### 1. 统一配置管理
- **单一配置文件**: 所有配置集中在 `configs/config.secure.yaml`
- **环境变量支持**: 支持环境变量覆盖配置值
- **多环境支持**: 通过环境变量切换不同环境配置

### 2. 安全增强
- **配置加密**: 敏感配置支持AES加密存储
- **密钥管理**: 支持外部密钥管理系统
- **访问控制**: 配置文件权限控制

### 3. 高级功能
- **热更新**: 配置修改无需重启服务
- **配置验证**: 启动时验证配置完整性
- **配置摘要**: 提供配置概览信息

## 迁移步骤

### 步骤1: 运行迁移工具

```bash
# 进入项目目录
cd D:/code/charlotte

# 运行配置迁移工具
go run cmd/migrate_config.go

# 或者使用指定参数
go run cmd/migrate_config.go --source . --target ./configs/config.secure.yaml --encrypt
```

### 步骤2: 检查迁移结果

迁移工具会生成以下文件：
- `configs/config.secure.yaml` - 新的安全配置文件
- `configs/config.template.yaml` - 配置模板文件

### 步骤3: 配置安全设置（可选但推荐）

#### 设置加密密钥
```bash
# 设置配置加密密钥（32字节）
export CHARLOTTE_ENCRYPTION_KEY=your-32-byte-encryption-key-here

# 或者使用密钥文件
export CHARLOTTE_ENCRYPTION_KEY=$(cat /path/to/encryption.key)
```

#### 设置敏感环境变量
```bash
# 数据库密码
export CHARLOTTE_DB_PASSWORD=your-secure-db-password

# JWT密钥
export CHARLOTTE_JWT_SECRET=your-32-byte-jwt-secret

# Redis密码
export CHARLOTTE_REDIS_PASSWORD=your-redis-password
```

### 步骤4: 更新应用代码

#### 旧代码（需要更新）
```go
// 旧方式：直接使用config包
config.Load("configs/config.yaml")
dbConfig := config.Global.Database
```

#### 新代码（推荐使用）
```go
// 新方式：使用安全配置管理器
scm := config.NewSecureConfigManager()
err := scm.LoadSecureConfig("configs/config.secure.yaml")
if err != nil {
    log.Fatal("配置加载失败:", err)
}

dbHost := scm.GetString("database.host")
dbPassword := scm.GetString("database.password")
```

#### 向后兼容方式
```go
// 保持向后兼容，自动使用新系统
config.LoadSecureConfig("")
// 或者
config.Load("configs/config.secure.yaml")
```

## 配置文件结构

### 新配置文件示例
```yaml
# configs/config.secure.yaml

server:
  name: ${CHARLOTTE_SERVER_NAME:-charlotte-api}
  port: ${CHARLOTTE_SERVER_PORT:-8080}
  mode: ${CHARLOTTE_SERVER_MODE:-debug}

database:
  host: ${CHARLOTTE_DB_HOST:-localhost}
  user: ${CHARLOTTE_DB_USER:-postgres}
  password: ${CHARLOTTE_DB_PASSWORD:-postgres}  # 支持加密

security:
  encryption_key: ${CHARLOTTE_ENCRYPTION_KEY:-}
```

### 环境变量优先级

新系统遵循以下优先级：
1. **环境变量**（最高优先级）
2. **配置文件中的值**
3. **默认值**（最低优先级）

## 安全最佳实践

### 1. 密钥管理
- **使用密钥管理系统**: HashiCorp Vault, AWS KMS, Azure Key Vault
- **定期轮换密钥**: 每3-6个月轮换一次加密密钥
- **分离密钥存储**: 密钥与配置分离存储

### 2. 配置保护
- **文件权限**: 配置文件设置为600权限
- **加密存储**: 敏感配置使用AES加密
- **访问审计**: 记录配置访问日志

### 3. 环境管理
- **开发环境**: 使用默认配置，不加密
- **测试环境**: 使用测试密钥，轻度加密
- **生产环境**: 使用强密钥，完全加密

## 故障排除

### 常见问题

#### Q1: 迁移后服务无法启动
**原因**: 配置路径或格式错误
**解决**: 检查配置文件路径和格式，使用验证功能

```go
// 验证配置完整性
if err := config.Validate(); err != nil {
    log.Fatal("配置验证失败:", err)
}
```

#### Q2: 加密配置无法解密
**原因**: 加密密钥不匹配或丢失
**解决**: 检查环境变量设置，重新设置密钥

```bash
# 检查密钥设置
echo $CHARLOTTE_ENCRYPTION_KEY

# 重新设置密钥
export CHARLOTTE_ENCRYPTION_KEY=correct-key-here
```

#### Q3: 环境变量不生效
**原因**: 环境变量名称或格式错误
**解决**: 检查环境变量前缀和命名规则

```bash
# 正确的环境变量命名
CHARLOTTE_DB_PASSWORD=password
CHARLOTTE_JWT_SECRET=secret

# 错误的环境变量命名（不会生效）
DB_PASSWORD=password
JWT_SECRET=secret
```

## 性能优化建议

### 1. 配置缓存
- 启用配置缓存减少文件IO
- 设置合理的缓存过期时间

### 2. 热更新优化
- 对频繁变化的配置启用热更新
- 对稳定配置禁用热更新减少开销

### 3. 内存管理
- 监控配置内存使用
- 定期清理过期配置缓存

## 监控和日志

### 配置访问日志
```go
// 记录配置访问日志
logger.Info("配置访问", 
    zap.String("key", key),
    zap.String("source", "secure_config"),
    zap.Bool("encrypted", isSensitive))
```

### 配置变更审计
```go
// 记录配置变更
scm.viper.OnConfigChange(func(e fsnotify.Event) {
    logger.Info("配置变更审计",
        zap.String("file", e.Name),
        zap.Time("timestamp", time.Now()),
        zap.String("user", getCurrentUser()))
})
```

## 总结

新的安全配置系统解决了旧系统的冗余和安全性问题，提供了：

- ✅ **统一配置管理** - 简化配置结构
- ✅ **增强安全性** - 支持配置加密
- ✅ **环境友好** - 支持多环境和变量覆盖
- ✅ **运维友好** - 热更新和验证功能
- ✅ **向后兼容** - 平滑迁移路径

通过遵循本指南，您可以安全、高效地迁移到新配置系统，并享受其带来的安全和便利特性。