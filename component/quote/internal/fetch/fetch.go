// Package fetch 是行情接口共用的 HTTP 请求层。
//
// 只负责「发 GET 请求 + 退避重试 + 响应体上限 + 可选编码解码」，不做业务解析，
// 也不定义各业务包的错误类型：HTTP 非 2xx 时返回 *StatusError，调用方可按自己的
// APIError 包装并把状态码带进错误信息。
//
// 本包无状态，创建后可被多个 goroutine 并发使用。
package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// 协议默认值。
const (
	// DefaultTimeout 单次请求超时。
	DefaultTimeout = 20 * time.Second
	// DefaultRetries 请求尝试次数（含首次）。
	DefaultRetries = 4
	// DefaultMaxBytes 响应体上限，防止异常响应把内存打满。
	DefaultMaxBytes = 32 << 20
	// DefaultUserAgent 默认 UA；站点普遍不校验，带浏览器 UA 更稳妥。
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/126.0 Safari/537.36"

	// backoffBase 退避基数，第 n 次重试等待 n 倍。
	backoffBase = 800 * time.Millisecond
	// errorBodyLimit 错误信息中保留的响应体长度。
	errorBodyLimit = 300
)

// Encoding 是响应体的字符编码。
type Encoding string

// 支持的编码；默认 UTF-8。
const (
	UTF8 Encoding = "utf-8"
	// GBK 用于腾讯行情快照（qt.gtimg.cn），按 UTF-8 解码会得到乱码。
	GBK Encoding = "gbk"
)

// Config 是请求层配置；零值代表使用默认值。
type Config struct {
	// UserAgent 请求 UA，默认 DefaultUserAgent。
	UserAgent string
	// Timeout 单次请求超时，默认 DefaultTimeout。
	Timeout time.Duration
	// Retries 请求尝试次数，小于 1 时按 DefaultRetries 处理。
	Retries int
	// MaxBytes 响应体上限，小于等于 0 时按 DefaultMaxBytes 处理。
	MaxBytes int64
	// HTTPClient 可注入自定义客户端，主要用于测试与代理场景。
	HTTPClient *http.Client
	// Sleep 可注入等待函数，测试时用来跳过退避等待。
	Sleep func(time.Duration)
}

func (c Config) withDefaults() Config {
	if strings.TrimSpace(c.UserAgent) == "" {
		c.UserAgent = DefaultUserAgent
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	if c.Retries <= 0 {
		c.Retries = DefaultRetries
	}
	if c.MaxBytes <= 0 {
		c.MaxBytes = DefaultMaxBytes
	}
	if c.Sleep == nil {
		c.Sleep = time.Sleep
	}
	return c
}

// Fetcher 是请求层客户端。
type Fetcher struct {
	cfg  Config
	http *http.Client
}

// New 创建请求层客户端，零值配置按默认值补齐。
func New(cfg Config) *Fetcher {
	cfg = cfg.withDefaults()
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &Fetcher{cfg: cfg, http: client}
}

// Config 返回生效配置（已补齐默认值）。
func (f *Fetcher) Config() Config { return f.cfg }

// StatusError 表示服务端返回了非 2xx 状态码。
//
// 各业务包可用 errors.As 取出状态码，填进自己的 APIError。
type StatusError struct {
	// URL 完整请求地址。
	URL string
	// Status HTTP 状态码。
	Status int
	// Body 响应体，已截断到 300 字节以内。
	Body string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("HTTP %d %s", e.Status, e.Body)
}

// Get 发起 GET 请求并返回原始响应体。
//
// headers 会覆盖默认请求头（如 Referer）。网络错误与 5xx / 429 按退避重试，
// 其余 4xx 立即返回 *StatusError；context 取消时不重试。
func (f *Fetcher) Get(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	return f.do(ctx, url, headers)
}

// GetText 发起 GET 请求并按指定编码解码。
//
// enc 为空时按 UTF-8 处理；非法编码按 UTF-8 处理。
func (f *Fetcher) GetText(ctx context.Context, url string, headers map[string]string, enc Encoding) (string, error) {
	raw, err := f.do(ctx, url, headers)
	if err != nil {
		return "", err
	}
	return Decode(raw, enc)
}

// Decode 把原始字节按指定编码转成 UTF-8 字符串。
func Decode(raw []byte, enc Encoding) (string, error) {
	if enc == GBK {
		text, err := simplifiedchinese.GBK.NewDecoder().Bytes(raw)
		if err != nil {
			return "", fmt.Errorf("GBK 解码失败: %w", err)
		}
		return string(text), nil
	}
	return string(raw), nil
}

// GetJSON 发起 GET 请求并把响应体按 JSON 解析到 out。
func (f *Fetcher) GetJSON(ctx context.Context, url string, headers map[string]string, out any) error {
	raw, err := f.do(ctx, url, headers)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("响应解析失败: %w", err)
	}
	return nil
}

// do 是重试主体。
func (f *Fetcher) do(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	var last error
	for attempt := 1; attempt <= f.cfg.Retries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		raw, err := f.once(ctx, url, headers)
		if err == nil {
			return raw, nil
		}
		last = err

		// 4xx（429 除外）是请求本身的问题，重试没有意义。
		var statusErr *StatusError
		if errors.As(err, &statusErr) && statusErr.Status >= 400 && statusErr.Status < 500 &&
			statusErr.Status != http.StatusTooManyRequests {
			return nil, err
		}
		// context 已取消或超时，不再等待。
		if ctx.Err() != nil {
			return nil, err
		}
		if attempt < f.cfg.Retries {
			f.cfg.Sleep(time.Duration(attempt) * backoffBase)
		}
	}
	return nil, fmt.Errorf("请求失败（已尝试 %d 次）: %w", f.cfg.Retries, last)
}

// once 是单次请求。
func (f *Fetcher) once(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("user-agent", f.cfg.UserAgent)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := f.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, f.cfg.MaxBytes))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &StatusError{URL: url, Status: resp.StatusCode, Body: truncate(string(raw), errorBodyLimit)}
	}
	return raw, nil
}

// truncate 截断过长的文本，避免错误信息携带整个响应体。
func truncate(s string, limit int) string {
	s = strings.TrimSpace(s)
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}
