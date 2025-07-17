@echo off

@REM Auto-elevation script
if not "%1"=="am_admin" (
    powershell -Command "Start-Process -FilePath '%0' -ArgumentList 'am_admin' -Verb RunAs"
    exit /b
)

@REM Check if running as administrator
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Failed to elevate to administrator.
    pause
    exit /b 1
)

@REM Run this script as administrator
echo.
echo ✅ Running as administrator...

echo Restarting LAMZU Automator...
echo.
@REM Define o diretório do script como diretório de trabalho
cd /d "%~dp0"

@REM Lista de executáveis possíveis (em ordem de preferência)
set "executable="
if exist "lamzu-automator.exe" set "executable=lamzu-automator.exe"

if "%executable%"=="" (
    echo ❌ LAMZU Automator executable not found!
    echo 📁 Current directory: %CD%
    echo 🔍 Looking for: lamzu-automator.exe
    echo 🔨 Please run build.bat first to create the executable
    dir *.exe /b 2>nul
    pause
    exit /b 1
)

echo 📁 Found executable: %executable%
echo Stopping current instance...
taskkill /f /im "%executable%" 2>nul
if %errorlevel% neq 0 (
    echo ❌ Failed to stop current instance. It may not be running.
) else (
    echo ✅ Current instance stopped.
)
echo Starting new instance...

start "" "%CD%\%executable%"
if %errorlevel% neq 0 (
    echo ❌ Failed to start new instance.
) else (
    echo ✅ New instance started successfully.
)

echo.
echo ✅ Process completed. Press any key to exit...
pause >nul