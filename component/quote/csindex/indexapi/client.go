// Package indexapi 封装中证指数官网的指数表现接口。
//
// 当前实现指数日行情（`/csindex-home/perf/index-perf`）：按指数代码取指定区间的
// 日收盘，本项目用它取基准指数（中证 2000 `932000`、中证全指 `000985`）。
//
// 协议要点：
//
//   - 指数有发布日，早于发布日的区间没有数据（中证 2000 为 2023-08-11）；
//   - `tradeDate` 可能是 `yyyyMMdd` 或 `yyyy-MM-dd`，本包统一归一化为 `yyyy-MM-dd`；
//   - 官网偶有非交易日重复行，本包按日期去重保留最后一条；
//   - 只给收盘等 OHLC，不含成交量与成交额。
package indexapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/acnxo/MetaExt/component/quote/internal/fetch"
)

// 协议默认值。
const (
	// DefaultBaseURL 站点地址。
	DefaultBaseURL = "https://www.csindex.com.cn"
	// DefaultIndexPerfPath 指数表现接口路径。
	DefaultIndexPerfPath = "/csindex-home/perf/index-perf"
	// DefaultReferer 站点要求的来源页。
	DefaultReferer = "https://www.csindex.com.cn/"
)

// Config 是客户端配置；零值代表使用协议默认值。
type Config struct {
	// BaseURL 站点地址，默认 DefaultBaseURL。
	BaseURL string
	// IndexPerfPath 指数表现路径，默认 DefaultIndexPerfPath。
	IndexPerfPath string
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
	if strings.TrimSpace(c.IndexPerfPath) == "" {
		c.IndexPerfPath = DefaultIndexPerfPath
	}
	if !strings.HasPrefix(c.IndexPerfPath, "/") {
		c.IndexPerfPath = "/" + c.IndexPerfPath
	}
	if strings.TrimSpace(c.Referer) == "" {
		c.Referer = DefaultReferer
	}
	return c
}

// Client 是中证指数客户端。
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
		return fmt.Sprintf("中证指数 %s 请求失败：HTTP %d %s", e.Path, e.HTTPStatus, e.Message)
	case e.Message != "":
		return fmt.Sprintf("中证指数 %s 请求失败：%s", e.Path, e.Message)
	default:
		return "中证指数 " + e.Path + " 请求失败"
	}
}

// Unwrap 返回被包装的底层错误。
func (e *APIError) Unwrap() error { return e.Err }

// truncateBody 截断响应体，避免错误信息过大。
func truncateBody(raw []byte) string {
	const limit = 300
	text := strings.TrimSpace(string(raw))
	if len(text) <= limit {
		return text
	}
	return fmt.Sprintf("%s...", text[:limit])
}
