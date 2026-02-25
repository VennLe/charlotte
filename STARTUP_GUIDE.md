# Charlotte API 启动指南

本指南介绍如何使用不同的启动方式来运行 Charlotte API 项目。

## 快速启动

### 方式 1: 批处理脚本 (Windows 推荐)

```bash
# 查看帮助
start.bat help

# 直接运行 (开发模式)
start.bat run

# 构建并启动
start.bat start

# 构建项目
start.bat build

# 清理构建产物
start.bat clean
```

### 方式 2: PowerShell 脚本 (Windows)

```powershell
# 查看帮助
.\start.ps1 help

# 直接运行 (开发模式)
.\start.ps1 run

# 构建并启动
.\start.ps1 start

# 构建项目
.\start.ps1 build
```

### 方式 3: Makefile (Linux/macOS/Windows with make)

```bash
# 查看所有可用命令
make help

# 直接运行 (开发模式)
make run

# 构建并启动
make start

# 仅构建
make build

# 开发模式 (热重载)
make dev
```

### 方式 4: Go 命令

```bash
# 直接运行
go run main.go start

# 构建并运行
go build -o bin/charlotte main.go
./bin/charlotte start
```

## 常用命令对照表

| 功能 | 批处理 | PowerShell | Makefile | Go 命令 |
|------|--------|------------|----------|---------|
| 帮助 | `start.bat help` | `.\start.ps1 help` | `make help` | - |
| 运行 | `start.bat run` | `.\start.ps1 run` | `make run` | `go run main.go start` |
| 构建 | `start.bat build` | `.\start.ps1 build` | `make build` | `go build -o bin/charlotte main.go` |
| 启动 | `start.bat start` | `.\start.ps1 start` | `make start` | `./bin/charlotte start` |
| 清理 | `start.bat clean` | `.\start.ps1 clean` | `make clean` | - |
| 迁移 | `start.bat migrate` | `.\start.ps1 migrate` | `make migrate` | `go run main.go migrate` |
| 配置 | `start.bat config` | `.\start.ps1 config` | `make config` | `go run main.go config show` |
| 测试 | `start.bat test` | `.\start.ps1 test` | `make test` | `go test ./...` |
| 依赖 | `start.bat deps` | `.\start.ps1 deps` | `make deps` | `go mod tidy` |
| 格式化 | `start.bat fmt` | `.\start.ps1 fmt` | `make fmt` | `go fmt ./...` |
| 检查 | `start.bat lint` | `.\start.ps1 lint` | `make lint` | `golangci-lint run` |

## 完整命令列表

### 批处理脚本 / PowerShell

```bash
# 运行相关
start.bat run              # 直接运行应用 (go run)
start.bat start            # 构建并启动应用
start.bat dev              # 开发模式 (需要 air)

# 构建相关
start.bat build            # 构建应用
start.bat clean            # 清理构建产物

# 数据库相关
start.bat migrate          # 执行数据库迁移

# 配置相关
start.bat config           # 显示当前配置
start.bat config-validate  # 验证配置
start.bat config-env       # 显示环境变量映射

# 信息相关
start.bat version          # 显示版本信息

# 开发相关
start.bat test             # 运行测试
start.bat deps             # 下载依赖
start.bat fmt              # 格式化代码
start.bat lint             # 运行代码检查

# Docker 相关 (PowerShell)
.\start.ps1 docker         # 构建并运行 Docker 容器
```

### Makefile (Linux/macOS)

