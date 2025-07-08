@echo off
echo Installing LAMZU Automator...

REM Check if executable exists
if not exist "%CD%\lamzu-automator.exe" (
    echo ❌ lamzu-automator.exe not found!
    echo 🔨 Please run build.bat first to create the executable
    pause
    exit /b 1
)

REM Stop service if it exists
sc stop "LAMZU Automator" 2>nul

REM Remove service if it exists
sc delete "LAMZU Automator" 2>nul

echo.
echo ⚠️  Note: Current version runs as console application, not Windows service
echo.
echo 📋 Installation options:
echo   1. Run manually: lamzu-automator.exe
echo   2. Run in background: lamzu-automator.exe -d
echo   3. Install as scheduled task (recommended)
echo.

set /p choice="Install as scheduled task for auto-startup? (y/n): "
if /i "%choice%"=="y" goto :install_task
if /i "%choice%"=="yes" goto :install_task

echo.
echo ✅ Executable ready at: %CD%\lamzu-automator.exe
echo 🚀 To start manually: lamzu-automator.exe
echo 📋 To run in background: lamzu-automator.exe -d
goto :end

:install_task
echo.
echo 🔐 Choose privilege level:
echo   1. Normal user (may have HID access issues)
echo   2. Administrator (recommended for HID access)
echo.
set /p admin_choice="Run with admin privileges? (y/n): "

echo.
echo 📅 Creating scheduled task for auto-startup...

if /i "%admin_choice%"=="y" goto :admin_task
if /i "%admin_choice%"=="yes" goto :admin_task

REM Create normal user task
schtasks /create /tn "LAMZU Automator" /tr "\"%CD%\lamzu-automator.exe\" -d" /sc onlogon /ru "%USERNAME%" /f 2>nul
goto :task_created

:admin_task
REM Create admin task (requires password or current admin session)
schtasks /create /tn "LAMZU Automator" /tr "\"%CD%\lamzu-automator.exe\" -d" /sc onlogon /rl highest /f 2>nul

:task_created

if %ERRORLEVEL% EQU 0 (
    echo ✅ Scheduled task created successfully!
    echo 🔄 Will start automatically on user login
    echo.
    set /p start_now="Start now? (y/n): "
    if /i "!start_now!"=="y" start "" "%CD%\lamzu-automator.exe" -d
    if /i "!start_now!"=="yes" start "" "%CD%\lamzu-automator.exe" -d
) else (
    echo ❌ Failed to create scheduled task
    echo 💡 Try running as Administrator
)

:end
echo.
pause