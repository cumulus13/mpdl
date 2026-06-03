@echo off
setlocal

set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0

echo [1/3] Downloading dependencies...
go mod tidy
if errorlevel 1 goto :err

echo [2/3] Building mpdl.exe ...
go build -o mpdl.exe .
if errorlevel 1 goto :err

echo [3/3] Done!
echo.
goto :end

:err
echo BUILD FAILED
exit /b 1

:end
