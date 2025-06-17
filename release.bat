@echo off
setlocal enabledelayedexpansion

:: Build script for creating multi-platform releases on Windows
echo 🦆 Building duckduckgo-chat-api
echo ======================================




:: Set variables
set APP_NAME=duckduckgo-chat-api
set VERSION=%1
if "%VERSION%"=="" set VERSION="v1.0.0"
set BUILD_DIR="releases"

echo Building %APP_NAME% %VERSION%

:: Clean build directory
echo Cleaning build directory...
if exist %BUILD_DIR% (
    rmdir /s /q %BUILD_DIR%
)
mkdir %BUILD_DIR%

:: Check Go
echo Checking Go installation...
where go >nul 2>nul
if %errorLevel% neq 0 (
    echo ❌ Go is not installed
    goto :error
)
for /f "tokens=2" %%g in ('go version') do set GO_VERSION=%%g
echo ✅ Go !GO_VERSION!

:: Install dependencies
echo 📦 Installing dependencies...
go mod tidy

:: Build binaries
echo 🔨 Building binaries...
set LDFLAGS="-s -w -X main.Version=%VERSION%"

set CGO_ENABLED=0

:: go env


:: Linux AMD64
echo   📦 Linux AMD64...
set GOOS=linux
set GOARCH=amd64
go build -ldflags=%LDFLAGS% -o %BUILD_DIR%\%APP_NAME%_%VERSION%_linux_amd64 .

:: Linux ARM64
echo   📦 Linux ARM64...
set GOOS=linux
set GOARCH=arm64
go build -ldflags=%LDFLAGS% -o %BUILD_DIR%\%APP_NAME%_%VERSION%_linux_arm64 .

:: Windows AMD64
echo   📦 Windows AMD64...
set GOOS=windows
set GOARCH=amd64
go build -ldflags=%LDFLAGS% -o %BUILD_DIR%\%APP_NAME%_%VERSION%_windows_amd64.exe .

:: Windows ARM64
echo   📦 Windows ARM64...
set GOOS=windows
set GOARCH=arm64
go build -ldflags=%LDFLAGS% -o %BUILD_DIR%\%APP_NAME%_%VERSION%_windows_arm64.exe .

:: macOS AMD64
echo   📦 macOS AMD64...
set GOOS=darwin
set GOARCH=amd64
go build -ldflags=%LDFLAGS% -o %BUILD_DIR%\%APP_NAME%_%VERSION%_darwin_amd64 .

:: macOS ARM64 (Apple Silicon)
echo   📦 macOS ARM64...
set GOOS=darwin
set GOARCH=arm64
go build -ldflags=%LDFLAGS% -o %BUILD_DIR%\%APP_NAME%_%VERSION%_darwin_arm64 .

echo.
echo ✅ Build completed successfully!
echo.
echo 📁 Binaries created:
dir %BUILD_DIR% /b /a-d

:: Create archives
echo.
echo 📦 Creating archives...
cd %BUILD_DIR%

:: Archives for Unix systems (tar.gz) - 需要GnuWin32工具
echo Creating tar.gz archives (requires GnuWin32 tar)...
for %%f in (*linux* *darwin*) do (
    if exist "%%f" (
        tar -czf "%%f.tar.gz" "%%f"
        echo   ✅ %%f.tar.gz
    )
)

:: Archives for Windows (zip)
echo Creating zip archives...
where zip >nul 2>nul
if %errorLevel% neq 0 (
    echo   ⚠️  zip command not found, skipping Windows archives
    echo   📦 Install zip: choco install zip
) else (
    for %%f in (*windows*.exe) do (
        if exist "%%f" (
            set "zipfile=%%~nf.zip"
            zip "!zipfile!" "%%f"
            echo   ✅ !zipfile!
        )
    )
)

cd ..

echo.
echo 📋 Release files summary:
dir %BUILD_DIR%\*.tar.gz %BUILD_DIR%\*.zip /b 2>nul || echo No archives created
echo.
echo 🏷️  To create a GitHub release, use:
echo   gh release create %VERSION% %BUILD_DIR%\*.tar.gz %BUILD_DIR%\*.zip --title "Release %VERSION%" --notes "Release %VERSION%"

goto :end

:error
echo An error occurred. Exiting...
:end
endlocal



