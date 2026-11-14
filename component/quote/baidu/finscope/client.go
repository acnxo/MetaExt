// Package finscope 封装百度股市通（finance.pae.baidu.com）行情接口。
//
// 当前实现 constituents 接口（指数 / 概念板块成分股列表），其余 getquotation 系列接口
// 可按同样的方式在 Client 上扩展：校验参数、构造有序查询串、发请求、解析响应四步。
//
// 该站点与 moomoo 有两点关键差异，使用前必须知道：
//
//   - **无签名、无 token**：服务端靠 Referer、浏览器 UA 与访问频率做风控，
//     参数顺序不影响结果；
//   - **外层 `ResultCode=0` 不代表业务成功**：风控时 `Result` 会被替换成
//     `{"code":403,"isCaptchaEnabled":true,"msg":"hit risk"}`，业务成败一律看 `Result`
//     这一层。本包把这种情况转成 *APIError，并用 IsRiskControl 区分，不会当作空数据返回。
//
// 约定：
//
//   - 行情类字段同时提供「带单位的展示字符串」与「rawData 原始数值」两份，取数值请用
//     RawData，它没有做单位换算（`15.42亿` 对应 1542319057）；
//   - 解析失败、类型不符、风控一律返回明确错误，不用空列表兜底。
package finscope

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// 协议默认值。
const (
	// DefaultBaseURL 接口站点。
	DefaultBaseURL = "https://finance.pae.baidu.com"
	// DefaultConstituentsPath 成分股列表接口路径。
	DefaultConstituentsPath = "/sapi/v1/constituents"
	// DefaultQuotePath 行情接口路径，多个 group 共用。
	DefaultQuotePath = "/vapi/v1/getquotation"
	// DefaultSuggestPath 搜索接口路径，用于按名称查代码。
	DefaultSuggestPath = "/vapi/v1/sug"
	// DefaultFuturesFiveDayPath 期货五日分时接口路径，与股票侧不同。
	DefaultFuturesFiveDayPath = "/selfselect/getstockquotation"
	// DefaultReferer 默认来源页；百度按来源页判断请求是否来自浏览器。
	DefaultReferer = "https://gushitong.baidu.com/"
	// DefaultUserAgent 浏览器 UA；建议与真实浏览器一致，空 UA 或工具 UA 更易被风控。
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	// DefaultClientType 客户端类型参数 finClientType，固定 pc。
	DefaultClientType = "pc"
	// DefaultStyle 返回样式，固定 tablelist（表格结构）。
	DefaultStyle = "tablelist"
	// DefaultSrcID 来源标识，行情接口固定 5353。
	DefaultSrcID = "5353"
	// DefaultPointType 点位类型，固定 string，决定行情以字符串表格返回。
	DefaultPointType = "string"
	// GroupKLine getquotation 的 K 线分组。
	GroupKLine = "quotation_kline_ab"
	// GroupFiveDay getquotation 的股票五日分时分组。
	GroupFiveDay = "quotation_fiveday_ab"
	// DefaultKLineCount K 线默认条数，241 可覆盖一个交易日的分钟线。
	DefaultKLineCount = 241
	// DefaultPageSize 单页条数默认值。实测 rn=5000 可一次返回该来源全部 2236 条，
	// 没有 500 硬上限。
	DefaultPageSize = 500
	// DefaultTimeout 单次请求超时。
	DefaultTimeout = 20 * time.Second
	// maxBodyBytes 限制响应体大小，正常响应在数百 KB 以内。
	maxBodyBytes = 32 << 20
	// errorBodyLimit 是错误信息中保留的响应体长度。
	errorBodyLimit = 300
)

// 业务码。
const (
	// CodeOK 网关层成功；仍需检查 Result 层。
	CodeOK = 0
	// CodeRiskControl 风控命中，普通 HTTP 请求无法通过（Result 层 code）。
	CodeRiskControl = 403
)

// APIError 表示本次请求没有拿到可用的业务数据。
//
// 分三种情况：HTTP 层失败（HTTPStatus 非 0）、风控命中（Code 非 0）、
// 响应无法解析（Message 非空且前两者为 0）。
type APIError struct {
	// Path 本次调用的接口路径，便于区分同一客户端下的不同接口。
	Path string
	// URL 完整请求地址，便于排查参数与来源页。
	URL string
	// HTTPStatus HTTP 状态码；请求未到达服务端时为 0。
	HTTPStatus int
	// Code 业务层错误码（`Result.code`），如风控为 403。
	Code int
	// Message 错误描述，优先取服务端返回的 msg。
	Message string
	// Body 服务端原始响应体（已截断），便于定位解析失败原因。
	Body string
}

func (e *APIError) Error() string {
	switch {
	case e.Code != 0:
		return fmt.Sprintf("百度 %s 调用失败：code=%d message=%s（url=%s）", e.Path, e.Code, e.Message, e.URL)
	case e.HTTPStatus != 0:
		return fmt.Sprintf("百度 %s HTTP 错误：%d %s（url=%s）", e.Path, e.HTTPStatus, e.Message, e.URL)
	default:
		return fmt.Sprintf("百度 %s 请求失败：%s（url=%s）", e.Path, e.Message, e.URL)
	}
}

// IsRiskControl 判断错误是否为站点风控命中。
//
// 命中时外层 ResultCode 仍为 0，只有 Result 层的 code 为 403；出现该错误说明需要降频、
// 复用连接或更换出口网络，而不是调整参数。
func IsRiskControl(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Code == CodeRiskControl
}

