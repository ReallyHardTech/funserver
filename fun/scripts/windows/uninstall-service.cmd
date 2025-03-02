@echo off
REM Windows Service Removal Script for Fun Server

echo Removing Fun Server Windows service...

REM Stop the service if it's running
sc stop fun

REM Delete the service
sc delete fun

echo Fun Server service has been removed.

exit /b 0 