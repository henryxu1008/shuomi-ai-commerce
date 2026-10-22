// Package client 提供TEMU平台API客户端核心功能
package client

import (
	"net/http"

	"task-processor/internal/core/logger"

	"github.com/imroc/req/v3"
	"github.com/sirupsen/logrus"
)

// APIClient TEMU API客户端 - 核心客户端，只负责基础的HTTP通信
type APIClient struct {
	config      *Config
	client      *req.Client
	storeID     int64
	cookies     []*http.Cookie
	proxyURL    string // 代理地址
	logger      *logrus.Entry
	httpManager *HTTPManager
	authManager *AuthManager
}

// NewAPIClient 创建TEMU API客户端
// NewAPIClient constructs only an explicit platform HTTP session. It does not
// read Store rows, fetch cookies, write MallID or mutate task pause state.
func NewAPIClient(storeID int64, config *Config, proxyURL string, cookies []*http.Cookie) *APIClient {
	if config == nil {
		config = DefaultConfig()
	}
	log := logger.GetGlobalLogger("TEMUAPIClient").WithField("storeID", storeID)
	c := &APIClient{config: config, storeID: storeID, proxyURL: proxyURL, logger: log}
	c.httpManager = NewHTTPManager(proxyURL, log, config)
	c.authManager = NewAuthManager(log)
	c.client = c.httpManager.CreateClient()
	c.SetCookies(cookies)
	return c
}

// SetCookies 设置Cookie
func (c *APIClient) SetCookies(cookies []*http.Cookie) {
	c.cookies = cookies
	// req包使用SetCommonCookies来设置全局Cookie
	c.client.SetCommonCookies(cookies...)
	c.logger.WithField("cookieNum", len(cookies)).Info("设置Cookie")
}

// ReloadCookies 重新加载Cookie
func (c *APIClient) ReloadCookies() error {
	return NewAuthExpiredError("explicit TEMU session required", nil)
}

// HasCookies 检查是否有Cookie
func (c *APIClient) HasCookies() bool {
	return len(c.cookies) > 0
}

// GetCookieCount 获取Cookie数量
func (c *APIClient) GetCookieCount() int {
	return len(c.cookies)
}

// GetCookieValue 获取指定名称的Cookie值
func (c *APIClient) GetCookieValue(name string) string {
	for _, cookie := range c.cookies {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	return ""
}

// GetMallID 从Cookie中获取MALL_ID
func (c *APIClient) GetMallID() string {
	return c.GetCookieValue("MALL_ID")
}

// SetCookieValue 设置指定名称的Cookie值
func (c *APIClient) SetCookieValue(name, value string) {
	// 查找并更新现有Cookie
	for _, cookie := range c.cookies {
		if cookie.Name == name {
			cookie.Value = value
			c.logger.Infof("更新Cookie %s 的值为: %s", name, value)
			// 更新req客户端的Cookie
			c.client.SetCommonCookies(c.cookies...)
			return
		}
	}

	// 如果Cookie不存在，创建新的Cookie
	newCookie := &http.Cookie{
		Name:   name,
		Value:  value,
		Domain: ".temu.com",
		Path:   "/",
	}
	c.cookies = append(c.cookies, newCookie)
	c.logger.Infof("创建新Cookie %s 的值为: %s", name, value)
	// 更新req客户端的Cookie
	c.client.SetCommonCookies(c.cookies...)
}

// SetMallID 设置Cookie中的MALL_ID
func (c *APIClient) SetMallID(mallID string) {
	c.SetCookieValue("MALL_ID", mallID)
}

// SendTEMURequest 发送TEMU API请求（带Cookie检查和重试逻辑）
func (c *APIClient) SendTEMURequest(request map[string]any, result any) error {
	return c.authManager.SendRequestWithAuth(c, request, result)
}

// SendHTTPRequest 发送HTTP请求的内部方法
func (c *APIClient) SendHTTPRequest(method, url string, headers map[string]string, body any, formFields map[string]string, fileFields map[string]any) (*req.Response, error) {
	return c.httpManager.SendRequest(c.client, method, url, headers, body, formFields, fileFields)
}

// GetStoreID 获取店铺ID
func (c *APIClient) GetStoreID() int64 {
	return c.storeID
}

// GetBaseURL 获取基础URL
func (c *APIClient) GetBaseURL() string {
	return c.config.BaseURL
}

// GetConfig 获取配置
func (c *APIClient) GetConfig() any {
	return c.config
}

// GetLogger 获取日志记录器
func (c *APIClient) GetLogger() *logrus.Entry {
	return c.logger
}

// GetCookieManager 获取Cookie管理器
func (c *APIClient) GetCookieManager() any {
	return nil
}
