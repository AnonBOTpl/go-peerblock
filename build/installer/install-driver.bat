@echo off
REM Install WinDivert kernel driver
REM Must be run as Administrator

echo Installing WinDivert driver...
sc create WinDivert type= kernel start= demand binPath= "%~dp0WinDivert64.sys"
if %errorlevel% neq 0 (
    echo Driver service already exists or failed to create.
)

echo Starting WinDivert driver...
sc start WinDivert
if %errorlevel% neq 0 (
    echo Driver may already be running.
)

echo Done.
