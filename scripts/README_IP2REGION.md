# IP2Region Database Download Guide

## Overview

The ip2region.xdb file is required for offline IP geolocation parsing. This file is approximately 10MB and contains IP address to location mapping data.

## Download Methods

### Method 1: Using Download Script (Recommended)

**Windows:**
```bash
cd scripts
.\download_ip2region.bat
```

**Linux/Mac:**
```bash
cd scripts
chmod +x download_ip2region.sh
./download_ip2region.sh
```

### Method 2: Manual Download

1. Visit the official release page:
   - https://github.com/lionsoul2014/ip2region/releases
   - Or: https://github.com/lionsoul2014/ip2region/tree/master/data

2. Download the `ip2region.xdb` file

3. Place it in the project's `data/` directory:
   ```
   meteorx/
   └── data/
       └── ip2region.xdb
   ```

### Method 3: Using curl (Command Line)

```bash
# Create data directory
mkdir -p data

# Download from GitHub
curl -L -o data/ip2region.xdb \
  "https://github.com/lionsoul2014/ip2region/raw/master/data/ip2region.xdb"

# Or use CDN mirror
curl -L -o data/ip2region.xdb \
  "https://cdn.jsdelivr.net/gh/lionsoul2014/ip2region/data/ip2region.xdb"
```

### Method 4: Using wget

```bash
mkdir -p data
wget -O data/ip2region.xdb \
  "https://github.com/lionsoul2014/ip2region/raw/master/data/ip2region.xdb"
```

## Configuration

After downloading, configure your `.env` file:

### Use Offline Mode (ip2region)
```env
METEORX_IP_LOCATION_PROVIDER=ip2region
METEORX_IP_LOCATION_DB_PATH=./data/ip2region.xdb
```

### Use Online Mode (HTTP API)
```env
METEORX_IP_LOCATION_PROVIDER=http-api
METEORX_IP_LOCATION_TIMEOUT=3
```

## Verification

Verify the downloaded file:

```bash
# Check file exists
ls -lh data/ip2region.xdb

# Expected size: ~10MB
# If file is too small (< 5MB), download may be incomplete
```

## Troubleshooting

### Download Failed

If all download sources fail:
1. Try using a VPN or proxy
2. Download from a different network
3. Use the manual download method
4. Or use HTTP API mode instead

### File Size Check

The ip2region.xdb file should be approximately 10MB. If it's smaller than 5MB, the download is incomplete.

### Permission Issues

Ensure the application has read access to the file:
```bash
chmod 644 data/ip2region.xdb
```

## Notes

- The ip2region.xdb file is NOT committed to Git (it's in .gitignore)
- Each developer needs to download it separately
- The file can be updated periodically for latest IP data
- Offline mode is faster and doesn't require internet access
- HTTP API mode requires internet but doesn't need the database file