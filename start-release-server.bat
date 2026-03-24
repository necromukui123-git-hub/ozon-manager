@echo off
setlocal
title Ozon Manager

cd /d %~dp0

echo ========================================
echo   Ozon Manager
echo ========================================
echo.

if not exist "server.exe" (
    echo [ERROR] server.exe not found.
    pause
    exit /b 1
)

if not exist "config\config.yaml" (
    echo [ERROR] config\config.yaml not found.
    echo Please copy config\config.yaml.example to config\config.yaml
    echo and fill PostgreSQL and JWT settings before starting.
    pause
    exit /b 1
)

if not exist "web\index.html" (
    echo [ERROR] web\index.html not found.
    echo Please rebuild the release package from the source machine.
    pause
    exit /b 1
)

echo Starting Ozon Manager...
echo Management UI: http://127.0.0.1:8080
echo.

server.exe
set EXIT_CODE=%ERRORLEVEL%

if not "%EXIT_CODE%"=="0" (
    echo.
    echo [ERROR] Ozon Manager exited with code %EXIT_CODE%.
    pause
)

endlocal & exit /b %EXIT_CODE%