```bash
# 开发相关
make dev                   # 开发模式 (热重载)
make run                   # 直接运行应用
make start                 # 构建并启动应用
make stop                  # 停止应用
make restart               # 重启应用

# 构建相关
make build                 # 构建应用
make build-all             # 交叉编译多平台
make build-linux           # 构建 Linux 二进制文件

# 数据库相关
make migrate               # 执行数据库迁移
make migrate-up            # 数据库迁移 (up)
make migrate-down          # 数据库迁移 (down)
make migrate-status        # 查看迁移状态

# 配置相关
make config                # 显示当前配置
make config-validate       # 验证配置
make config-env            # 显示环境变量映射

# 版本信息
make version               # 显示版本信息
make info                  # 显示项目信息

# 代码质量
make fmt                   # 格式化代码
make lint                  # 运行代码检查
make vet                   # 运行 go vet
make check                 # 运行所有代码检查

# 测试相关
make test                  # 运行测试
make test-short            # 快速测试 (跳过集成测试)
make test-cover            # 测试并显示覆盖率
make benchmark             # 运行性能测试

# 依赖管理
make deps                  # 下载依赖
make deps-verify           # 验证依赖
make deps-update           # 更新依赖

# 清理
make clean                 # 清理构建产物
make clean-all             # 清理所有产物 (包括缓存)

# Docker 相关
make docker-build          # 构建 Docker 镜像
make docker-run            # 运行 Docker 容器
make docker-stop           # 停止 Docker 容器
make docker-logs           # 查看 Docker 容器日志
make docker-restart        # 重启 Docker 容器
make docker-compose-up     # 使用 Docker Compose 启动
make docker-compose-down   # 使用 Docker Compose 停止

# 安装卸载
make install               # 安装应用到系统 (需要 sudo)
make uninstall             # 从系统卸载应用

# CI/CD
make ci                    # CI 流水线
make ci-fast               # 快速 CI 检查

# 安全检查
make security              # 运行安全检查

# 性能分析
make pprof-cpu             # CPU 性能分析
make pprof-mem             # 内存性能分析
```

## 启动前准备

### 1. 安装依赖

```bash
# 下载 Go 依赖
start.bat deps
# 或
go mod download
```

### 2. 配置文件

确保配置文件存在：

```bash
configs/config.yaml
```

如果不存在，可以从示例文件复制：

```bash
cp .env.example configs/config.yaml
# 然后编辑配置文件
```

### 3. 启动依赖服务

确保以下服务已启动：

- PostgreSQL (默认: localhost:5432)
- Redis (默认: localhost:6379)
- Kafka (可选, 默认: localhost:9092)

## 推荐开发流程

### Windows 用户 (批处理)

```bash
# 1. 安装开发依赖
start.bat deps

# 2. 格式化代码
start.bat fmt

# 3. 运行测试
start.bat test

# 4. 启动开发模式 (需要先安装 air: go install github.com/cosmtrek/air@latest)
start.bat dev

# 或直接运行
start.bat run
```

### Windows 用户 (PowerShell)

```powershell
# 1. 安装开发依赖
.\start.ps1 deps

# 2. 格式化代码
.\start.ps1 fmt

# 3. 运行测试
.\start.ps1 test

# 4. 启动开发模式
.\start.ps1 dev

# 或直接运行
.\start.ps1 run
```

### Linux/macOS 用户

```bash
# 1. 安装开发依赖
make deps

# 2. 格式化代码
make fmt

# 3. 运行测试
make test

# 4. 启动开发模式 (需要先安装 air: go install github.com/cosmtrek/air@latest)
make dev

# 或直接运行
make run
```

## 常见问题

### 1. 端口被占用

如果 8080 端口被占用，可以修改配置文件中的端口：

```yaml
# configs/config.yaml
server:
  port: "8081"  # 修改为其他端口
```

### 2. 数据库连接失败

检查 PostgreSQL 是否运行，并确认配置文件中的数据库连接信息是否正确。

### 3. Redis 连接失败

检查 Redis 是否运行：

```bash
# Windows
redis-cli ping

# Linux/macOS
redis-cli ping
```

### 4. PowerShell 执行策略

如果 PowerShell 脚本无法执行，需要设置执行策略：

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

## 生产部署

### 使用 Docker Compose

```bash
# 启动所有服务
docker-compose up -d --build

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

### 使用部署脚本

```powershell
# 部署到虚拟机
.\scripts\deploy.ps1 -Mode VM -Version "1.0.0"

# 部署到 Docker
.\scripts\deploy.ps1 -Mode Docker -Version "1.0.0"
```

## 获取帮助

- 使用 `start.bat help` (批处理)
- 使用 `.\start.ps1 help` (PowerShell)
- 使用 `make help` (Makefile)
- 或直接使用 `go run main.go --help`
