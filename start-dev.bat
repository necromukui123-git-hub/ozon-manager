@echo off
title Ozon Manager Launcher
echo ========================================
echo   Ozon Manager Dev Environment
echo ========================================
echo.

:: Check backend
if not exist "%~dp0backend\cmd\server\main.go" (
    echo [ERROR] Backend entry file not found
    pause
    exit /b 1
)

:: Check frontend
if not exist "%~dp0frontend\package.json" (
    echo [ERROR] Frontend package.json not found
    pause
    exit /b 1
)

:: Check backend config
if not exist "%~dp0backend\config\config.yaml" (
    echo [WARNING] Backend config not found!
    echo Please copy backend\config\config.yaml.example to config.yaml
    pause
    exit /b 1
)

:: Start backend
echo [1/2] Starting backend (port 8080)...
start "Ozon Backend" cmd /k "cd /d %~dp0backend && go run cmd/server/main.go"

:: Wait for backend
echo Waiting for backend...
timeout /t 3 /nobreak >nul

:: Start frontend
echo [2/2] Starting frontend (port 5173)...
start "Ozon Frontend" cmd /k "cd /d %~dp0frontend && npm run dev"

echo.
echo ========================================
echo   Services started:
echo   - Backend:  http://localhost:8080
echo   - Frontend: http://localhost:5173
echo ========================================
echo.
echo Press any key to close this window
pause >nul
