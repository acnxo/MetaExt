// Package dataapi 封装巨潮资讯的数据接口。
//
// 当前实现深市证券清单：一次请求返回深市全部代码与简称，无分页，
// 是「代码清单」的首选来源。该接口**不返回上市日期**，需要时由调用方另行补齐。
//
// 协议要点：
//
//   - 站点只支持 HTTP，路径固定为 `/new/data/szse_stock.json`；
//   - 响应可能是 `{"stockList": [...]}`，也可能直接是数组，两种都要兼容；
//   - 返回包含创业板、B 股等，调用方按代码前缀自行过滤。
package dataapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/acnxo/MetaExt/component/quote/internal/fetch"
)

// 协议默认值。
const (
	// DefaultBaseURL 站点地址；巨潮该接口只支持 HTTP。
	DefaultBaseURL = "http://www.cninfo.com.cn"
	// DefaultStockListPath 深市证券清单路径。
	DefaultStockListPath = "/new/data/szse_stock.json"
	// DefaultReferer 站点要求的来源页。
	DefaultReferer = "http://www.cninfo.com.cn/"
)

// Config 是客户端配置；零值代表使用协议默认值。
type Config struct {
	// BaseURL 站点地址，默认 DefaultBaseURL。
	BaseURL string
	// StockListPath 证券清单路径，默认 DefaultStockListPath。
	StockListPath string
	// Referer 来源页，默认 DefaultReferer。
	Referer string
	// Timeout 单次请求超时，默认 20 秒。
	Timeout time.Duration
	// Retries 请求尝试次数，默认 4。
	Retries int
	// HTTPClient 可注入自定义客户端，主要用于测试与代理场景。
	HTTPClient *http.Client
	// Sleep 可注入等待函数，测试时用来跳过退避等待。
	Sleep func(time.Duration)
}

func (c Config) withDefaults() Config {
	if strings.TrimSpace(c.BaseURL) == "" {
		c.BaseURL = DefaultBaseURL
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	if strings.TrimSpace(c.StockListPath) == "" {
		c.StockListPath = DefaultStockListPath
	}
	if !strings.HasPrefix(c.StockListPath, "/") {
		c.StockListPath = "/" + c.StockListPath
	}
	if strings.TrimSpace(c.Referer) == "" {
		c.Referer = DefaultReferer
	}
	return c
}

// Client 是巨潮数据客户端。
//
// Client 并发安全：创建后仅读取配置，内部不保存可变状态。
type Client struct {
	cfg Config
	req *fetch.Fetcher
}

// New 创建客户端，零值配置按协议默认值补齐。
func New(cfg Config) *Client {
	cfg = cfg.withDefaults()
	return &Client{
		cfg: cfg,
		req: fetch.New(fetch.Config{
			Timeout:    cfg.Timeout,
			Retries:    cfg.Retries,
			HTTPClient: cfg.HTTPClient,
			Sleep:      cfg.Sleep,
		}),
	}
}

// Config 返回生效配置（已补齐默认值）。
func (c *Client) Config() Config { return c.cfg }

// APIError 表示接口没有返回可用数据。
type APIError struct {
	// Path 接口路径。
	Path string
	// URL 完整请求地址。
	URL string
	// HTTPStatus HTTP 状态码，请求未到达服务端时为 0。
	HTTPStatus int
	// Message 错误说明。
	Message string
	// Body 响应体，已截断。
	Body string
	// Err 被包装的底层错误，可能为 nil。
	Err error
}

func (e *APIError) Error() string {
	switch {
	case e.HTTPStatus != 0:
		return fmt.Sprintf("巨潮 %s 请求失败：HTTP %d %s", e.Path, e.HTTPStatus, e.Message)
	case e.Message != "":
		return fmt.Sprintf("巨潮 %s 请求失败：%s", e.Path, e.Message)
	default:
		return "巨潮 " + e.Path + " 请求失败"
	}
}

// Unwrap 返回被包装的底层错误。
func (e *APIError) Unwrap() error { return e.Err }
