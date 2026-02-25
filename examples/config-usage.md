# Charlotte API 配置使用示例

## 概述

Charlotte API 使用 YAML 配置文件管理所有组件的配置信息，支持环境变量覆盖和热更新功能。

## 基本用法

### 1. 使用默认配置
```bash
# 自动加载 configs/config.yaml
./charlotte-server
```

### 2. 指定配置文件
```bash
# 开发环境
./charlotte-server --config configs/config.development.yaml

# 生产环境
./charlotte-server --config configs/config.production.yaml

# 自定义配置文件
./charlotte-server --config /path/to/your/config.yaml
```

### 3. 使用环境变量
```bash
# 设置环境变量
export CHARLOTTE_DATABASE_HOST=postgres-primary
export CHARLOTTE_DATABASE_PASSWORD=your_password
export CHARLOTTE_JWT_SECRET=your_secret_key

# 启动服务
./charlotte-server
```

## 配置验证

### 使用配置工具验证
```bash
# 验证配置完整性
./config-tool validate --config configs/config.yaml

# 显示当前配置
./config-tool show --config configs/config.yaml

# 查看环境变量映射
./config-tool env
```

### 在代码中验证
```go
package main

import (
	"github.com/VennLe/charlotte/internal/config"
	"github.com/VennLe/charlotte/pkg/logger"
)

func main() {
	// 加载配置
	config.LoadConfig("configs/config.yaml")
	
	// 验证配置
	if err := config.Validate(); err != nil {
		logger.Fatal("配置验证失败", zap.Error(err))
	}
	
	logger.Info(config.GetConfigSummary())
}
```

## 环境变量映射

### 配置项与环境变量对应关系

| 配置项 | 环境变量 | 示例 |
|--------|----------|------|
| `server.name` | `CHARLOTTE_SERVER_NAME` | `charlotte-api` |
| `server.port` | `CHARLOTTE_SERVER_PORT` | `8080` |
| `database.host` | `CHARLOTTE_DATABASE_HOST` | `localhost` |
| `database.user` | `CHARLOTTE_DATABASE_USER` | `postgres` |
| `database.password` | `CHARLOTTE_DATABASE_PASSWORD` | `password123` |
| `redis.host` | `CHARLOTTE_REDIS_HOST` | `redis-primary` |
| `jwt.secret` | `CHARLOTTE_JWT_SECRET` | `your-secret-key` |

### 环境变量优先级

环境变量优先级最高，会覆盖配置文件中的设置：

```bash
# 配置文件中的数据库端口是5432
export CHARLOTTE_DATABASE_PORT=5433
# 实际使用的端口是5433
```

## 热更新功能

### 配置文件热更新
当配置文件发生变化时，系统会自动重新加载配置：

```yaml
# 修改前
server:
  port: "8080"

# 修改后
server:
  port: "9090"
```

日志输出：
```
INFO   配置已热更新     {"file": "configs/config.yaml"}
```

### 安全注意事项
- 敏感配置（如密码、密钥）建议使用环境变量
- 生产环境建议禁用热更新功能
- 配置变更后建议验证服务功能

## 多环境配置示例

### 开发环境配置
```bash
# configs/config.development.yaml
./charlotte-server --config configs/config.development.yaml
```

### 生产环境配置
```bash
# 使用环境变量覆盖敏感信息
export CHARLOTTE_DATABASE_PASSWORD=$(aws secretsmanager get-secret-value --secret-id db-password --query SecretString --output text)
export CHARLOTTE_JWT_SECRET=$(aws secretsmanager get-secret-value --secret-id jwt-secret --query SecretString --output text)

./charlotte-server --config configs/config.production.yaml
```

### Docker 环境配置
```dockerfile
# Dockerfile
ENV CHARLOTTE_SERVER_PORT=8080
ENV CHARLOTTE_DATABASE_HOST=postgres
ENV CHARLOTTE_REDIS_HOST=redis

CMD ["./charlotte-server"]
```

## 最佳实践

### 1. 开发环境
- 使用 `config.development.yaml`
- 启用详细日志和调试模式
- 启用数据库迁移和索引创建

### 2. 测试环境
- 使用独立的数据库和Redis实例
- 配置测试专用的Kafka主题
- 启用配置验证

### 3. 生产环境
- 使用环境变量存储敏感信息
- 禁用详细日志和调试模式
- 谨慎启用数据库迁移
- 定期轮换密钥和密码

### 4. 安全建议
- 不要在配置文件中存储密码
- 使用密钥管理服务（如AWS Secrets Manager）
- 定期审计配置权限
- 启用配置变更日志记录

## 故障排除

### 常见问题

1. **配置加载失败**
   - 检查配置文件路径和权限
   - 验证YAML语法是否正确
   - 查看日志中的详细错误信息

2. **环境变量不生效**
   - 确认环境变量名称是否正确
   - 检查环境变量是否已导出
   - 重启服务使环境变量生效

3. **热更新失败**
   - 检查文件监视器权限
   - 确认配置文件未被其他进程占用
   - 查看系统日志中的错误信息

### 调试技巧

```bash
# 启用详细日志
export CHARLOTTE_LOG_LEVEL=debug

# 查看配置加载过程
./charlotte-server --config configs/config.yaml

# 使用配置工具调试
./config-tool validate --config configs/config.yaml -v
```