// Config 客户端配置；零值代表使用协议默认值。
type Config struct {
	// BaseURL 接口站点，默认 DefaultBaseURL，结尾斜杠会被去掉。
	BaseURL string
	// ConstituentsPath 成分股列表接口路径，默认 DefaultConstituentsPath。
	ConstituentsPath string
	// QuotePath 行情接口路径，默认 DefaultQuotePath。
	QuotePath string
	// SuggestPath 搜索接口路径，默认 DefaultSuggestPath。
	SuggestPath string
	// FuturesFiveDayPath 期货五日分时接口路径，默认 DefaultFuturesFiveDayPath。
	FuturesFiveDayPath string
	// Referer 来源页，默认 DefaultReferer。
	//
	// 建议传个股页或板块页地址（`https://finance.baidu.com/stock/ab-600009`），
	// 比通用首页更贴近真实浏览，风控更宽松。
	Referer string
	// UserAgent 请求 UA，默认 DefaultUserAgent。
	UserAgent string
	// Header 额外请求头；在内置默认头之后逐项 Set，同名覆盖内置值。
	Header map[string]string
	// Timeout 单次请求超时，默认 20s。
	Timeout time.Duration
	// HTTPClient 可注入自定义客户端，主要用于测试与代理场景。
	HTTPClient *http.Client
}

func (c Config) withDefaults() Config {
	if strings.TrimSpace(c.BaseURL) == "" {
		c.BaseURL = DefaultBaseURL
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	c.ConstituentsPath = normalizePath(c.ConstituentsPath, DefaultConstituentsPath)
	c.QuotePath = normalizePath(c.QuotePath, DefaultQuotePath)
	c.SuggestPath = normalizePath(c.SuggestPath, DefaultSuggestPath)
	c.FuturesFiveDayPath = normalizePath(c.FuturesFiveDayPath, DefaultFuturesFiveDayPath)
	if strings.TrimSpace(c.Referer) == "" {
		c.Referer = DefaultReferer
	}
	if strings.TrimSpace(c.UserAgent) == "" {
		c.UserAgent = DefaultUserAgent
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	return c
}

// normalizePath 为空时取默认值，并补上前导斜杠。
func normalizePath(path, fallback string) string {
	if strings.TrimSpace(path) == "" {
		path = fallback
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

// Param 是查询参数的「名称 + 值」，按切片顺序拼进查询串。
//
// 百度接口不签名、参数顺序不影响结果，但仍按文档中的参数顺序构造，
// 便于与文档、抓包记录逐项对照。
type Param struct {
	Name  string
	Value string
}

// Client 是百度股市通接口客户端。
//
// Client 并发安全：创建后仅读取配置，内部不保存可变状态。
type Client struct {
	cfg  Config
	http *http.Client
}

// New 创建客户端，零值配置按协议默认值补齐。
func New(cfg Config) *Client {
	cfg = cfg.withDefaults()
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &Client{cfg: cfg, http: client}
}

// Config 返回生效配置（已补齐默认值）。
func (c *Client) Config() Config { return c.cfg }

// get 按有序参数向指定路径发一次 GET 请求并返回响应体。
func (c *Client) get(ctx context.Context, path string, params []Param) ([]byte, error) {
	target := c.cfg.BaseURL + path
	if len(params) > 0 {
		target += "?" + encodeQuery(params)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, &APIError{Path: path, URL: target, Message: "构造请求失败: " + err.Error()}
	}
	// 不发送 content-type：GET 无请求体，百度按此判断请求是否来自浏览器。
	req.Header.Set("accept", "application/json, text/plain, */*")
	req.Header.Set("referer", c.cfg.Referer)
	req.Header.Set("user-agent", c.cfg.UserAgent)
	for name, value := range c.cfg.Header {
		if strings.TrimSpace(name) == "" {
			continue
		}
		req.Header.Set(name, value)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &APIError{Path: path, URL: target, Message: "请求失败: " + err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, &APIError{
			Path: path, URL: target, HTTPStatus: resp.StatusCode,
			Message: "读取响应失败: " + err.Error(),
		}
	}
	if resp.StatusCode != http.StatusOK {
		apiErr := &APIError{
			Path: path, URL: target, HTTPStatus: resp.StatusCode,
			Body: truncate(string(body), errorBodyLimit),
		}
		// 风控时 HTTP 状态同样是 403，业务码在响应体里；识别出来让上层统一用
		// IsRiskControl 判断，不必区分「HTTP 403」与「HTTP 200 + Result.code=403」两种形态。
		if code, message, ok := parseRiskControlBody(body); ok {
			apiErr.Code = code
			apiErr.Message = message
		}
		return nil, apiErr
	}
	return body, nil
}

// parseRiskControlBody 从响应体中取出业务层风控标记。
func parseRiskControlBody(body []byte) (int, string, bool) {
	var envelope struct {
		Result json.RawMessage `json:"Result"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return 0, "", false
	}
	raw := bytes.TrimSpace(envelope.Result)
	if len(raw) == 0 || raw[0] != '{' {
		return 0, "", false
	}
	var risk riskControl
	if err := json.Unmarshal(raw, &risk); err != nil || risk.Code == 0 {
		return 0, "", false
	}
	return risk.Code, risk.Message, true
}

// encodeQuery 按参数切片顺序拼接查询串。
func encodeQuery(params []Param) string {
	var b strings.Builder
	for i, p := range params {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(p.Name))
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(p.Value))
	}
	return b.String()
}

// truncate 截断过长的文本，避免错误信息携带整个响应体。
func truncate(s string, limit int) string {
	s = strings.TrimSpace(s)
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}
