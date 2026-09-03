package iplocation

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

// LocalIPLocator 基于本地 ip2region.xdb 文件的 IP 地理位置解析器
type LocalIPLocator struct {
	searcher *xdb.Searcher
	cache    sync.Map
}

// NewLocalLocator 创建本地 IP 解析器
// dbPath: ip2region.xdb 数据库文件路径
func NewLocalLocator(dbPath string) (*LocalIPLocator, error) {
	// 从数据文件加载头部信息，获取版本
	header, err := xdb.LoadHeaderFromFile(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load ip2region header: %w", err)
	}

	// 从头部获取版本信息
	version, err := xdb.VersionFromHeader(header)
	if err != nil {
		return nil, fmt.Errorf("failed to get version from header: %w", err)
	}

	// 使用全内存方式加载 ip2region.xdb
	cBuff, err := xdb.LoadContentFromFile(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load ip2region.xdb: %w", err)
	}

	// 创建 searcher
	searcher, err := xdb.NewWithBuffer(version, cBuff)
	if err != nil {
		return nil, fmt.Errorf("failed to create searcher: %w", err)
	}

	return &LocalIPLocator{
		searcher: searcher,
	}, nil
}

// Locate 查询 IP 地理位置
func (l *LocalIPLocator) Locate(ctx context.Context, ip string) (*IPLocation, error) {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return &IPLocation{
			Country:  "本地",
			Region:   "-",
			City:     "本地",
			ISP:      "-",
			FullText: "本地",
		}, nil
	}

	// 检查缓存
	if cached, ok := l.cache.Load(ip); ok {
		return cached.(*IPLocation), nil
	}

	// 查询数据库（尝试 Search 方法）
	regionStr, err := l.searcher.Search(ip)
	if err != nil {
		// 查询失败，返回未知
		location := &IPLocation{
			Country:  "未知",
			Region:   "-",
			City:     "未知",
			ISP:      "-",
			FullText: "未知",
		}
		l.cache.Store(ip, location)
		return location, nil
	}

	// 解析结果
	location := parseIPLocation(regionStr)

	// 缓存结果
	l.cache.Store(ip, location)
	return location, nil
}

// Close 关闭定位器
func (l *LocalIPLocator) Close() {
	l.searcher.Close()
	l.cache = sync.Map{}
}

// parseIPLocation 解析 ip2region 返回的字符串
// 格式：国家|区域|省份|城市|ISP
// 例如：中国|0|北京|北京市|联通
func parseIPLocation(text string) *IPLocation {
	if text == "" {
		return &IPLocation{
			Country:  "未知",
			Region:   "-",
			City:     "未知",
			ISP:      "-",
			FullText: "未知",
		}
	}

	parts := strings.Split(text, "|")

	loc := &IPLocation{
		FullText: text,
	}

	if len(parts) > 0 && parts[0] != "0" {
		loc.Country = parts[0]
	} else {
		loc.Country = "未知"
	}

	if len(parts) > 1 && parts[1] != "0" {
		loc.Region = parts[1]
	} else {
		loc.Region = "-"
	}

	if len(parts) > 2 && parts[2] != "0" {
		loc.City = parts[2]
	} else {
		loc.City = "未知"
	}

	if len(parts) > 3 && parts[3] != "0" {
		loc.ISP = parts[3]
	} else {
		loc.ISP = "-"
	}

	return loc
}