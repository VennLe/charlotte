@echo off
setlocal enabledelayedexpansion

REM Charlotte API Startup Script
REM Windows Batch Version

set APP_NAME=charlotte
set BIN_DIR=bin

if "%1"=="" goto :help
if "%1"=="help" goto :help
if "%1"=="run" goto :run
if "%1"=="start" goto :start
if "%1"=="dev" goto :dev
if "%1"=="build" goto :build
if "%1"=="clean" goto :clean
if "%1"=="migrate" goto :migrate
if "%1"=="config" goto :config
if "%1"=="version" goto :version
if "%1"=="test" goto :test
if "%1"=="deps" goto :deps
if "%1"=="fmt" goto :fmt
if "%1"=="lint" goto :lint
goto :help

:help
echo Charlotte API - Startup Script
echo.
echo Usage:
echo   start.bat [command]
echo.
echo Available Commands:
echo   run              - Run application directly (go run)
echo   start            - Build and start application
echo   dev              - Development mode (requires air)
echo   build            - Build application
echo   clean            - Clean build artifacts
echo   migrate          - Run database migration
echo   config           - Show current configuration
echo   config-validate  - Validate configuration
echo   config-env       - Show environment variable mapping
echo   version          - Show version information
echo   test             - Run tests
echo   deps             - Download dependencies
echo   fmt              - Format code
echo   lint             - Run code linter
echo.
goto :end

:run
echo [INFO] Starting application...
go run main.go start
goto :end

:start
call :build
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Build failed
    goto :end
)
echo [INFO] Starting %APP_NAME%...
if not exist %BIN_DIR% mkdir %BIN_DIR%
if not exist logs mkdir logs
%BIN_DIR%\%APP_NAME%.exe start
goto :end

:dev
echo [INFO] Starting development mode...
where air >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo [ERROR] air not found, please run: go install github.com/cosmtrek/air@latest
    goto :end
)
air
goto :end

:build
echo [INFO] Building %APP_NAME%...
if not exist %BIN_DIR% mkdir %BIN_DIR%
go build -v -ldflags "-s -w" -o %BIN_DIR%\%APP_NAME%.exe main.go
if %ERRORLEVEL% equ 0 (
    echo [SUCCESS] Build completed: %BIN_DIR%\%APP_NAME%.exe
) else (
    echo [ERROR] Build failed
)
goto :end

:clean
echo [INFO] Cleaning build artifacts...
if exist %BIN_DIR% rmdir /s /q %BIN_DIR%
if exist coverage.out del /q coverage.out
if exist coverage.html del /q coverage.html
echo [SUCCESS] Clean completed
goto :end

:migrate
echo [INFO] Running database migration...
go run main.go migrate
goto :end

:config
echo [INFO] Showing current configuration...
go run main.go config show
goto :end

:config-validate
echo [INFO] Validating configuration...
go run main.go config validate
goto :end

:config-env
echo [INFO] Showing environment variable mapping...
go run main.go config env
goto :end

:version
echo [INFO] Showing version information...
go run main.go version
goto :end

:test
echo [INFO] Running tests...
go test -v -race -coverprofile=coverage.out ./...
if %ERRORLEVEL% equ 0 (
    echo [SUCCESS] Tests completed
    echo Coverage report: coverage.html
    go tool cover -html=coverage.out -o coverage.html
)
goto :end

:deps
echo [INFO] Downloading dependencies...
go mod download
go mod tidy
echo [SUCCESS] Dependencies updated
goto :end

:fmt
echo [INFO] Formatting code...
go fmt ./...
echo [SUCCESS] Format completed
goto :end

:lint
echo [INFO] Running code linter...
where golangci-lint >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo [ERROR] golangci-lint not found
    echo Please run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    goto :end
)
golangci-lint run ./...
goto :end

:end
endlocal
