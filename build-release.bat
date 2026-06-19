@echo off
echo ========================================
echo  SecureShift - Build All Platforms
echo ========================================
echo.

if not exist release mkdir release

echo 🧹 Cleaning build cache...
go clean -cache
go mod tidy
echo.

echo 🔨 Building Windows 64-bit...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w -H=windowsgui" -o release/secure-shift-windows-amd64.exe main.go
echo ✅ Windows 64-bit complete
echo.

echo 🔨 Building Windows 32-bit...
set GOOS=windows
set GOARCH=386
go build -ldflags="-s -w -H=windowsgui" -o release/secure-shift-windows-386.exe main.go
echo ✅ Windows 32-bit complete
echo.

echo 🔨 Building Linux 64-bit...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o release/secure-shift-linux-amd64 main.go
echo ✅ Linux 64-bit complete
echo.

echo 🔨 Building Linux 32-bit...
set GOOS=linux
set GOARCH=386
go build -ldflags="-s -w" -o release/secure-shift-linux-386 main.go
echo ✅ Linux 32-bit complete
echo.

echo 🔨 Building macOS Intel...
set GOOS=darwin
set GOARCH=amd64
go build -ldflags="-s -w" -o release/secure-shift-darwin-amd64 main.go
echo ✅ macOS Intel complete
echo.

echo 🔨 Building macOS Apple Silicon...
set GOOS=darwin
set GOARCH=arm64
go build -ldflags="-s -w" -o release/secure-shift-darwin-arm64 main.go
echo ✅ macOS Apple Silicon complete
echo.

echo 📦 Creating ZIP archives...
powershell -command "Compress-Archive -Path release/secure-shift-windows-amd64.exe -DestinationPath release/SecureShift-v1.0.0-windows-amd64.zip -Force"
powershell -command "Compress-Archive -Path release/secure-shift-windows-386.exe -DestinationPath release/SecureShift-v1.0.0-windows-386.zip -Force"
powershell -command "Compress-Archive -Path release/secure-shift-linux-amd64 -DestinationPath release/SecureShift-v1.0.0-linux-amd64.zip -Force"
powershell -command "Compress-Archive -Path release/secure-shift-linux-386 -DestinationPath release/SecureShift-v1.0.0-linux-386.zip -Force"
powershell -command "Compress-Archive -Path release/secure-shift-darwin-amd64 -DestinationPath release/SecureShift-v1.0.0-darwin-amd64.zip -Force"
powershell -command "Compress-Archive -Path release/secure-shift-darwin-arm64 -DestinationPath release/SecureShift-v1.0.0-darwin-arm64.zip -Force"
echo ✅ ZIP archives created
echo.

echo ========================================
echo  ✅ Release build complete!
echo ========================================
echo.
echo 📁 Files in release folder:
dir release
echo.
pause