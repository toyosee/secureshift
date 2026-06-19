@echo off
echo ========================================
echo  SecureShift - Fresh Build Script
echo ========================================
echo.

echo 🧹 Step 1: Cleaning build cache...
go clean -cache
go clean -modcache
go clean -testcache
echo ✅ Done
echo.

echo 🧹 Step 2: Removing old binaries...
if exist secure-shift.exe del secure-shift.exe
if exist secureshift.exe del secureshift.exe
if exist release\secure-shift.exe del release\secure-shift.exe
echo ✅ Done
echo.

echo 🧹 Step 3: Removing test data...
if exist test rmdir /s /q test 2>nul
if exist temp rmdir /s /q temp 2>nul
if exist tmp rmdir /s /q tmp 2>nul
if exist test_project rmdir /s /q test_project 2>nul
if exist test-clone rmdir /s /q test-clone 2>nul
if exist test.js del test.js 2>nul
if exist test_project.zip del test_project.zip 2>nul
if exist secureshift.db del secureshift.db 2>nul
echo ✅ Done
echo.

echo 📦 Step 4: Tidying dependencies...
go mod tidy
echo ✅ Done
echo.

echo 🔨 Step 5: Building fresh binary...
go build -ldflags="-s -w" -o secure-shift.exe main.go
echo ✅ Done
echo.

echo Build complete!
echo ========================================
echo  File: secure-shift.exe
echo  Size: 
dir secure-shift.exe | find "secure-shift.exe"
echo ========================================
echo.
echo To run: secure-shift.exe
echo Open: http://localhost:8080
echo.
pause