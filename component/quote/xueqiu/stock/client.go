// Package stock 封装雪球行情接口。
//
// 当前实现日线（`/v5/stock/chart/kline.json`）：一次请求返回全历史（最多 8000 根），
// `volume` 原生是**股**、`amount` 原生是**元**，是唯一「成交量 + 成交额」都无需换算的来源。
//
// 协议要点：
//
//   - 站点要求先访问 `https://xueqiu.com/` 拿到 cookie，再带 cookie 请求数据接口；
//   - cookie 失效时数据接口会返回空 `item`，本包会自动重建会话并重试一次；
//   - 响应用列式结构（`column` + `item`），必须按列名查下标，不能写死位置；
//   - 会话保存在注入的 `http.Client` 的 cookie jar 里，预热只做一次。
package stock

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"

	"github.com/acnxo/MetaExt/component/quote/internal/fetch"
)

// 协议默认值。
const (
	// DefaultBaseURL 数据接口站点。
	DefaultBaseURL = "https://stock.xueqiu.com"
	// DefaultWarmURL 预热地址，用于获取会话 cookie。
	DefaultWarmURL = "https://xueqiu.com/"
	// DefaultReferer 站点要求的来源页。
	DefaultReferer = "https://xueqiu.com/"
)

// Config 是客户端配置；零值代表使用协议默认值。
type Config struct {
	// BaseURL 数据接口站点，默认 DefaultBaseURL。
	BaseURL string
	// WarmURL 预热地址，默认 DefaultWarmURL。
	WarmURL string
	// Referer 来源页，默认 DefaultReferer。
	Referer string
	// Timeout 单次请求超时，默认 20 秒。
	Timeout time.Duration
	// Retries 请求尝试次数，默认 4。
	Retries int
	// HTTPClient 可注入自定义客户端；若其没有 cookie jar，本包会挂一个。
	HTTPClient *http.Client
	// Sleep 可注入等待函数，测试时用来跳过退避等待。
	Sleep func(time.Duration)
}

func (c Config) withDefaults() Config {
	if strings.TrimSpace(c.BaseURL) == "" {
		c.BaseURL = DefaultBaseURL
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	if strings.TrimSpace(c.WarmURL) == "" {
		c.WarmURL = DefaultWarmURL
	}
	if strings.TrimSpace(c.Referer) == "" {
		c.Referer = DefaultReferer
	}
	return c
}

// Client 是雪球行情客户端。
//
// Client 并发安全：会话状态（是否已预热）由互斥锁保护；预热只做一次。
type Client struct {
	cfg Config
	req *fetch.Fetcher

	mu     sync.Mutex
	warmed bool
}

// New 创建客户端，零值配置按协议默认值补齐。
func New(cfg Config) *Client {
	cfg = cfg.withDefaults()

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	// 复制一份客户端再挂 cookie jar，避免修改调用方传入的实例。
	cloned := *client
	if cloned.Jar == nil {
		jar, _ := cookiejar.New(nil)
		cloned.Jar = jar
	}

	return &Client{
		cfg: cfg,
		req: fetch.New(fetch.Config{
			Timeout:    cfg.Timeout,
			Retries:    cfg.Retries,
			HTTPClient: &cloned,
			Sleep:      cfg.Sleep,
		}),
	}
}

// Config 返回生效配置（已补齐默认值）。
func (c *Client) Config() Config { return c.cfg }

// warm 预热会话：访问首页拿 cookie，已完成时直接返回。
//
// 持锁期间完成网络请求，避免并发首请求重复预热；后续请求直接返回。
func (c *Client) warm(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.warmed {
		return nil
	}
	if _, err := c.req.Get(ctx, c.cfg.WarmURL, map[string]string{"referer": c.cfg.Referer}); err != nil {
		return err
	}
	c.warmed = true
	return nil
}

// invalidateSession 标记会话失效，下次请求会重新预热。
func (c *Client) invalidateSession() {
	c.mu.Lock()
	c.warmed = false
	c.mu.Unlock()
}

// APIError 表示接口没有返回可用数据。
type APIError struct {
	// Path 接口路径。
	Path string
	// URL 完整请求地址。
	URL string
	// HTTPStatus HTTP 状态码，请求未到达服务端时为 0。
	HTTPStatus int
	// Code 业务错误码（响应里的 error_code），HTTP 层失败时为 0。
	Code int
	// Message 错误说明。
	Message string
	// Body 响应体，已截断。
	Body string
	// Err 被包装的底层错误，可能为 nil。
	Err error
}

func (e *APIError) Error() string {
	switch {
	case e.Code != 0:
		return fmt.Sprintf("雪球 %s 失败：error_code=%d %s", e.Path, e.Code, e.Message)
	case e.HTTPStatus != 0:
		return fmt.Sprintf("雪球 %s 请求失败：HTTP %d %s", e.Path, e.HTTPStatus, e.Message)
	case e.Message != "":
		return fmt.Sprintf("雪球 %s 请求失败：%s", e.Path, e.Message)
	default:
		return "雪球 " + e.Path + " 请求失败"
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
