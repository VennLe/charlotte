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
        
        # 使用 ControlMaster 进行更可靠的连接测试
        $sshArgs = @(
            "-o", "ConnectTimeout=5"
            "-o", "StrictHostKeyChecking=no"
            "-o", "BatchMode=yes"
        )
        
        try {
            $null = ssh @sshArgs "$VMUser@$VMIP" "echo ok" 2>&1
            # 检查 $LASTEXITCODE 而不是输出内容
            if ($LASTEXITCODE -eq 0) {
                Write-Success "SSH 连接正常"
            } else {
                throw "SSH 认证失败或连接超时"
            }
        } catch {
            Write-Error "无法连接到 VM ($VMIP)，请检查："
            Write-Host "  1. VM 是否运行"
            Write-Host "  2. IP 地址是否正确"
            Write-Host "  3. SSH 密钥是否配置"
            Write-Host "  4. 密码认证是否启用 (如无密钥)"
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

    # 保存原始环境变量
    $origGOOS = $env:GOOS
    $origGOARCH = $env:GOARCH
    $origCGO = $env:CGO_ENABLED

    try {
        # 设置环境变量
        $env:GOOS = "linux"
        $env:GOARCH = "amd64"
        $env:CGO_ENABLED = "0"

        # 构建参数 - 使用数组避免引号问题
        $buildTime = Get-Date -Format "yyyy-MM-dd_HH:mm:ss"
        $ldflags = @(
            "-s", "-w",
            "-X", "main.Version=$Version",
            "-X", "main.BuildTime=$buildTime"
        )

        # 直接调用命令，避免 Invoke-Expression
        $outputPath = "bin/enterprise-api"
        Write-Host "执行: go build -ldflags $(($ldflags -join ' ')) -o $outputPath cmd/server/main.go" -ForegroundColor Gray

        & go build -ldflags $ldflags -o $outputPath cmd/server/main.go

        if ($LASTEXITCODE -ne 0) {
            Write-Error "构建失败，退出码: $LASTEXITCODE"
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
            $null = Get-Command upx -ErrorAction Stop
            Write-Step "压缩二进制文件..."
            upx -9 -q bin/enterprise-api
            $newSize = (Get-Item $binaryPath).Length / 1MB
            Write-Success "压缩完成: $([math]::Round($newSize, 2)) MB"
        } catch {
            Write-Warning "未找到 UPX，跳过压缩"
        }
    }
    finally {
        # 恢复环境变量
        $env:GOOS = $origGOOS
        $env:GOARCH = $origGOARCH
        $env:CGO_ENABLED = $origCGO
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
    
    # 使用 utf8 编码，兼容 PowerShell 5.x 和 7+
    $startScript | Out-File -FilePath "bin/start.sh" -Encoding utf8 -NoNewline

    Write-Success "配置文件准备完成"
}

# 上传文件 (带重试)
function Upload-File {
    param(
        [string]$LocalPath,
        [string]$RemotePath,
        [int]$MaxRetries = 3
    )

    $retryCount = 0
    while ($retryCount -lt $MaxRetries) {
        try {
            scp -o "ConnectTimeout=10" $LocalPath "${VMUser}@${VMIP}:${RemotePath}"
            return
        } catch {
            $retryCount++
            if ($retryCount -lt $MaxRetries) {
                Write-Warning "上传失败，$((MaxRetries - $retryCount)) 秒后重试..."
                Start-Sleep -Seconds 3
            } else {
                throw "文件上传失败: $LocalPath -> $RemotePath"
            }
        }
    }
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

    # 上传文件 (带重试)
    Write-Step "上传文件..."
    Upload-File "bin/enterprise-api" "$remotePath/enterprise-api"
    Upload-File "bin/config.yaml" "$remotePath/config.yaml"
    Upload-File "bin/start.sh" "$remotePath/start.sh"

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
    
    $healthOk = $false
    try {
        $health = Invoke-RestMethod -Uri "http://${VMIP}:8080/health" -TimeoutSec 10
        if ($health.status -eq "ok" -or $health -eq "OK" -or $health -eq "200") {
            Write-Success "服务健康: $($health | ConvertTo-Json -Compress)"
            $healthOk = $true
        } else {
            Write-Warning "服务返回非正常状态: $($health | ConvertTo-Json -Compress)"
        }
    } catch {
        Write-Warning "健康检查请求失败: $($_.Exception.Message)"
    }
    
    if (-not $healthOk) {
        Write-Warning "建议手动检查服务状态"
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
