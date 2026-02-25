#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Enterprise API 构建部署脚本
.DESCRIPTION
    交叉编译 Go 项目并部署到 VMware Ubuntu 虚拟机
.PARAMETER Version
    版本号，默认为 1.0.0
.PARAMETER VMIP
    虚拟机 IP 地址，默认为 192.168.1.100
.PARAMETER VMUser
    虚拟机用户名，默认为 ubuntu
.PARAMETER SkipDeploy
    跳过部署步骤，仅构建
.EXAMPLE
    .\build.ps1 -Version "1.0.1" -VMIP "192.168.1.50"
#>

[CmdletBinding()]
param(
    [string]$Version = "1.0.0",
    [string]$VMIP = "192.168.1.100",
    [string]$VMUser = "ubuntu",
    [switch]$SkipDeploy
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "Continue"

# 颜色定义
$Colors = @{
    Success = "Green"
    Info    = "Cyan"
    Warning = "Yellow"
    Error   = "Red"
}

function Write-Step {
    param([string]$Message)
    Write-Host "👉 $Message" -ForegroundColor $Colors.Info
}

function Write-Success {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor $Colors.Success
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠️  $Message" -ForegroundColor $Colors.Warning
}

function Write-Error {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor $Colors.Error
}

# 检查依赖
function Test-Dependencies {
    Write-Step "检查依赖..."

    # 检查 Go
    try {
        $goVersion = go version
        Write-Success "Go 已安装: $goVersion"
    } catch {
        Write-Error "未找到 Go，请先安装 Go 1.21+"
        exit 1
    }

    # 检查 SSH 连接
    if (-not $SkipDeploy) {
        Write-Step "测试 SSH 连接到 $VMIP..."
        try {
            $testResult = ssh -o ConnectTimeout=5 -o BatchMode=yes $VMUser@$VMIP "echo 'SSH OK'" 2>&1
            if ($testResult -eq "SSH OK") {
                Write-Success "SSH 连接正常"
            } else {
                throw "SSH 连接失败"
            }
        } catch {
            Write-Error "无法连接到 VM ($VMIP)，请检查："
            Write-Host "  1. VM 是否运行"
            Write-Host "  2. IP 地址是否正确"
            Write-Host "  3. SSH 密钥是否配置"
            exit 1
        }
    }
}

# 清理旧构建
function Clear-OldBuilds {
    Write-Step "清理旧构建..."
    if (Test-Path "bin") {
        Remove-Item -Recurse -Force "bin" -ErrorAction SilentlyContinue
    }
    New-Item -ItemType Directory -Force -Path "bin" | Out-Null
    Write-Success "清理完成"
}

# 编译
function Build-Project {
    Write-Step "开始交叉编译 (Linux amd64)..."

    # 设置环境变量
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"

    # 构建参数
    $ldflags = "-s -w -X main.Version=$Version -X main.BuildTime=$(Get-Date -Format 'yyyy-MM-dd_HH:mm:ss')"

    # 执行构建
    $buildCmd = "go build -ldflags `"$ldflags`" -o bin/enterprise-api cmd/server/main.go"
    Write-Host "执行: $buildCmd" -ForegroundColor Gray

    Invoke-Expression $buildCmd

    if ($LASTEXITCODE -ne 0) {
        Write-Error "构建失败"
        exit 1
    }

    # 检查二进制文件
    $binaryPath = "bin/enterprise-api"
    if (Test-Path $binaryPath) {
        $size = (Get-Item $binaryPath).Length / 1MB
        Write-Success "构建成功: $binaryPath ($([math]::Round($size, 2)) MB)"
    } else {
        Write-Error "构建产物未找到"
        exit 1
    }

    # 尝试压缩 (如果安装了 upx)
    try {
        $upxCheck = upx --version 2>$null
        if ($upxCheck) {
            Write-Step "压缩二进制文件..."
            upx -9 -q bin/enterprise-api
            $newSize = (Get-Item $binaryPath).Length / 1MB
            Write-Success "压缩完成: $([math]::Round($newSize, 2)) MB"
        }
    } catch {
        Write-Warning "未找到 UPX，跳过压缩"
    }
}

# 复制配置文件
function Copy-ConfigFiles {
    Write-Step "复制配置文件..."

    # 确保配置文件存在
    if (-not (Test-Path "configs/config.yaml")) {
        Write-Error "配置文件 configs/config.yaml 不存在"
        exit 1
    }

    Copy-Item "configs/config.yaml" "bin/config.yaml" -Force

    # 创建启动脚本
    $startScript = @"
#!/bin/bash
cd /opt/enterprise-api
./enterprise-api start --config=/opt/enterprise-api/config.yaml
"@
    $startScript | Out-File -FilePath "bin/start.sh" -Encoding UTF8NoBOM -NoNewline

    Write-Success "配置文件准备完成"
}

# 部署到 VM
function Deploy-ToVM {
    if ($SkipDeploy) {
        Write-Warning "跳过部署步骤"
        return
    }

    Write-Step "部署到虚拟机 $VMIP..."

    $remotePath = "/opt/enterprise-api"
    $backupPath = "/opt/enterprise-api-backup-$(Get-Date -Format 'yyyyMMdd-HHmmss')"

    # 创建远程目录
    ssh $VMUser@$VMIP "sudo mkdir -p $remotePath/logs && sudo chown -R $VMUser`:$VMUser $remotePath"

    # 备份旧版本
    Write-Step "备份旧版本..."
    ssh $VMUser@$VMIP @"
        if [ -f $remotePath/enterprise-api ]; then
            sudo mkdir -p $backupPath
            sudo cp $remotePath/enterprise-api $backupPath/
            sudo cp $remotePath/config.yaml $backupPath/ 2>/dev/null || true
            echo "备份完成: $backupPath"
        fi
"@

    # 上传文件
    Write-Step "上传文件..."
    scp bin/enterprise-api bin/config.yaml bin/start.sh ${VMUser}@${VMIP}:$remotePath/

    # 设置权限
    ssh $VMUser@$VMIP "chmod +x $remotePath/enterprise-api $remotePath/start.sh"

    # 检查 systemd 服务
    Write-Step "配置系统服务..."
    $serviceExists = ssh $VMUser@$VMIP "systemctl list-unit-files | grep enterprise-api" 2>$null
    if (-not $serviceExists) {
        Write-Warning "systemd 服务未配置，请手动配置"
        Write-Host @"
请在 VM 中执行以下命令：
sudo cp deployments/systemd/enterprise-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable enterprise-api
"@
    } else {
        # 重启服务
        Write-Step "重启服务..."
        ssh $VMUser@$VMIP "sudo systemctl restart enterprise-api"

        # 等待服务启动
        Start-Sleep -Seconds 2

        # 检查状态
        $serviceStatus = ssh $VMUser@$VMIP "systemctl is-active enterprise-api" 2>&1
        if ($serviceStatus -eq "active") {
            Write-Success "服务启动成功"
        } else {
            Write-Error "服务启动失败，请检查日志: sudo journalctl -u enterprise-api -n 50"
        }
    }

    # 健康检查
    Write-Step "健康检查..."
    Start-Sleep -Seconds 3
    try {
        $health = Invoke-RestMethod -Uri "http://${VMIP}:8080/health" -TimeoutSec 10
        Write-Success "服务健康: $($health | ConvertTo-Json -Compress)"
    } catch {
        Write-Warning "健康检查失败: $_"
    }
}

# 主流程
function Main {
    Write-Host @"
========================================
  Enterprise API 构建部署脚本
  Version: $Version
  Target: $VMIP
========================================
"@ -ForegroundColor $Colors.Info

    $startTime = Get-Date

    Test-Dependencies
    Clear-OldBuilds
    Build-Project
    Copy-ConfigFiles
    Deploy-ToVM

    $duration = (Get-Date) - $startTime
    Write-Success "全部完成！耗时: $($duration.ToString('mm\:ss'))"

    if (-not $SkipDeploy) {
        Write-Host @"
========================================
访问地址:
  - API:    http://$VMIP`:8080
  - Health: http://$VMIP`:8080/health

常用命令:
  查看日志: ssh $VMUser@$VMIP "sudo journalctl -u enterprise-api -f"
  重启服务: ssh $VMUser@$VMIP "sudo systemctl restart enterprise-api"
========================================
"@ -ForegroundColor $Colors.Info
    }
}

# 执行
Main