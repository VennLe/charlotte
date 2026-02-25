# Charlotte API Makefile
# 项目构建、测试和部署管理

.PHONY: help build run dev start stop restart migrate clean test lint fmt deps docker docker-build docker-run docker-stop docker-compose-up docker-compose-down install uninstall

# 默认目标
.DEFAULT_GOAL := help

# ============================================
# 变量定义
# ============================================

APP_NAME := charlotte
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date +%Y-%m-%d_%H:%M:%S)

# Go 相关
GO := go
GOFLAGS := -v
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)

# Docker 相关
DOCKER_IMAGE := $(APP_NAME)
DOCKER_TAG := $(VERSION)
DOCKER_REGISTRY ?=

# 路径
CMD_DIR := cmd
CONFIG_DIR := configs
BIN_DIR := bin
LOGS_DIR := logs

# ============================================
# 帮助信息
# ============================================

help: ## 显示帮助信息
	@echo "$(APP_NAME) API - 常用命令"
	@echo ""
	@echo "使用方法:"
	@echo "  make [目标]"
	@echo ""
	@echo "可用目标:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ============================================
# 开发相关
# ============================================

dev: ## 开发模式运行 (热重载)
	@echo "🚀 启动开发模式..."
	@which air > /dev/null || (echo "❌ 未安装 air，请运行: go install github.com/cosmtrek/air@latest" && exit 1)
	@air

run: ## 直接运行应用 (编译后执行)
	@echo "🚀 启动应用..."
	@$(GO) run main.go start

start: build ## 构建并启动应用
	@echo "🚀 启动 $(APP_NAME)..."
	@mkdir -p $(LOGS_DIR)
	@$(BIN_DIR)/$(APP_NAME) start

stop: ## 停止应用
	@echo "⏹️  停止 $(APP_NAME)..."
	@pkill -f $(BIN_DIR)/$(APP_NAME) || echo "应用未运行"

restart: stop start ## 重启应用

# ============================================
# 构建相关
# ============================================

build: ## 构建应用
	@echo "🔨 构建 $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	@$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) main.go
	@echo "✅ 构建完成: $(BIN_DIR)/$(APP_NAME)"

build-all: ## 交叉编译多平台
	@echo "🔨 交叉编译 $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	@GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-linux-amd64 main.go
	@GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-darwin-amd64 main.go
	@GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-darwin-arm64 main.go
	@GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-windows-amd64.exe main.go
	@echo "✅ 多平台构建完成"

build-linux: ## 构建 Linux 二进制文件
	@echo "🔨 构建 Linux 版本..."
	@mkdir -p $(BIN_DIR)
	@GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-linux-amd64 main.go
	@echo "✅ 构建完成: $(BIN_DIR)/$(APP_NAME)-linux-amd64"

# ============================================
# 数据库迁移
# ============================================

migrate: ## 执行数据库迁移
	@echo "📊 执行数据库迁移..."
	@$(GO) run main.go migrate

migrate-up: ## 数据库迁移 (up)
	@$(GO) run main.go migrate up

migrate-down: ## 数据库迁移 (down)
	@$(GO) run main.go migrate down

migrate-status: ## 查看迁移状态
	@$(GO) run main.go migrate status

# ============================================
# 配置管理
# ============================================

config: ## 显示当前配置
	@$(GO) run main.go config show

config-validate: ## 验证配置
	@$(GO) run main.go config validate

config-env: ## 显示环境变量映射
	@$(GO) run main.go config env

# ============================================
# 版本信息
# ============================================

version: ## 显示版本信息
	@$(GO) run main.go version

info: ## 显示项目信息
	@echo "$(APP_NAME) 项目信息"
	@echo "===================="
	@echo "版本: $(VERSION)"
	@echo "构建时间: $(BUILD_TIME)"
	@echo "Go 版本: $$($(GO) version)"
	@echo "Git 分支: $$((git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown"))"
	@echo "Git 提交: $$((git rev-parse --short HEAD 2>/dev/null || echo "unknown"))"

