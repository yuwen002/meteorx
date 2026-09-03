package iplocation

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
)

// LocalIPLocator 基于本地 ip2region.xdb 文件的 IP 地理位置解析器
type LocalIPLocator struct {
	dbPath string
	cache  sync.Map
	mu     sync.RWMutex
	vIndex []byte
}

// NewLocalLocator 创建本地 IP 解析器
// dbPath: ip2region.xdb 数据库文件路径
func NewLocalLocator(dbPath string) (*LocalIPLocator, error) {
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("ip2region database file not found: %s", dbPath)
	}

	locator := &LocalIPLocator{
		dbPath: dbPath,
	}

	// 加载向量索引
	if err := locator.loadVectorIndex(); err != nil {
		return nil, fmt.Errorf("failed to load vector index: %w", err)
	}

	return locator, nil
}

// loadVectorIndex 加载向量索引到内存
func (l *LocalIPLocator) loadVectorIndex() error {
	file, err := os.Open(l.dbPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 读取索引长度（头部 256 字节后的 8 字节）
	indexLength := make([]byte, 8)
	if _, err := file.ReadAt(indexLength, 256); err != nil {
		return err
	}

	// 读取向量索引
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.vIndex = make([]byte, indexLength[0]|indexLength[1]<<8|indexLength[2]<<16|indexLength[3]<<24)
	if _, err := file.ReadAt(l.vIndex, 264); err != nil {
		return err
	}

	return nil
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

	// 查询数据库
	location, err := l.queryIP(ip)
	if err != nil {
		return &IPLocation{
			Country:  "未知",
			Region:   "-",
			City:     "未知",
			ISP:      "-",
			FullText: "未知",
		}, nil
	}

	// 缓存结果
	l.cache.Store(ip, location)
	return location, nil
}

// queryIP 从 ip2region 数据库查询 IP
func (l *LocalIPLocator) queryIP(ip string) (*IPLocation, error) {
	// 简化的实现：使用预定义的 IP 段映射
	// 实际项目中应该使用完整的 ip2region.xdb 解析
	
	// 这里提供一个基础实现，可以根据需要扩展
	location := l.parseIP(ip)
	
	return location, nil
}

// parseIP 解析 IP 地址（简化版本）
func (l *LocalIPLocator) parseIP(ip string) *IPLocation {
	// 这是一个简化实现
	// 完整的实现需要读取 ip2region.xdb 文件并进行二进制搜索
	
	// 返回默认值
	return &IPLocation{
		Country:  "中国",
		Region:   "-",
		City:     "-",
		ISP:      "-",
		FullText: "中国",
	}
}

// Close 关闭定位器（清理缓存）
func (l *LocalIPLocator) Close() {
	l.cache = sync.Map{}
}

// IPLocationFromXdb 从 ip2region 格式字符串解析位置信息
func IPLocationFromXdb(text string) *IPLocation {
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
	
	if len(parts) > 0 {
		loc.Country = parts[0]
	}
	if len(parts) > 1 && parts[1] != "0" {
		loc.Region = parts[1]
	}
	if len(parts) > 2 && parts[2] != "0" {
		loc.City = parts[2]
	}
	if len(parts) > 3 && parts[3] != "0" {
		loc.ISP = parts[3]
	}

	return loc
}