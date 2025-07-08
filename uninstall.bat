@echo off
echo Uninstalling LAMZU Automator...

REM Stop any running processes
taskkill /f /im "lamzu-automator.exe" 2>nul

REM Remove Windows service (if exists)
sc stop "LAMZU Automator" 2>nul
sc delete "LAMZU Automator" 2>nul

REM Remove scheduled task (if exists)
schtasks /delete /tn "LAMZU Automator" /f 2>nul

echo ✅ LAMZU Automator uninstalled!
pause