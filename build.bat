@echo off
set VERSION=dev
set GOPROXY=https://goproxy.cn,https://goproxy.io,direct

echo ========================================
echo  Dolphin Build Script
echo ========================================

echo.
echo [1/3] Building Go binary...
go build -ldflags="-X 'dolphin/cmd.Version=%VERSION%'" -o dolphin.exe .
if errorlevel 1 (
    echo FAILED: Go build (exit code %errorlevel%)
    exit /b %errorlevel%
)
echo OK: dolphin.exe

echo.
echo [2/3] Building C# WebHost...
set WPHOST=deps\win\webhost\src\WebHost\WebHost.csproj
if exist %WPHOST% (
    dotnet build %WPHOST% -c Release --nologo -v q
    if errorlevel 1 (
        echo WARNING: WebHost build failed (exit code %errorlevel%)
    ) else (
        echo OK: WebHost
    )
) else (
    echo SKIP: %WPHOST% not found
)

echo.
echo [3/3] Done
echo   dolphin.exe  - main binary
