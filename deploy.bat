@echo off
setlocal
cd /d "%~dp0deploy"
go run . %*
exit /b %errorlevel%
