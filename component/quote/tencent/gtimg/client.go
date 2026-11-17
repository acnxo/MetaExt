// Package gtimg 封装腾讯 gtimg 的行情接口。
//
// 当前实现两个接口，分别位于 `gtimg.cn` 的两个子域：
//
//   - GetQuoteSnapshot（`qt.gtimg.cn/q=`）：批量快照，一次可带数百只代码，
//     返回最新价、今开、成交量、成交额、总市值与流通市值；
//   - GetKLine（`web.ifzq.gtimg.cn/appstock/app/fqkline/get`）：日线，支持前复权。
//
// 协议要点：
//
//   - 快照响应是 **GBK 文本**，不是 JSON，需按 GBK 解码后按 `~` 拆分；
//   - 单批最多约 900 只（URL 约 8.1 KB），超过会触发 HTTP 414，本包会自动二分降级；
//   - 市值单位是**亿元**（东财是元），成交量单位是**手**；
//   - 必须带 `referer: https://finance.qq.com/`。
package gtimg

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/acnxo/MetaExt/component/quote/internal/fetch"
)

// 协议默认值。
const (
	// DefaultBaseURL 快照接口站点。
	DefaultBaseURL = "https://qt.gtimg.cn"
	// DefaultKLineBaseURL 日线接口站点。
	DefaultKLineBaseURL = "https://web.ifzq.gtimg.cn"
	// DefaultReferer 站点要求的来源页。
	DefaultReferer = "https://finance.qq.com/"
	// DefaultBatch 单批代码数；实测 900 只（URL 约 8.1 KB）可用，950 只触发 414。
	DefaultBatch = 900
	// DefaultWorkers 快照分批的并发数。
	DefaultWorkers = 4
	// minSplitCodes 二分降级的终止阈值：单批不超过该值时不再拆分。
	minSplitCodes = 100
)

// Config 是客户端配置；零值代表使用协议默认值。
type Config struct {
	// BaseURL 快照接口站点，默认 DefaultBaseURL。
	BaseURL string
	// KLineBaseURL 日线接口站点，默认 DefaultKLineBaseURL。
	KLineBaseURL string
	// Referer 来源页，默认 DefaultReferer。
	Referer string
	// Batch 单批代码数，为空或非正时取 DefaultBatch。
	Batch int
	// Workers 分批并发数，为空或非正时取 DefaultWorkers。
	Workers int
	// MinSplitCodes 二分降级的终止阈值：单批不超过该值仍失败时不再拆分。
	// 为空或非正时取 minSplitCodes（100）。
	MinSplitCodes int
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
	if strings.TrimSpace(c.KLineBaseURL) == "" {
		c.KLineBaseURL = DefaultKLineBaseURL
	}
	c.KLineBaseURL = strings.TrimRight(c.KLineBaseURL, "/")
	if strings.TrimSpace(c.Referer) == "" {
		c.Referer = DefaultReferer
	}
	if c.Batch <= 0 {
		c.Batch = DefaultBatch
	}
	if c.Workers <= 0 {
		c.Workers = DefaultWorkers
	}
	if c.MinSplitCodes <= 0 {
		c.MinSplitCodes = minSplitCodes
	}
	return c
}

// Client 是腾讯行情客户端。
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
	// URL 最后一次尝试的完整请求地址。
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
		return fmt.Sprintf("腾讯 %s 请求失败：HTTP %d %s", e.Path, e.HTTPStatus, e.Message)
	case e.Message != "":
		return fmt.Sprintf("腾讯 %s 请求失败：%s", e.Path, e.Message)
	default:
		return "腾讯 " + e.Path + " 请求失败"
	}
}

// Unwrap 返回被包装的底层错误。
func (e *APIError) Unwrap() error { return e.Err }

// symbolOf 把六位代码转成腾讯的带前缀代码。
//
// 前缀规则：6 开头为沪市、4/8 开头为北交所、其余为深市。
// 920 段（北交所新代码）与 B 股无法按首位区分，需要调用方显式传 Symbol。
func symbolOf(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	switch code[0] {
	case '6':
		return "sh" + code
	case '4', '8':
		return "bj" + code
	default:
		return "sz" + code
	}
}

// truncateBody 截断响应体，避免错误信息过大。
func truncateBody(raw []byte) string {
	const limit = 300
	text := strings.TrimSpace(string(raw))
	if len(text) <= limit {
		return text
	}
	return fmt.Sprintf("%s...", text[:limit])
}
