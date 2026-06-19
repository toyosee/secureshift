@echo off
echo Cleaning SecureShift build environment...
echo.

echo 1. Cleaning Go cache...
go clean -cache
go clean -modcache
go clean -testcache

echo 2. Removing old binary...
if exist secure-shift.exe del secure-shift.exe
if exist secureshift.exe del secureshift.exe

echo 3. Removing test directories...
if exist test rmdir /s /q test
if exist temp rmdir /s /q temp
if exist tmp rmdir /s /q tmp
if exist test_project rmdir /s /q test_project
if exist test-clone rmdir /s /q test-clone

echo 4. Removing test files...
if exist test.js del test.js
if exist test.sh del test.sh
if exist test_project.zip del test_project.zip
if exist test_upload.sh del test_upload.sh

echo 5. Removing development database...
if exist secureshift.db del secureshift.db

echo.
echo ✅ Clean complete!
pause