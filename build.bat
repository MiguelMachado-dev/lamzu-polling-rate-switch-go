@echo off
echo 🔧 LAMZU Automator Build Script
echo ==============================
echo.

REM Verificar se estamos no diretório correto
if not exist "main.go" (
    echo ❌ Error: main.go not found. Please run from project directory.
    pause
    exit /b 1
)

echo 📦 Downloading dependencies...
go mod tidy
if %errorlevel% neq 0 (
    echo ❌ Failed to download dependencies
    pause
    exit /b 1
)

echo.
echo 🏗️ Building LAMZU Automator (GUI mode - no console window)...
go build -ldflags="-s -w -H=windowsgui" -o lamzu-automator.exe .
if %errorlevel% neq 0 (
    echo ❌ Build failed
    pause
    exit /b 1
)

echo.
echo ✅ Build completed successfully!
echo.
echo 📁 Generated file: lamzu-automator.exe
echo.
echo 🚀 Usage:
echo   lamzu-automator.exe debug    - Debug device connection
echo   lamzu-automator.exe          - Start monitoring
echo   lamzu-automator.exe set 2000 - Set polling rate manually
echo   lamzu-automator.exe -h       - Show help
echo.
echo ⚠️  Remember to run as Administrator for HID access!
echo.
pause
