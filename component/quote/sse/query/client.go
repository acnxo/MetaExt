// Package query 封装上交所 sseQuery 的行情基础数据接口。
//
// 当前实现 A 股证券列表（含上市日期）：一次请求即可拿到沪市全部 A 股，
// 无需分页。站点只支持 HTTP，且要求带 referer 做来源校验。
//
// 协议要点：
//
//   - `sqlId` 固定为 `COMMON_SSE_CP_GPJCTPZ_GPLB_GP_L`，`STOCK_TYPE=1` 表示 A 股；
//   - `pageHelp.pageSize` 设 3000 即可覆盖全部沪市 A 股，多于一条也一次返回；
//   - `LIST_DATE` 为 `yyyyMMdd`，不是八位数字的行直接丢弃；
//   - 返回体可能被标成 text/html，但内容仍是 JSON。
package query

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/acnxo/MetaExt/component/quote/internal/fetch"
)

// 协议默认值。
const (
	// DefaultBaseURL 站点地址；上交所该接口只支持 HTTP。
	DefaultBaseURL = "http://query.sse.com.cn"
	// DefaultStockListPath 证券列表路径。
	DefaultStockListPath = "/sseQuery/commonQuery.do"
	// DefaultReferer 站点要求的来源页。
	DefaultReferer = "http://www.sse.com.cn/"
	// DefaultSQLID 股票列表查询标识。
	DefaultSQLID = "COMMON_SSE_CP_GPJCTPZ_GPLB_GP_L"
	// DefaultPageSize 每页条数；3000 足以覆盖全部沪市 A 股。
	DefaultPageSize = 3000
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

// Client 是上交所基础数据客户端。
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
		return fmt.Sprintf("上交所 %s 请求失败：HTTP %d %s", e.Path, e.HTTPStatus, e.Message)
	case e.Message != "":
		return fmt.Sprintf("上交所 %s 解析失败：%s", e.Path, e.Message)
	default:
		return "上交所 " + e.Path + " 请求失败"
	}
}

// Unwrap 返回被包装的底层错误。
func (e *APIError) Unwrap() error { return e.Err }
