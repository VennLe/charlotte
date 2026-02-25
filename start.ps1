<#
.SYNOPSIS
    Charlotte API Startup Script
.DESCRIPTION
    PowerShell script for managing Charlotte API build, run, and deployment
.PARAMETER Command
    Command to execute (default: help)
.EXAMPLE
    .\start.ps1 run
.EXAMPLE
    .\start.ps1 build
.EXAMPLE
    .\start.ps1 start
#>

[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet("run", "start", "dev", "build", "clean", "migrate", "config", "config-validate", "config-env", "version", "test", "deps", "fmt", "lint", "docker", "help")]
    [string]$Command = "help"
)

$ErrorActionPreference = "Stop"
$AppName = "charlotte"
$BinDir = "bin"
$LogFile = "logs/app.log"

function Write-Step {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Show-Help {
    Write-Host @"
Charlotte API - Startup Script

Usage:
  .\start.ps1 [command]

Available Commands:
  run              - Run application directly (go run)
  start            - Build and start application
  dev              - Development mode (requires air)
  build            - Build application
  clean            - Clean build artifacts
  migrate          - Run database migration
  config           - Show current configuration
  config-validate  - Validate configuration
  config-env       - Show environment variable mapping
  version          - Show version information
  test             - Run tests
  deps             - Download dependencies
  fmt              - Format code
  lint             - Run code linter
  docker           - Build and run Docker container
  help             - Show this help message
"@ -ForegroundColor Yellow
}

function Invoke-Run {
    Write-Step "Starting application..."
    go run main.go start
}

function Invoke-Start {
    Invoke-Build
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Build failed"
        return
    }

    Write-Step "Starting $AppName..."
    if (!(Test-Path $BinDir)) { New-Item -ItemType Directory -Path $BinDir | Out-Null }
    if (!(Test-Path "logs")) { New-Item -ItemType Directory -Path "logs" | Out-Null }

    & "$BinDir/$AppName.exe" start
}

function Invoke-Dev {
    Write-Step "Starting development mode..."
    if (!(Get-Command air -ErrorAction SilentlyContinue)) {
        Write-Error "air not found, please run: go install github.com/cosmtrek/air@latest"
        return
    }
    air
}

function Invoke-Build {
    Write-Step "Building $AppName..."
    if (!(Test-Path $BinDir)) { New-Item -ItemType Directory -Path $BinDir | Out-Null }

    $ldflags = "-s -w"
    go build -v -ldflags $ldflags -o "$BinDir/$AppName.exe" main.go

    if ($LASTEXITCODE -eq 0) {
        Write-Success "Build completed: $BinDir/$AppName.exe"
    } else {
        Write-Error "Build failed"
    }
}

function Invoke-Clean {
    Write-Step "Cleaning build artifacts..."
    if (Test-Path $BinDir) { Remove-Item -Recurse -Force $BinDir }
    if (Test-Path "coverage.out") { Remove-Item -Force "coverage.out" }
    if (Test-Path "coverage.html") { Remove-Item -Force "coverage.html" }
    Write-Success "Clean completed"
}

function Invoke-Migrate {
    Write-Step "Running database migration..."
    go run main.go migrate
}

function Invoke-Config {
    Write-Step "Showing current configuration..."
    go run main.go config show
}

function Invoke-ConfigValidate {
    Write-Step "Validating configuration..."
    go run main.go config validate
}

function Invoke-ConfigEnv {
    Write-Step "Showing environment variable mapping..."
    go run main.go config env
}

function Invoke-Version {
    Write-Step "Showing version information..."
    go run main.go version
}

function Invoke-Test {
    Write-Step "Running tests..."
    go test -v -race -coverprofile=coverage.out ./...

    if ($LASTEXITCODE -eq 0) {
        Write-Success "Tests completed"
        Write-Host "Coverage report: coverage.html" -ForegroundColor Cyan
        go tool cover -html=coverage.out -o coverage.html
    }
}

function Invoke-Deps {
    Write-Step "Downloading dependencies..."
    go mod download
    go mod tidy
    Write-Success "Dependencies updated"
}

function Invoke-Fmt {
    Write-Step "Formatting code..."
    go fmt ./...
    Write-Success "Format completed"
}

function Invoke-Lint {
    Write-Step "Running code linter..."
    if (!(Get-Command golangci-lint -ErrorAction SilentlyContinue)) {
        Write-Error "golangci-lint not found"
        Write-Host "Please run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" -ForegroundColor Yellow
        return
    }
    golangci-lint run ./...
}

function Invoke-Docker {
    Write-Step "Building Docker image..."
    docker build -t $AppName:latest .

    if ($LASTEXITCODE -eq 0) {
        Write-Success "Docker image built successfully"

        Write-Step "Running Docker container..."
        docker run -d --name $AppName -p 8080:8080 --restart unless-stopped $AppName:latest

        if ($LASTEXITCODE -eq 0) {
            Write-Success "Docker container started successfully"
        }
    }
}

# Main execution
switch ($Command) {
    "run" { Invoke-Run }
    "start" { Invoke-Start }
    "dev" { Invoke-Dev }
    "build" { Invoke-Build }
    "clean" { Invoke-Clean }
    "migrate" { Invoke-Migrate }
    "config" { Invoke-Config }
    "config-validate" { Invoke-ConfigValidate }
    "config-env" { Invoke-ConfigEnv }
    "version" { Invoke-Version }
    "test" { Invoke-Test }
    "deps" { Invoke-Deps }
    "fmt" { Invoke-Fmt }
    "lint" { Invoke-Lint }
    "docker" { Invoke-Docker }
    "help" { Show-Help }
    default { Show-Help }
}