# ============================================
# 代码质量
# ============================================

fmt: ## 格式化代码
	@echo "📝 格式化代码..."
	@$(GO) fmt ./...
	@echo "✅ 格式化完成"

lint: ## 运行代码检查
	@echo "🔍 运行代码检查..."
	@which golangci-lint > /dev/null || (echo "❌ 未安装 golangci-lint，请运行: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	@golangci-lint run ./...

vet: ## 运行 go vet
	@echo "🔍 运行 go vet..."
	@$(GO) vet ./...

check: fmt vet lint ## 运行所有代码检查

# ============================================
# 测试相关
# ============================================

test: ## 运行测试
	@echo "🧪 运行测试..."
	@$(GO) test -v -race -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✅ 测试完成，覆盖率报告: coverage.html"

test-short: ## 快速测试 (跳过集成测试)
	@$(GO) test -short -v ./...

test-cover: ## 测试并显示覆盖率
	@$(GO) test -cover ./...

benchmark: ## 运行性能测试
	@$(GO) test -bench=. -benchmem ./...

# ============================================
# 依赖管理
# ============================================

deps: ## 下载依赖
	@echo "📦 下载依赖..."
	@$(GO) mod download
	@$(GO) mod tidy
	@echo "✅ 依赖更新完成"

deps-verify: ## 验证依赖
	@$(GO) mod verify

deps-update: ## 更新依赖
	@echo "📦 更新依赖..."
	@$(GO) get -u ./...
	@$(GO) mod tidy
	@echo "✅ 依赖更新完成"

# ============================================
# 清理
# ============================================

clean: ## 清理构建产物
	@echo "🧹 清理构建产物..."
	@rm -rf $(BIN_DIR)
	@rm -rf coverage.out coverage.html
	@echo "✅ 清理完成"

clean-all: clean ## 清理所有产物 (包括缓存)
	@echo "🧹 清理所有产物..."
	@$(GO) clean -cache -modcache -testcache
	@rm -rf tmp
	@echo "✅ 清理完成"

# ============================================
# Docker 相关
# ============================================

docker-build: ## 构建 Docker 镜像
	@echo "🐳 构建 Docker 镜像..."
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -t $(DOCKER_IMAGE):latest .
	@echo "✅ Docker 镜像构建完成"

docker-run: ## 运行 Docker 容器
	@echo "🐳 运行 Docker 容器..."
	@docker run -d --name $(APP_NAME) -p 8080:8080 --restart unless-stopped $(DOCKER_IMAGE):latest

docker-stop: ## 停止 Docker 容器
	@echo "🐳 停止 Docker 容器..."
	@docker stop $(APP_NAME) || true
	@docker rm $(APP_NAME) || true

docker-logs: ## 查看 Docker 容器日志
	@docker logs -f $(APP_NAME)

docker-restart: docker-stop docker-run ## 重启 Docker 容器

docker-compose-up: ## 使用 Docker Compose 启动
	@docker-compose up -d --build

docker-compose-down: ## 使用 Docker Compose 停止
	@docker-compose down

docker-compose-logs: ## 查看 Docker Compose 日志
	@docker-compose logs -f

# ============================================
# 安装和卸载
# ============================================

install: ## 安装应用到系统 (需要 sudo)
	@echo "📦 安装 $(APP_NAME) 到系统..."
	@$(GO) install $(GOFLAGS) -ldflags "$(LDFLAGS)" ./main.go
	@echo "✅ 安装完成: $$($(GO) env GOPATH)/bin/$(APP_NAME)"

uninstall: ## 从系统卸载应用
	@echo "🗑️  卸载 $(APP_NAME)..."
	@rm -f $$($(GO) env GOPATH)/bin/$(APP_NAME)
	@echo "✅ 卸载完成"

# ============================================
# 生成相关
# ============================================

gen-mock: ## 生成 mock 文件
	@echo "🔧 生成 mock 文件..."
	@which mockgen > /dev/null || (echo "❌ 未安装 mockgen，请运行: go install github.com/golang/mock/mockgen@latest" && exit 1)
	@find . -name "*.go" -type f | grep -v "_test.go" | xargs -I {} sh -c 'mockgen -source {} -destination mock_{}.go' 2>/dev/null || true
	@echo "✅ Mock 文件生成完成"

gen-proto: ## 生成 protobuf 文件
	@echo "🔧 生成 protobuf 文件..."
	@which protoc > /dev/null || (echo "❌ 未安装 protoc" && exit 1)
	@cd proto && protoc --go_out=../pkg --go-grpc_out=../pkg *.proto
	@echo "✅ Protobuf 文件生成完成"

# ============================================
# 系统服务 (Linux)
# ============================================

install-service: ## 安装 systemd 服务 (需要 sudo)
	@echo "📦 安装 systemd 服务..."
	@sudo cp deployments/$(APP_NAME).service /etc/systemd/system/
	@sudo systemctl daemon-reload
	@sudo systemctl enable $(APP_NAME)
	@echo "✅ 服务安装完成"
	@echo "启动服务: sudo systemctl start $(APP_NAME)"

uninstall-service: ## 卸载 systemd 服务 (需要 sudo)
	@echo "🗑️  卸载 systemd 服务..."
	@sudo systemctl stop $(APP_NAME) || true
	@sudo systemctl disable $(APP_NAME) || true
	@sudo rm -f /etc/systemd/system/$(APP_NAME).service
	@sudo systemctl daemon-reload
	@echo "✅ 服务卸载完成"

# ============================================
# 监控和日志
# ============================================

logs: ## 查看应用日志
	@tail -f $(LOGS_DIR)/app.log || echo "日志文件不存在"

log-info: ## 查看 Info 级别日志
	@grep "INFO" $(LOGS_DIR)/app.log || echo "无 INFO 日志"

log-error: ## 查看 Error 级别日志
	@grep "ERROR" $(LOGS_DIR)/app.log || echo "无 ERROR 日志"

# ============================================
# 生产环境部署
# ============================================

deploy: build-all ## 部署到生产环境
	@echo "🚀 部署到生产环境..."
	@echo "请使用部署脚本: ./scripts/deploy.ps1"

deploy-vm: build-linux ## 部署到虚拟机
	@echo "🚀 部署到虚拟机..."
	@powershell.exe -File ./scripts/deploy.ps1 -Mode VM -Version $(VERSION)

deploy-docker: docker-build ## 部署到 Docker
	@echo "🚀 部署到 Docker..."
	@powershell.exe -File ./scripts/deploy.ps1 -Mode Docker -Version $(VERSION)

# ============================================
# CI/CD 辅助
# ============================================

ci: check test ## CI 流水线
	@echo "✅ CI 检查通过"

ci-fast: fmt vet test-short ## 快速 CI 检查
	@echo "✅ 快速 CI 检查通过"

# ============================================
# 文档
# ============================================

docs: ## 生成文档
	@echo "📚 生成文档..."
	@which godoc > /dev/null || (echo "❌ 未安装 godoc" && exit 1)
	@godoc -http=:6060 &
	@echo "📚 文档服务已启动: http://localhost:6060"

# ============================================
# 安全检查
# ============================================

security: ## 运行安全检查
	@echo "🔒 运行安全检查..."
	@which gosec > /dev/null || (echo "❌ 未安装 gosec，请运行: go install github.com/securego/gosec/v2/cmd/gosec@latest" && exit 1)
	@gosec ./...
	@echo "✅ 安全检查完成"

# ============================================
# 性能分析
# ============================================

pprof-cpu: ## CPU 性能分析
	@echo "📊 开始 CPU 性能分析 (30秒)..."
	@$(GO) tool pprof -http=:9999 http://localhost:6060/debug/pprof/profile?seconds=30

pprof-mem: ## 内存性能分析
	@echo "📊 开始内存性能分析..."
	@$(GO) tool pprof -http=:9999 http://localhost:6060/debug/pprof/heap
