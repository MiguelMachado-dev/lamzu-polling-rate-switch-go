@echo off
echo 🔧 LAMZU Automator Build Script (Console Mode)
echo =============================================
echo.

REM Verificar se estamos no diretório correto
if not exist "main.go" (
    echo ❌ Error: main.go not found. Please run from project directory.
    pause
    exit /b 1
)

echo 📦 Downloading dependencies...
"C:\Program Files\Go\bin\go.exe" mod tidy
if %errorlevel% neq 0 (
    echo ❌ Failed to download dependencies
    pause
    exit /b 1
)

echo.
echo 🏗️ Building LAMZU Automator (Console mode - with console window)...
"C:\Program Files\Go\bin\go.exe" build -ldflags="-s -w" -o lamzu-automator-console.exe .
if %errorlevel% neq 0 (
    echo ❌ Build failed
    pause
    exit /b 1
)

echo.
echo ✅ Build completed successfully!
echo.
echo 📁 Generated file: lamzu-automator-console.exe
echo.
echo 🚀 Usage:
echo   lamzu-automator-console.exe debug    - Debug device connection
echo   lamzu-automator-console.exe          - Start monitoring
echo   lamzu-automator-console.exe set 2000 - Set polling rate manually
echo   lamzu-automator-console.exe -h       - Show help
echo.
echo ⚠️  Remember to run as Administrator for HID access!
echo.
echo 💡 This version shows console output for debugging.
echo    Use build.bat for GUI version without console window.
echo.
pause