// Package webapi 封装深交所 ShowReport 数据接口。
//
// 当前实现 A 股证券列表（含上市日期）：站点每页只返回 20 条、总量约 146 页，
// 因此需要先取第 1 页拿到总页数，再并发翻完其余页。
//
// 协议要点：
//
//   - 站点只支持 HTTP，`CATALOGID=1110` 为上市公司列表，`TABKEY=tab1` 为 A 股页签；
//   - 顶层是数组（按页签分组），本项目只取第 0 个元素；
//   - 简称字段带 `<u>` / `<a>` 高亮标签，需清理后才能入库；
//   - `agssrq` 为 `yyyy-MM-dd`，与上交所的 `yyyyMMdd` 格式不同。
package webapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/acnxo/MetaExt/component/quote/internal/fetch"
)

// 协议默认值。
const (
	// DefaultBaseURL 站点地址；深交所该接口只支持 HTTP。
	DefaultBaseURL = "http://www.szse.cn"
	// DefaultStockListPath 证券列表路径。
	DefaultStockListPath = "/api/report/ShowReport/data"
	// DefaultReferer 站点要求的来源页。
	DefaultReferer = "http://www.szse.cn/"
	// DefaultCatalogID 上市公司列表的目录标识。
	DefaultCatalogID = "1110"
	// DefaultTabKey A 股页签。
	DefaultTabKey = "tab1"
	// DefaultWorkers 并发翻页数。
	DefaultWorkers = 8
)

// Config 是客户端配置；零值代表使用协议默认值。
type Config struct {
	// BaseURL 站点地址，默认 DefaultBaseURL。
	BaseURL string
	// StockListPath 证券列表路径，默认 DefaultStockListPath。
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

// Client 是深交所基础数据客户端。
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
		return fmt.Sprintf("深交所 %s 请求失败：HTTP %d %s", e.Path, e.HTTPStatus, e.Message)
	case e.Message != "":
		return fmt.Sprintf("深交所 %s 请求失败：%s", e.Path, e.Message)
	default:
		return "深交所 " + e.Path + " 请求失败"
	}
}

// Unwrap 返回被包装的底层错误。
func (e *APIError) Unwrap() error { return e.Err }
