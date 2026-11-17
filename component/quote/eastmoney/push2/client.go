// Package push2 封装东方财富 push2 / push2his 行情接口。
//
// 当前实现两个接口，分别部署在同集群的两个域名下：
//
//   - GetQuoteList（`push2.eastmoney.com/api/qt/clist/get`）：全市场当日快照，
//     可按总市值或上市日期排序翻页；
//   - GetKLine（`push2his.eastmoney.com/api/qt/stock/kline/get`）：日线，
//     含真实的成交量（手）与成交额（元），也用来取交易日历。
//
// 协议要点：
//
//   - 站点有多个同集群镜像子域，主域偶发 502，本包按配置顺序轮换；
//   - 必须带 `referer: https://quote.eastmoney.com/`；
//   - 连续翻页与长区间 K 线请求会被限频，表现为 HTTP 200 但 `data=null`；
//   - 市值单位是**元**（腾讯是亿元），成交量单位是**手**。
package push2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/acnxo/MetaExt/component/quote/internal/fetch"
)

// 协议默认值。
var (
	// DefaultListHosts 行情列表的镜像域名，按顺序轮换。
	DefaultListHosts = []string{
		"https://82.push2.eastmoney.com",
		"https://1.push2.eastmoney.com",
		"https://7.push2.eastmoney.com",
		"https://push2.eastmoney.com",
	}
	// DefaultKLineHosts 历史 K 线的镜像域名，按顺序轮换。
	DefaultKLineHosts = []string{
		"https://push2his.eastmoney.com",
		"https://1.push2his.eastmoney.com",
		"https://82.push2his.eastmoney.com",
	}
)

// 固定常量。
const (
	// DefaultReferer 站点要求的来源页。
	DefaultReferer = "https://quote.eastmoney.com/"
	// DefaultPageSize 行情列表每页条数。
	DefaultPageSize = 100
	// UniverseMainBoard 沪深主板（深 t:6、沪 t:2），不含创业板、科创板、北交所。
	UniverseMainBoard = "m:0+t:6,m:1+t:2"
	// UniverseAllA 全部 A 股，含创业板 t:80 与科创板 t:23。
	UniverseAllA = "m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23"
	// SortByTotalMV 按总市值排序。
	SortByTotalMV = "f20"
	// SortByListDate 按上市日期排序。
	SortByListDate = "f26"
)

// Config 是客户端配置；零值代表使用协议默认值。
type Config struct {
	// ListHosts 行情列表镜像域名，默认 DefaultListHosts。
	ListHosts []string
	// KLineHosts 历史 K 线镜像域名，默认 DefaultKLineHosts。
	KLineHosts []string
	// Referer 来源页，默认 DefaultReferer。
	Referer string
	// Timeout 单次请求超时，默认 20 秒。
	Timeout time.Duration
	// Retries 每个镜像内的请求尝试次数，默认 4。
	Retries int
	// HTTPClient 可注入自定义客户端，主要用于测试与代理场景。
	HTTPClient *http.Client
	// Sleep 可注入等待函数，测试时用来跳过退避等待。
	Sleep func(time.Duration)
}

func (c Config) withDefaults() Config {
	if len(c.ListHosts) == 0 {
		c.ListHosts = DefaultListHosts
	}
	if len(c.KLineHosts) == 0 {
		c.KLineHosts = DefaultKLineHosts
	}
	if strings.TrimSpace(c.Referer) == "" {
		c.Referer = DefaultReferer
	}
	return c
}

// Client 是东财行情客户端。
//
// Client 并发安全：镜像轮换用原子计数，内部不保存其它可变状态。
type Client struct {
	cfg  Config
	req  *fetch.Fetcher
	next atomic.Uint64
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
		return fmt.Sprintf("东财 %s 请求失败：HTTP %d %s", e.Path, e.HTTPStatus, e.Message)
	case e.Message != "":
		return fmt.Sprintf("东财 %s 请求失败：%s", e.Path, e.Message)
	default:
		return "东财 " + e.Path + " 请求失败"
	}
}

// Unwrap 返回被包装的底层错误。
func (e *APIError) Unwrap() error { return e.Err }

// getRaw 依次轮换镜像域名请求，返回第一个可用响应。
//
// usable 为 nil 表示只要 HTTP 成功就算可用；否则由调用方判断响应内容是否可用
// （例如 `data` 为 null 时视为被限频，换下一个镜像再试）。
func (c *Client) getRaw(ctx context.Context, hosts []string, path string, query url.Values,
	usable func([]byte) bool) ([]byte, string, error) {
	var (
		lastErr    error
		lastStatus int
		lastBody   string
		lastURL    string
	)
	for attempt := 0; attempt < len(hosts); attempt++ {
		host := c.rotate(hosts)
		apiURL := host + path + "?" + query.Encode()
		raw, err := c.req.Get(ctx, apiURL, map[string]string{"referer": c.cfg.Referer})
		lastURL = apiURL
		if err != nil {
			var statusErr *fetch.StatusError
			if errors.As(err, &statusErr) {
				lastStatus, lastBody = statusErr.Status, statusErr.Body
			}
			lastErr = err
			continue
		}
		if usable != nil && !usable(raw) {
			lastErr = errors.New("响应不可用（可能被限频）")
			lastBody = truncateBody(raw)
			continue
		}
		return raw, apiURL, nil
	}
	return nil, lastURL, &APIError{
		Path:       path,
		URL:        lastURL,
		HTTPStatus: lastStatus,
		Message:    lastBody,
		Body:       lastBody,
		Err:        lastErr,
	}
}

// rotate 取下一个镜像域名（并发安全）。
func (c *Client) rotate(hosts []string) string {
	if len(hosts) == 1 {
		return hosts[0]
	}
	n := c.next.Add(1) - 1
	return hosts[int(n%uint64(len(hosts)))]
}

// wrapParseError 把解析错误包装成 *APIError。
func wrapParseError(path, apiURL string, raw []byte, err error) error {
	return &APIError{
		Path:    path,
		URL:     apiURL,
		Message: "响应解析失败: " + err.Error(),
		Body:    truncateBody(raw),
		Err:     err,
	}
}

// hasData 判断东财响应里的 `data` 是否为非 null 对象。
func hasData(raw []byte) bool {
	var probe struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return false
	}
	trimmed := strings.TrimSpace(string(probe.Data))
	return trimmed != "" && trimmed != "null"
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
