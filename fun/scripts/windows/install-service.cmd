@echo off
REM Windows Service Installation Script for Fun Server

echo Installing Fun Server as a Windows service...

REM Get the installation directory from the MSI (should be passed as %1)
set INSTALL_DIR=%1
if "%INSTALL_DIR%"=="" set INSTALL_DIR=%ProgramFiles%\Fun

REM Create the Windows service using sc.exe
sc create fun binPath= "\"%INSTALL_DIR%\fun.exe\" --daemon" DisplayName= "Fun Server" start= auto
sc description fun "Fun Server manages Docker containers and communicates with the Fun orchestrator"

echo Fun Server has been installed as a Windows service.
echo The service will start automatically on system boot.
echo You can also start it manually using the Windows Services panel.

exit /b 0 