package iplocation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type IPLocation struct {
	Country  string `json:"country"`
	Region   string `json:"region"`
	City     string `json:"city"`
	ISP      string `json:"isp"`
	FullText string `json:"full_text"`
}

type IPLocator interface {
	Locate(ctx context.Context, ip string) (*IPLocation, error)
}

type HTTPAPILocator struct {
	cache    sync.Map
	client   *http.Client
	timeout  time.Duration
	provider string
}

func NewHTTPLocator(provider string, timeout time.Duration) *HTTPAPILocator {
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	if provider == "" {
		provider = "ip-api"
	}
	return &HTTPAPILocator{
		client: &http.Client{Timeout: timeout},
		provider: provider,
		timeout: timeout,
	}
}

func (l *HTTPAPILocator) Locate(ctx context.Context, ip string) (*IPLocation, error) {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return &IPLocation{
			Country:  "本地",
			Region:   "-",
			City:     "本地",
			ISP:      "-",
			FullText: "本地",
		}, nil
	}

	if cached, ok := l.cache.Load(ip); ok {
		return cached.(*IPLocation), nil
	}

	location, err := l.queryAPI(ctx, ip)
	if err != nil {
		return &IPLocation{
			Country:  "未知",
			Region:   "-",
			City:     "未知",
			ISP:      "-",
			FullText: "未知",
		}, nil
	}

	l.cache.Store(ip, location)
	return location, nil
}

func (l *HTTPAPILocator) queryAPI(ctx context.Context, ip string) (*IPLocation, error) {
	switch l.provider {
	case "ip-api":
		return l.queryIPAPI(ctx, ip)
	case "ipinfo":
		return l.queryIPInfo(ctx, ip)
	default:
		return l.queryIPAPI(ctx, ip)
	}
}

func (l *HTTPAPILocator) queryIPAPI(ctx context.Context, ip string) (*IPLocation, error) {
	url := fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN", ip)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ip-api returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Status      string  `json:"status"`
		Country     string  `json:"country"`
		RegionName  string  `json:"regionName"`
		City        string  `json:"city"`
		ISP         string  `json:"isp"`
		Message     string  `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("ip-api error: %s", result.Message)
	}

	parts := []string{}
	if result.Country != "" {
		parts = append(parts, result.Country)
	}
	if result.RegionName != "" {
		parts = append(parts, result.RegionName)
	}
	if result.City != "" {
		parts = append(parts, result.City)
	}
	if result.ISP != "" {
		parts = append(parts, result.ISP)
	}

	return &IPLocation{
		Country:  result.Country,
		Region:   result.RegionName,
		City:     result.City,
		ISP:      result.ISP,
		FullText: strings.Join(parts, " / "),
	}, nil
}

func (l *HTTPAPILocator) queryIPInfo(ctx context.Context, ip string) (*IPLocation, error) {
	url := fmt.Sprintf("https://ipinfo.io/%s/json", ip)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ipinfo returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Country string `json:"country"`
		Region  string `json:"region"`
		City    string `json:"city"`
		Org     string `json:"org"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	parts := []string{}
	if result.Country != "" {
		parts = append(parts, result.Country)
	}
	if result.Region != "" {
		parts = append(parts, result.Region)
	}
	if result.City != "" {
		parts = append(parts, result.City)
	}
	if result.Org != "" {
		parts = append(parts, result.Org)
	}

	return &IPLocation{
		Country:  result.Country,
		Region:   result.Region,
		City:     result.City,
		ISP:      result.Org,
		FullText: strings.Join(parts, " / "),
	}, nil
}

type MockLocator struct {
	Locations map[string]*IPLocation
}

func (m *MockLocator) Locate(ctx context.Context, ip string) (*IPLocation, error) {
	if loc, ok := m.Locations[ip]; ok {
		return loc, nil
	}
	return &IPLocation{
		Country:  "中国",
		Region:   "北京",
		City:     "北京",
		ISP:      "中国电信",
		FullText: "中国 / 北京 / 北京 / 中国电信",
	}, nil
}