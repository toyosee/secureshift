@echo off
echo ========================================
echo  SecureShift - Release Build
echo ========================================
echo.

if not exist release mkdir release

echo 🧹 Clean build...
go clean -cache
go mod tidy

echo 🔨 Building Windows 64-bit...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w" -o release\secure-shift-windows-amd64.exe main.go

echo 🔨 Building Windows 32-bit...
set GOOS=windows
set GOARCH=386
go build -ldflags="-s -w" -o release\secure-shift-windows-386.exe main.go

echo 📦 Creating ZIP archive...
powershell -command "Compress-Archive -Path release\secure-shift-windows-amd64.exe -DestinationPath release\SecureShift-v1.0.0-windows-amd64.zip -Force"

echo.
echo ✅ Release build complete!
echo ========================================
echo  Files in release folder:
dir release
echo ========================================
echo.
pause