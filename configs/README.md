# Charlotte API 配置系统说明

## 配置架构概述

Charlotte API 采用**分层配置架构**，将配置分为两个主要层次：

1. **全局环境变量** (`.env`文件) - 基础设施连接信息
2. **组件配置文件** (`configs/`目录) - 应用组件详细配置

这种架构的优势：
- **环境隔离**：不同环境使用不同的配置文件
- **安全分离**：敏感信息通过环境变量管理
- **配置复用**：组件配置在不同环境中可复用

## 配置层次说明

### 1. 全局环境变量 (`.env`文件)

**用途**：存储基础设施连接信息和全局配置
**位置**：项目根目录下的`.env`文件
**特点**：
- 包含服务器地址、端口等基础设施信息
- 通过环境变量注入，支持不同部署环境
- 敏感信息（密码、密钥）的安全管理

**示例配置**：
```bash
# 服务器全局配置
SERVER_NAME=charlotte-api
SERVER_PORT=8080
SERVER_MODE=debug

# 基础设施连接配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
REDIS_HOST=localhost
KAFKA_BROKER_1=localhost:9092
NACOS_HOST=localhost

# 安全配置
JWT_SECRET=your-jwt-secret
```

### 2. 组件配置文件 (`configs/`目录)

**用途**：存储应用组件的详细配置参数
**位置**：`configs/`目录下的YAML文件
**特点**：
- 按环境分类：开发、测试、生产环境
- 包含性能调优、功能开关等详细参数
- 支持配置热更新

**环境配置文件**：
- `config.yaml` - 默认配置（基础参数）
- `config.development.yaml` - 开发环境配置
- `config.test.yaml` - 测试环境配置  
- `config.production.yaml` - 生产环境配置

## 配置加载优先级

配置加载遵循以下优先级（从高到低）：

1. **命令行参数** (`--config`)
2. **环境变量** (以`CHARLOTTE_`前缀)
3. **配置文件** (按环境加载)
4. **默认值** (代码中设置的默认值)

## 环境配置说明

### 开发环境 (`config.development.yaml`)
**特点**：
- 启用调试工具（pprof、详细日志）
- 宽松的性能限制
- 自动数据库迁移
- 控制台日志输出

**适用场景**：本地开发、功能测试

### 测试环境 (`config.test.yaml`)
**特点**：
- 独立的测试数据库
- 启用数据库表重建
- 详细的错误信息
- 适合自动化测试

**适用场景**：单元测试、集成测试

### 生产环境 (`config.production.yaml`)
**特点**：
- 严格的安全配置
- 优化的性能参数
- 禁用调试功能
- 文件日志输出

**适用场景**：线上部署

## 配置项分类

### 基础设施配置
- `server` - 服务器基本配置
- `database` - 数据库连接和连接池配置
- `redis` - Redis连接和连接池配置
- `kafka` - Kafka消费者配置
- `nacos` - Nacos配置管理

### 安全配置
- `jwt` - JWT认证配置
- `security` - 安全策略（CORS、限流等）

### 性能配置
- `performance` - 请求处理性能参数
- `health` - 健康检查配置

### 监控配置
- `monitoring` - 指标监控和追踪配置
- `log` - 日志系统配置

### 开发工具配置
- `devtools` - 开发调试工具配置
- `migrate` - 数据库迁移配置

## 使用示例

### 1. 本地开发环境
```bash
# 复制环境变量模板
cp .env.example .env

# 修改.env文件中的配置
vim .env

# 启动应用（自动加载开发环境配置）
go run main.go
```

### 2. 生产环境部署
```bash
# 设置生产环境变量
export CHARLOTTE_SERVER_MODE=release
export CHARLOTTE_DB_HOST=postgres-primary
export CHARLOTTE_REDIS_HOST=redis-primary

# 启动应用（加载生产环境配置）
./charlotte-api
```

### 3. 自定义配置文件
```bash
# 使用自定义配置文件
go run main.go --config /path/to/custom-config.yaml
```

## 配置热更新

系统支持配置热更新，修改配置文件后无需重启服务：

1. 修改对应的YAML配置文件
2. 系统自动检测变化并重新加载配置
3. 日志中会显示配置更新信息

## 最佳实践

### 1. 环境变量管理
- 敏感信息（密码、密钥）必须通过环境变量设置
- 不同部署环境使用不同的环境变量文件
- 生产环境使用安全的密钥管理服务

### 2. 配置文件管理
- 开发环境启用所有调试功能
- 测试环境使用独立的数据库实例
- 生产环境禁用不必要的调试功能

### 3. 配置验证
- 应用启动时会验证配置完整性
- 关键配置缺失会阻止应用启动
- 建议在部署前进行配置验证

## 故障排除

### 常见问题

1. **配置加载失败**
   - 检查配置文件语法（YAML格式）
   - 确认文件路径和权限
   - 查看日志中的错误信息

2. **环境变量不生效**
   - 确认环境变量前缀为`CHARLOTTE_`
   - 检查变量名大小写和格式
   - 重启应用使环境变量生效

3. **配置热更新失败**
   - 确认文件监视功能已启用
   - 检查文件系统权限
   - 查看配置变更日志

## 扩展配置

如需添加新的配置项，请：

1. 在`internal/config/config.go`中添加结构体定义
2. 在`setDefaults`函数中设置默认值
3. 在对应的环境配置文件中添加配置项
4. 更新配置验证逻辑

## 相关文件

- `internal/config/config.go` - 配置加载和结构定义
- `.env.example` - 环境变量模板
- `configs/config.yaml` - 默认配置
- `configs/config.development.yaml` - 开发环境配置
- `configs/config.test.yaml` - 测试环境配置
- `configs/config.production.yaml` - 生产环境配置