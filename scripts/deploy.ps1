#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Charlotte API 部署脚本
.DESCRIPTION
    支持多种部署方式：VM、Docker、Docker Compose
.PARAMETER Mode
    部署模式: VM, Docker, Compose (默认: VM)
.PARAMETER Version
    版本号，默认为 1.0.0
.PARAMETER VMIP
    虚拟机 IP 地址（VM 模式）
.PARAMETER VMUser
    虚拟机用户名（VM 模式）
.PARAMETER DockerRegistry
    Docker 镜像仓库地址
.PARAMETER SkipBuild
    跳过构建步骤
.PARAMETER SkipMigrate
    跳过数据库迁移
.EXAMPLE
    .\deploy.ps1 -Mode VM -Version "1.0.1" -VMIP "192.168.1.100"
.EXAMPLE
    .\deploy.ps1 -Mode Docker -Version "1.0.1" -DockerRegistry "registry.example.com"
.EXAMPLE
    .\deploy.ps1 -Mode Compose
#>

[CmdletBinding()]
param(
    [ValidateSet("VM", "Docker", "Compose")]
    [string]$Mode = "VM",

    [string]$Version = "1.0.0",

    [string]$VMIP = "192.168.1.100",
    [string]$VMUser = "ubuntu",

    [string]$DockerRegistry = "",

    [switch]$SkipBuild,
    [switch]$SkipMigrate
)

$ErrorActionPreference = "Stop"
$AppName = "charlotte"
$ProjectRoot = $PSScriptRoot | Split-Path -Parent

# 颜色定义
$Colors = @{
    Success = "Green"
    Info    = "Cyan"
    Warning = "Yellow"
    Error   = "Red"
}

function Write-Step { param([string]$Message) Write-Host "👉 $Message" -ForegroundColor $Colors.Info }
function Write-Success { param([string]$Message) Write-Host "✅ $Message" -ForegroundColor $Colors.Success }
function Write-Warning { param([string]$Message) Write-Host "⚠️  $Message" -ForegroundColor $Colors.Warning }
function Write-Error { param([string]$Message) Write-Host "❌ $Message" -ForegroundColor $Colors.Error }

# ============ VM 部署相关函数 ============

function Test-VMConnection {
    param([string]$IP, [string]$User)
    Write-Step "测试 SSH 连接到 $IP..."
    try {
        $testResult = ssh -o ConnectTimeout=5 -o BatchMode=yes $User@$IP "echo 'SSH OK'" 2>&1
        if ($testResult -eq "SSH OK") {
            Write-Success "SSH 连接正常"
            return $true
        }
    } catch { }
    Write-Error "无法连接到 $IP，请检查网络和 SSH 配置"
    return $false
}

function Backup-RemoteVersion {
    param([string]$IP, [string]$User, [string]$RemotePath)
    Write-Step "备份旧版本..."
    $backupPath = "$RemotePath-backup-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
    ssh $User@$IP @"
if [ -f $RemotePath/$AppName ]; then
    sudo mkdir -p $backupPath
    sudo cp $RemotePath/$AppName $backupPath/
    sudo cp $RemotePath/config.yaml $backupPath/ 2>/dev/null || true
    echo "备份完成: $backupPath"
fi
"@
}

function Deploy-ToVM {
    param([string]$IP, [string]$User, [string]$Ver)

    $remotePath = "/opt/$AppName"

    if (-not (Test-VMConnection -IP $VMIP -User $VMUser)) {
        exit 1
    }

    # 备份
    Backup-RemoteVersion -IP $VMIP -User $VMUser -RemotePath $remotePath

    # 创建远程目录
    Write-Step "创建远程目录..."
    ssh $VMUser@$VMIP "sudo mkdir -p $remotePath/logs && sudo chown -R $VMUser`:$VMUser $remotePath"

    # 上传文件
    Write-Step "上传文件..."
    scp bin/$AppName bin/config.yaml ${VMUser}@${VMIP}:$remotePath/

    # 设置权限
    ssh $VMUser@$VMIP "chmod +x $remotePath/$AppName"

    # 执行数据库迁移（可选）
    if (-not $SkipMigrate) {
        Write-Step "执行数据库迁移..."
        ssh $VMUser@$VMIP "cd $remotePath && ./$AppName migrate"
    }

    # 重启服务
    Write-Step "重启服务..."
    ssh $VMUser@$VMIP "sudo systemctl restart $AppName 2>/dev/null || sudo pkill $AppName; sleep 2; cd $remotePath && nohup ./$AppName start --config=$remotePath/config.yaml > $remotePath/logs/app.log 2>&1 &"

    # 健康检查
    Start-Sleep -Seconds 3
    try {
        $health = Invoke-RestMethod -Uri "http://${VMIP}:8080/health" -TimeoutSec 10
        Write-Success "服务健康: $($health | ConvertTo-Json -Compress)"
    } catch {
        Write-Warning "健康检查失败: $_"
    }
}

# ============ Docker 部署相关函数 ============

