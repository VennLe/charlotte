@echo off
echo Testing Charlotte API startup...
echo.

echo [1/4] Starting application...
start /B go run main.go start -c configs/config.dev.yaml > logs\startup.log 2>&1

echo [2/4] Waiting for service to start...
timeout /t 5 /nobreak > nul

echo [3/4] Checking if service is running...
tasklist /FI "IMAGENAME eq go.exe" | find /I "go.exe" > nul
if %ERRORLEVEL% equ 0 (
    echo [OK] Go process is running
) else (
    echo [FAIL] Go process not found
    goto :showlog
)

echo [4/4] Testing health endpoint...
curl -s http://localhost:8080/health > nul 2>&1
if %ERRORLEVEL% equ 0 (
    echo [OK] Health endpoint is responding
    echo.
    echo Service started successfully!
    echo Health check: http://localhost:8080/health
    echo API docs: http://localhost:8080/swagger/index.html
) else (
    echo [WARN] Health endpoint not responding (service might still be starting)
)

:showlog
if exist logs\startup.log (
    echo.
    echo === Startup Log ===
    type logs\startup.log
) else (
    echo No log file found
)

echo.
echo Press Ctrl+C to stop the service
pause
