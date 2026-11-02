// Package tradecalendar 提供新浪财经 A 股交易日历的客户端。
//
// 数据来源是站点的一个静态文本资源
// （`GET https://finance.sina.com.cn/realstock/company/klc_td_sh.txt`），响应形如：
//
//	var datelist="LC/AAApNDXC...";var KLC_TD_SH=datelist;
//
// 引号内不是明文日期，而是一段按位排列、带游程压缩的日期序列，需按固定算法解码后
// 才能得到交易日。本包负责拉取、解码，并提供按日期判断是否为交易日的查询能力。
//
// 约定：
//
//   - 日期统一用 UTC 零点的 time.Time 表示，只表示日历日期，不带时区含义；
//     IsTradeDay 按入参显示的年月日判断，不做时区换算；
//   - 解析失败返回明确错误，不会用空列表或零值日期兜底；
//   - 结果不含缓存，调用方可自行按数据源的 max-age 缓存。
//
// 该资源不是厂商文档化的 API，格式与编码可能随站点调整；解码算法的校验和不匹配
// 或数据提前结束时会直接报错，避免把损坏数据当成有效日历使用。
package cnatradecalendar

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultBaseURL 是新浪财经站点根地址。
	DefaultBaseURL = "https://finance.sina.com.cn"
	// DefaultCalendarPath 是交易日历资源路径，响应类型为 text/plain。
	DefaultCalendarPath = "/realstock/company/klc_td_sh.txt"
	// DefaultUserAgent 是默认请求 UA，非浏览器 UA 可能被站点拒绝。
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/120.0 Safari/537.36"
	// DefaultReferer 是默认来源页。
	DefaultReferer = "https://finance.sina.com.cn/"

	// defaultTimeout 是默认 HTTP 超时。
	defaultTimeout = 15 * time.Second
	// maxBodyBytes 限制响应体大小，正常响应约 500 字节。
	maxBodyBytes = 1 << 20
	// errorBodyLimit 是错误信息中保留的响应体长度。
	errorBodyLimit = 200
)

// Config 是客户端配置，零值可用。
type Config struct {
	// BaseURL 是站点根地址，默认 DefaultBaseURL。
	BaseURL string
	// CalendarPath 是交易日历资源路径，默认 DefaultCalendarPath。
	CalendarPath string
	// HTTPClient 是自定义客户端，默认带 15 秒超时的 http.Client。
	HTTPClient *http.Client
	// UserAgent 是请求 UA，默认 DefaultUserAgent。
	UserAgent string
	// Referer 是来源页，默认 DefaultReferer。
	Referer string
}

func (c Config) withDefaults() Config {
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	if c.CalendarPath == "" {
		c.CalendarPath = DefaultCalendarPath
	}
	if !strings.HasPrefix(c.CalendarPath, "/") {
		c.CalendarPath = "/" + c.CalendarPath
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: defaultTimeout}
	}
	if c.UserAgent == "" {
		c.UserAgent = DefaultUserAgent
	}
	if c.Referer == "" {
		c.Referer = DefaultReferer
	}
	return c
}

// Client 是交易日历客户端。
type Client struct {
	cfg Config
}

// New 按配置创建客户端，未设置的字段取默认值。
func New(cfg Config) *Client {
	return &Client{cfg: cfg.withDefaults()}
}

// Config 返回生效后的配置副本，便于排查实际使用的地址与请求头。
func (c *Client) Config() Config {
	return c.cfg
}

// APIError 表示本次请求没有拿到可用的响应。
type APIError struct {
	// HTTPStatus 是响应状态码，网络错误时为 0。
	HTTPStatus int
	// Message 是响应体摘要或网络错误的说明。
	Message string
	// Err 是被包装的底层错误，可能为 nil。
	Err error
}

func (e *APIError) Error() string {
	if e.HTTPStatus != 0 {
		return fmt.Sprintf("获取交易日历失败：HTTP %d %s", e.HTTPStatus, e.Message)
	}
	return "获取交易日历失败：" + e.Message
}

// Unwrap 返回被包装的底层错误。
func (e *APIError) Unwrap() error { return e.Err }

// GetTradeCalendar 拉取并解析 A 股交易日历。
//
// 返回的 Dates 按日期升序排列，日期为 UTC 零点。响应非 200、内容缺少 datelist
// 变量、编码非法或校验和不匹配时返回错误。
func (c *Client) GetTradeCalendar(ctx context.Context) (*TradeCalendar, error) {
	url := c.cfg.BaseURL + c.cfg.CalendarPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &APIError{Message: "构造请求失败", Err: err}
	}
	req.Header.Set("user-agent", c.cfg.UserAgent)
	req.Header.Set("referer", c.cfg.Referer)
	req.Header.Set("accept", "text/plain,*/*")

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, &APIError{Message: "请求失败", Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, &APIError{HTTPStatus: resp.StatusCode, Message: "读取响应失败", Err: err}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{
			HTTPStatus: resp.StatusCode,
			Message:    truncate(string(body), errorBodyLimit),
		}
	}
	return Parse(string(body))
}

// truncate 截断过长的文本，避免错误信息携带整个响应体。
func truncate(s string, limit int) string {
	s = strings.TrimSpace(s)
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}

// ErrNoData 表示响应中没有找到 datelist 变量。
var ErrNoData = errors.New("响应中未找到 datelist 变量")