function Build-DockerImage {
    param([string]$Ver, [string]$Registry)

    Write-Step "构建 Docker 镜像..."

    $imageName = if ($Registry) { "$Registry/$AppName" } else { $AppName }
    $imageTag = "$imageName:$Ver"
    $imageTagLatest = "$imageName:latest"

    # 创建 Dockerfile
    $dockerfile = @"
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o $AppName cmd/server/main.go

FROM alpine:3.18
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/$AppName .
COPY configs/config.production.yaml ./config.yaml
EXPOSE 8080
CMD ["./$AppName", "start", "--config=./config.yaml"]
"@

    # 检查是否需要多阶段构建（根据实际项目结构调整）
    # 对于 Charlotte 项目，使用更简单的 Dockerfile
    $simpleDockerfile = @"
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.Version=$Ver" -o $AppName ./main.go

FROM alpine:3.18
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/$AppName .
COPY configs/config.production.yaml ./config.yaml
EXPOSE 8080
ENV GIN_MODE=release
CMD ["./$AppName", "start", "--config=./config.yaml"]
"@

    $simpleDockerfile | Out-File -FilePath "Dockerfile" -Encoding UTF8NoBOM

    # 构建镜像
    docker build -t $imageTag -t $imageTagLatest .

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Docker 镜像构建失败"
        exit 1
    }

    Write-Success "镜像构建成功: $imageTag"

    # 推送到仓库（如果指定）
    if ($Registry) {
        Write-Step "推送镜像到仓库..."
        docker push $imageTag
        docker push $imageTagLatest
        Write-Success "镜像推送成功"
    }
}

function Deploy-ToDocker {
    param([string]$Ver, [string]$Registry)

    $imageName = if ($Registry) { "$Registry/$AppName" } else { $AppName }
    $imageTag = "$imageName:$Ver"

    Write-Step "停止旧容器..."
    docker stop $AppName 2>$null | Out-Null
    docker rm $AppName 2>$null | Out-Null

    Write-Step "启动新容器..."
    docker run -d `
        --name $AppName `
        --restart unless-stopped `
        -p 8080:8080 `
        -e GIN_MODE=release `
        -v $(Resolve-Path "configs/config.production.yaml"):/app/config.yaml:ro `
        $imageTag

    Write-Success "容器启动成功"

    # 健康检查
    Start-Sleep -Seconds 3
    try {
        $health = Invoke-RestMethod -Uri "http://localhost:8080/health" -TimeoutSec 10
        Write-Success "服务健康: $($health | ConvertTo-Json -Compress)"
    } catch {
        Write-Warning "健康检查失败: $_"
    }
}

# ============ Docker Compose 部署相关函数 ============

function Deploy-ToCompose {
    Write-Step "使用 Docker Compose 部署..."

    # 创建 docker-compose.yml（如果不存在）
    $composeFile = @"
version: '3.8'

services:
  charlotte:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: charlotte
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - GIN_MODE=release
      - CHARLOTTE_SERVER_PORT=8080
    volumes:
      - ./configs/config.production.yaml:/app/config.yaml:ro
    depends_on:
      - redis
      - postgres
      - kafka
    networks:
      - charlotte-net

  postgres:
    image: postgres:15-alpine
    container_name: charlotte-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: charlotte
      POSTGRES_PASSWORD: charlotte123
      POSTGRES_DB: charlotte
    volumes:
      - postgres-data:/var/lib/postgresql/data
    networks:
      - charlotte-net

  redis:
    image: redis:7-alpine
    container_name: charlotte-redis
    restart: unless-stopped
    command: redis-server --requirepass redis123
    volumes:
      - redis-data:/data
    networks:
      - charlotte-net

  kafka:
    image: confluentinc/cp-kafka:7.5.0
    container_name: charlotte-kafka
    restart: unless-stopped
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    depends_on:
      - zookeeper
    networks:
      - charlotte-net

  zookeeper:
    image: confluentinc/cp-zookeeper:7.5.0
    container_name: charlotte-zookeeper
    restart: unless-stopped
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000
    networks:
      - charlotte-net

networks:
  charlotte-net:
    driver: bridge

volumes:
  postgres-data:
  redis-data:
"@

    if (-not (Test-Path "docker-compose.yml")) {
        $composeFile | Out-File -FilePath "docker-compose.yml" -Encoding UTF8NoBOM
        Write-Success "创建 docker-compose.yml"
    }

    # 构建并启动
    docker-compose up -d --build

    Write-Success "Docker Compose 部署完成"

    # 显示状态
    docker-compose ps
}

# ============ 主流程 ============

function Main {
    Write-Host @"
========================================
  Charlotte API 部署脚本
  模式: $Mode
  版本: $Version
========================================
"@ -ForegroundColor $Colors.Info

    $startTime = Get-Date

    switch ($Mode) {
        "VM" {
            if (-not $SkipBuild) {
                # 调用 build.ps1 构建
                Write-Step "调用构建脚本..."
                & "$ProjectRoot/scripts/build.ps1" -Version $Version -SkipDeploy
            }
            Deploy-ToVM -IP $VMIP -User $VMUser -Ver $Version
        }
        "Docker" {
            if (-not $SkipBuild) {
                Build-DockerImage -Ver $Version -Registry $DockerRegistry
            }
            Deploy-ToDocker -Ver $Version -Registry $DockerRegistry
        }
        "Compose" {
            Deploy-ToCompose
        }
    }

    $duration = (Get-Date) - $startTime
    Write-Success "部署完成！耗时: $($duration.ToString('mm\:ss'))"
}

Main
