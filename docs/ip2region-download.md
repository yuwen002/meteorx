# IP2Region 数据库下载指南

## 概述

ip2region.xdb 文件用于离线 IP 地理位置解析。该文件约 10MB，包含 IP 地址到地理位置的映射数据。

## 下载方式

### 方式一：使用下载脚本（推荐）

**Windows：**
```bash
cd scripts
.\download_ip2region.bat
```

**Linux/Mac：**
```bash
cd scripts
chmod +x download_ip2region.sh
./download_ip2region.sh
```

### 方式二：手动下载

1. 访问官方发布页面：
   - https://github.com/lionsoul2014/ip2region/releases
   - 或：https://github.com/lionsoul2014/ip2region/tree/master/data

2. 下载 `ip2region.xdb` 文件

3. 将文件放置到项目的 `data/` 目录：
   ```
   meteorx/
   └── data/
       └── ip2region.xdb
   ```

### 方式三：使用 curl 命令行

```bash
# 创建 data 目录
mkdir -p data

# 从 GitHub 下载
curl -L -o data/ip2region.xdb \
  "https://github.com/lionsoul2014/ip2region/raw/master/data/ip2region.xdb"

# 或使用 CDN 镜像
curl -L -o data/ip2region.xdb \
  "https://cdn.jsdelivr.net/gh/lionsoul2014/ip2region/data/ip2region.xdb"
```

### 方式四：使用 wget

```bash
mkdir -p data
wget -O data/ip2region.xdb \
  "https://github.com/lionsoul2014/ip2region/raw/master/data/ip2region.xdb"
```

## 配置说明

下载完成后，在 `.env` 文件中配置：

### 使用离线模式（ip2region）
```env
METEORX_IP_LOCATION_PROVIDER=ip2region
METEORX_IP_LOCATION_DB_PATH=./data/ip2region.xdb
```

### 使用在线模式（HTTP API）
```env
METEORX_IP_LOCATION_PROVIDER=http-api
METEORX_IP_LOCATION_TIMEOUT=3
```

## 验证文件

验证下载的文件：

```bash
# 检查文件是否存在
ls -lh data/ip2region.xdb

# 预期大小：约 10MB
# 如果文件太小（< 5MB），可能下载不完整
```

## 常见问题

### 下载失败

如果所有下载源都失败：
1. 尝试使用 VPN 或代理
2. 更换网络环境下载
3. 使用手动下载方式
4. 或改用 HTTP API 模式

### 文件大小检查

ip2region.xdb 文件应该约 10MB。如果小于 5MB，说明下载不完整。

### 权限问题

确保应用程序有文件读取权限：
```bash
chmod 644 data/ip2region.xdb
```·

## 注意事项

- ip2region.xdb 文件不会提交到 Git（已在 .gitignore 中忽略）
- 每个开发者需要单独下载该文件
- 可以定期更新文件以获取最新的 IP 数据
- 离线模式速度更快，不需要网络访问
- HTTP API 模式需要网络，但不需要数据库文件