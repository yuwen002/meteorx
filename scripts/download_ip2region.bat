@echo off
echo ========================================
echo Download ip2region.xdb database file
echo ========================================
echo.

cd ..
if not exist data mkdir data

echo Downloading ip2region.xdb...
echo.

REM Try different download sources
curl -L -o data/ip2region.xdb "https://github.com/lionsoul2014/ip2region/raw/master/data/ip2region.xdb"

if %errorlevel% neq 0 (
    echo First source failed, trying mirror...
    curl -L -o data/ip2region.xdb "https://cdn.jsdelivr.net/gh/lionsoul2014/ip2region/data/ip2region.xdb"
)

if %errorlevel% neq 0 (
    echo.
    echo All download sources failed!
    echo.
    echo Please download manually:
    echo 1. Visit: https://github.com/lionsoul2014/ip2region/releases
    echo 2. Download ip2region.xdb file
    echo 3. Place it in: data/ip2region.xdb
    echo.
    echo Or use HTTP API mode:
    echo Set METEORX_IP_LOCATION_PROVIDER=http-api in .env file
    echo.
    pause
    exit /b 1
)

echo.
echo SUCCESS: ip2region.xdb downloaded to data/ip2region.xdb
pause