@echo off
setlocal EnableExtensions EnableDelayedExpansion
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
call :find_listening_pid 8080 BACKEND_PID
if defined BACKEND_PID (
    echo [1/2] Backend already listening on port 8080 ^(PID !BACKEND_PID!^), skipping start.
) else (
    echo [1/2] Starting backend on port 8080...
    start "Ozon Backend" cmd /k "cd /d %~dp0backend && go run ./cmd/server"
    echo Waiting for backend...
    timeout /t 3 /nobreak >nul
)

:: Start frontend
call :find_listening_pid 5173 FRONTEND_PID
if defined FRONTEND_PID (
    echo [2/2] Frontend already listening on port 5173 ^(PID !FRONTEND_PID!^), skipping start.
) else (
    echo [2/2] Starting frontend on port 5173...
    start "Ozon Frontend" cmd /k "cd /d %~dp0frontend && npm run dev"
)

echo.
echo ========================================
echo   Services started:
echo   - Backend:  http://localhost:8080
echo   - Frontend: http://localhost:5173
echo ========================================
echo.
echo Press any key to close this window
pause >nul
exit /b 0

:find_listening_pid
set "%~2="
for /f "tokens=5" %%p in ('netstat -ano ^| findstr LISTENING ^| findstr /C:":%~1 "') do (
    set "%~2=%%p"
    goto :eof
)
goto :eof
