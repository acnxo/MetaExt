// Package quoteapiv2 封装 moomoo（富途）网页行情接口 /quote-api/quote-v2/*。
//
// 本包只做「HTTP 协议 → 统一结构」的转换，不涉及存储与业务编排，结果交给调用方
// 自行处理。当前实现 get-kline 与 get-stock-list 两个接口，其余接口可按同样的方式在
// Client 上扩展：构造有序参数、签名、发请求、解析响应四步。
//
// 对外方法名与接口路径保持映射关系：取路径末段按驼峰命名，例如 /quote-api/quote-v2/
// get-kline 对应 GetKLine、get-stock-list 对应 GetStockList。要新增接口时沿用同一规则，
// 以便从方法名直接看出调用的是哪个接口。请求参数与返回类型同样带上接口前缀，避免不同
// 接口之间用泛泛的通用名。
//
// 每个接口一个源文件：client.go 只放配置、错误与请求发送等公共部分，kline.go、
// stocklist.go 分别放各自接口的请求参数、返回结构与解析。
//
// 请求流程上有两点必须遵守：
//
//   - 查询串的键序就是签名输入的键序，不能用 map 或 url.Values 排序后发送；
//   - 所有参数值都要先字符串化再参与签名，数值类型的差异会导致 token 不一致。
package quoteapiv2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 协议默认值。
const (
	// DefaultBaseURL 接口站点。
	DefaultBaseURL = "https://www.moomoo.com"
	// DefaultKLinePath K 线接口路径。
	DefaultKLinePath = "/quote-api/quote-v2/get-kline"
	// DefaultStockListPath 板块股票列表接口路径。
	DefaultStockListPath = "/quote-api/quote-v2/get-stock-list"
	// DefaultOptionQuotePath 期权合约批量报价接口路径。
	DefaultOptionQuotePath = "/quote-api/quote-v2/get-option-quote"
	// DefaultNewsListPath 个股新闻列表接口路径。
	DefaultNewsListPath = "/quote-api/quote-v2/get-news-list"
	// DefaultEtfListPath ETF 列表接口路径。
	DefaultEtfListPath = "/quote-api/quote-v2/get-etf-list"
	// DefaultFuturesListPath 期货合约列表接口路径。
	DefaultFuturesListPath = "/quote-api/quote-v2/get-futures-list"
	// DefaultHistoryCashTrendPath 历史资金流水接口路径。
	DefaultHistoryCashTrendPath = "/quote-api/quote-v2/get-history-cash-trend"
	// DefaultQuoteMinutePath 分钟级分时行情接口路径。
	DefaultQuoteMinutePath = "/quote-api/quote-v2/get-quote-minute"
	// DefaultOrigin POST 请求的 Origin 头。
	DefaultOrigin = "https://www.moomoo.com"
	// DefaultLocale Cookie 中的语言标记。
	DefaultLocale = "zh-cn"
	// DefaultReferer 默认请求来源页，简体中文站首页。
	//
	// 该值决定服务端返回数据的语言与数值单位口径，不只是「来源标识」：
	// 站点根路径（https://www.moomoo.com/）返回英文名与 T/B/M/K 单位，
	// 简体中文站（/hans/）返回中文名与 万亿/亿/万 单位。
	DefaultReferer = "https://www.moomoo.com/hans/"
	// DefaultUserAgent 浏览器 UA；站点对非浏览器 UA 可能直接拒绝。
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"
	// DefaultTimeout 单次请求超时。
	DefaultTimeout = 20 * time.Second
	// KLineTypeDay 日 K 线类型编号。
	KLineTypeDay = "2"
	// DefaultPlateType 股票列表接口的板块类型默认值。
	DefaultPlateType = "1"
	// DefaultNewsMarketType 新闻列表接口的市场类型默认值。
	DefaultNewsMarketType = "1"
	// DefaultNewsType 新闻列表接口的新闻类型默认值；缺该参数服务端会返回 404。
	DefaultNewsType = "0"
	// DefaultNewsSubType 新闻列表接口的新闻子类型默认值。
	DefaultNewsSubType = "0"
	// DefaultEtfRouteType ETF 列表接口的路由类型默认值。
	DefaultEtfRouteType = "1"
	// DefaultEtfPlateID ETF 列表接口的板块 ID 默认值（美股全部 ETF）。
	DefaultEtfPlateID = "10002799"
	// DefaultEtfRankType ETF 列表接口的排行类型默认值（按 AUM 降序）。
	DefaultEtfRankType = "6"
	// DefaultEtfPage ETF 列表接口的页码默认值；该接口页码从 0 开始。
	DefaultEtfPage = "0"
	// DefaultEtfPageSize ETF 列表接口的每页数量默认值。
	DefaultEtfPageSize = "50"
	// DefaultFuturesPage 期货列表接口的页码默认值；该接口页码从 0 开始。
	DefaultFuturesPage = "0"
	// DefaultFuturesPageSize 期货列表接口的每页数量默认值。
	DefaultFuturesPageSize = "40"
	// DefaultCashTrendBaseTime 资金流水接口的分页基准时间戳默认值，0 表示最新一段。
	DefaultCashTrendBaseTime = "0"

	// CashTrendTypeDay 资金流水粒度：日。
	CashTrendTypeDay = "1"
	// CashTrendTypeWeek 资金流水粒度：周。
	CashTrendTypeWeek = "2"
	// CashTrendTypeMonth 资金流水粒度：月。
	CashTrendTypeMonth = "3"
)

// DefaultReferers 按市场类型（marketType）区分的默认来源页，取对应市场的简体中文列表页。
//
// 列表类接口请求会带 marketType，命中本表比 DefaultReferer 更贴近实际浏览页面；
// 未列出的市场回落 DefaultReferer。key 为接口的 marketType 取值。
var DefaultReferers = map[string]string{
	"1": "https://www.moomoo.com/hans/quote/hk/stock-list/all-hk-stocks/top-market-cap",
	"2": "https://www.moomoo.com/hans/quote/us/stock-list/all-us-stocks/top-market-cap",
	"4": "https://www.moomoo.com/hans/quote/cn/stock-list/all-cn-stocks/top-market-cap",
}

// 业务错误码。
const (
	// CodeOK 业务成功。
	CodeOK = 0
	// CodeTokenInvalid quote-token 被服务端判定无效。
	CodeTokenInvalid = -403
	// CodeParamsError 必填参数缺失或取值越界，服务端消息通常为 Params Error。
	CodeParamsError = 500
	// CodeParamsMissing 必填参数缺失，如新闻列表缺少 type 时返回 404。
	CodeParamsMissing = 404
)

// APIError 接口层错误：HTTP 失败或业务码非成功。
type APIError struct {
	// Path 本次调用的接口路径，用于区分同一客户端下的不同接口。
	Path string
	// URL 完整请求地址（含查询串），便于排查签名与参数问题。
	URL string
	// HTTPStatus HTTP 状态码；请求未到达服务端时为 0。
	HTTPStatus int
	// Code 业务错误码；HTTP 层失败时为 0。
	Code int
	// Message 服务端返回的错误描述或本地错误描述。
	Message string
	// Body 服务端原始响应体（已截断），便于定位解析失败原因。
	Body string
}

func (e *APIError) Error() string {
	switch {
	case e.Code != 0:
		return fmt.Sprintf("moomoo %s 调用失败：code=%d message=%s（url=%s）", e.Path, e.Code, e.Message, e.URL)
	case e.HTTPStatus != 0:
		return fmt.Sprintf("moomoo %s HTTP 错误：%d（url=%s）", e.Path, e.HTTPStatus, e.URL)
	case e.Message != "":
		// 解析失败、网络错误等情况下没有状态码与业务码，必须带上原因。
		return fmt.Sprintf("moomoo %s 请求失败：%s（url=%s）", e.Path, e.Message, e.URL)
	default:
		return fmt.Sprintf("moomoo %s 请求失败（url=%s）", e.Path, e.URL)
	}
}

// IsTokenRejected 判断错误是否为 quote-token 被服务端拒绝。
//
// 出现该错误通常意味着签名输入构造有误（键序、值字符串化或参数遗漏），
// 而不是站点风控。
func IsTokenRejected(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Code == CodeTokenInvalid
}

// IsParamsError 判断错误是否为必填参数缺失或取值越界。
func IsParamsError(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Code == CodeParamsError
}

// IsParamsMissing 判断错误是否为必填参数缺失（服务端以 404 表示）。
func IsParamsMissing(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Code == CodeParamsMissing
}

// Config 客户端配置；零值代表使用协议默认值。
type Config struct {
	// BaseURL 接口站点，默认 https://www.moomoo.com，结尾斜杠会被去掉。
	BaseURL string
	// KLinePath K 线接口路径，默认 DefaultKLinePath。
	KLinePath string
	// StockListPath 板块股票列表接口路径，默认 DefaultStockListPath。
	StockListPath string
	// OptionQuotePath 期权合约批量报价接口路径，默认 DefaultOptionQuotePath。
	OptionQuotePath string
	// NewsListPath 个股新闻列表接口路径，默认 DefaultNewsListPath。
	NewsListPath string
	// EtfListPath ETF 列表接口路径，默认 DefaultEtfListPath。
	EtfListPath string
	// FuturesListPath 期货合约列表接口路径，默认 DefaultFuturesListPath。
	FuturesListPath string
	// HistoryCashTrendPath 历史资金流水接口路径，默认 DefaultHistoryCashTrendPath。
	HistoryCashTrendPath string
	// QuoteMinutePath 分钟级分时行情接口路径，默认 DefaultQuoteMinutePath。
	QuoteMinutePath string
	// CSRF 对应请求头 futu-x-csrf-token 与 Cookie 中的 csrfToken。
	// 实测该接口不校验该值，但请求中必须存在，故默认使用占位值。
	CSRF string
	// Locale Cookie 语言标记，默认 zh-cn。
	//
	// 注意：该值只写进 Cookie，实测不改变返回数据的语言；语言由 Referer 决定。
	Locale string
	// Referer 全局请求来源页；为空时依次回落 Referers（按市场）、
	// DefaultReferers（按市场）、DefaultReferer。
	//
	// 注意：该值决定返回数据的语言与数值单位口径（中文站 /hans/ → 中文名与 万亿/亿/万）。
	Referer string
	// Referers 按市场类型（marketType）区分的来源页，优先于 Referer；
	// 未命中时回落 Referer 与默认值。
	Referers map[string]string
	// Header 额外请求头；在内置默认头与 referer 之后逐项 Set，同名覆盖内置值。
	//
	// 键为空白时跳过；New 时做浅拷贝，调用方可安全复用同一个 map。
	// quote-token 与 cookie 请优先用专用字段（CSRF / Locale），Header 仅作兼容扩展点，
	// 覆盖它们会绕过本包对签名与 CSRF 的一致性保证。
	Header map[string]string
	// Timeout 单次请求超时，默认 20s。
	Timeout time.Duration
	// HTTPClient 可注入自定义客户端，主要用于测试与代理场景。
	HTTPClient *http.Client
	// Now 可注入时钟，用于测试时间戳与签名。
	Now func() time.Time
}

// withDefaults 补齐零值字段并规整路径格式。
func (c Config) withDefaults() Config {
	if strings.TrimSpace(c.BaseURL) == "" {
		c.BaseURL = DefaultBaseURL
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	c.KLinePath = normalizePath(c.KLinePath, DefaultKLinePath)
	c.StockListPath = normalizePath(c.StockListPath, DefaultStockListPath)
	c.OptionQuotePath = normalizePath(c.OptionQuotePath, DefaultOptionQuotePath)
	c.NewsListPath = normalizePath(c.NewsListPath, DefaultNewsListPath)
	c.EtfListPath = normalizePath(c.EtfListPath, DefaultEtfListPath)
	c.FuturesListPath = normalizePath(c.FuturesListPath, DefaultFuturesListPath)
	c.HistoryCashTrendPath = normalizePath(c.HistoryCashTrendPath, DefaultHistoryCashTrendPath)
	c.QuoteMinutePath = normalizePath(c.QuoteMinutePath, DefaultQuoteMinutePath)
	if strings.TrimSpace(c.CSRF) == "" {
		// 服务端不校验，但请求头必须存在；用占位值避免误以为需要登录态。
		c.CSRF = "0000000000"
	}
	if strings.TrimSpace(c.Locale) == "" {
		c.Locale = DefaultLocale
	}
	// 浅拷贝调用方提供的 map：避免其运行期修改导致数据竞争，也避免本包回写。
	if len(c.Referers) > 0 {
		referers := make(map[string]string, len(c.Referers))
		for key, value := range c.Referers {
			referers[key] = value
		}
		c.Referers = referers
	}
	if len(c.Header) > 0 {
		header := make(map[string]string, len(c.Header))
		for key, value := range c.Header {
			header[key] = value
		}
		c.Header = header
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	if c.Now == nil {
		c.Now = time.Now
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

// defaultIfEmpty 去除首尾空白，为空时返回 fallback。
func defaultIfEmpty(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

// intParam 校验收到的整数字符串：为空表示省略，非空则必须是大于等于 min 的整数。
func intParam(value, field string, min int) (string, error) {
	text := strings.TrimSpace(value)
	if text == "" {
		return "", nil
	}
	number, err := strconv.Atoi(text)
	if err != nil {
		return "", fmt.Errorf("moomoo %s 必须是整数，实际 %q", field, text)
	}
	if number < min {
		return "", fmt.Errorf("moomoo %s 不能小于 %d，实际 %d", field, min, number)
	}
	return text, nil
}

// Client moomoo 行情接口客户端。
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

// get 按有序参数向指定路径发一次 GET 请求；签名输入为查询参数。
//
// referer 取全局默认值（含 Config.Referer / Config.Header 覆盖）；列表类接口请用 getList。
func (c *Client) get(ctx context.Context, path string, params []Param) ([]byte, error) {
	return c.do(ctx, http.MethodGet, path, params, nil, SignParams(params), "")
}

// getList 与 get 相同，但 referer 按 marketType 选择对应市场的列表页。
//
// 列表类接口（股票 / ETF / 期货）的请求参数含 marketType，命中
// DefaultReferers / Config.Referers 比通用来源页更贴近真实浏览页面，
// 服务端据此决定返回数据的语言与数值单位口径。
func (c *Client) getList(ctx context.Context, path string, params []Param) ([]byte, error) {
	return c.do(ctx, http.MethodGet, path, params, nil, SignParams(params), marketTypeParam(params))
}

// post 向指定路径发一次 POST 请求；签名输入为请求体的原始字节。
//
// body 必须是最终要发送的紧凑 JSON：签名对字节逐字敏感，签名后不得再格式化。
func (c *Client) post(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.do(ctx, http.MethodPost, path, nil, body, Sign(string(body)), "")
}

// do 按给定方法发一次请求并返回响应体；非 200 视为失败。
//
// token 由调用方计算：GET 走查询参数签名，POST 走请求体签名，本方法不区分。
// refererMarket 非空时优先用对应市场的来源页；为空时用全局默认值。
func (c *Client) do(ctx context.Context, method, path string, query []Param, body []byte, token string, refererMarket string) ([]byte, error) {
	target := c.cfg.BaseURL + path
	if len(query) > 0 {
		target = c.url(path, query)
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return nil, fmt.Errorf("构造 moomoo 请求失败: %w", err)
	}
	req.Header.Set("quote-token", token)
	req.Header.Set("futu-x-csrf-token", c.cfg.CSRF)
	req.Header.Set("cookie", fmt.Sprintf("csrfToken=%s; locale=%s", c.cfg.CSRF, c.cfg.Locale))
	req.Header.Set("accept", "application/json")
	req.Header.Set("referer", c.refererFor(refererMarket))
	req.Header.Set("user-agent", DefaultUserAgent)
	if method == http.MethodPost {
		req.Header.Set("content-type", "application/json")
		req.Header.Set("origin", DefaultOrigin)
	}
	// 扩展头最后应用：同名覆盖内置值（含 referer），调用方显式值优先。
	c.applyHeader(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &APIError{Path: path, URL: req.URL.String(), Message: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, &APIError{Path: path, URL: req.URL.String(), HTTPStatus: resp.StatusCode, Message: err.Error()}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{
			Path: path, URL: req.URL.String(), HTTPStatus: resp.StatusCode,
			Body: truncate(string(respBody), 500),
		}
	}
	return respBody, nil
}

// url 拼接完整请求地址。
func (c *Client) url(path string, params []Param) string {
	return c.cfg.BaseURL + path + "?" + SignedQuery(params)
}

// refererFor 按「市场覆盖 → 全局覆盖 → 市场默认 → 全局默认」选择来源页。
//
// Config.Header 中的 referer 名不在此处理，它在所有内置头之后覆盖本方法的返回值。
func (c *Client) refererFor(marketType string) string {
	if marketType != "" {
		if value := strings.TrimSpace(c.cfg.Referers[marketType]); value != "" {
			return value
		}
	}
	if value := strings.TrimSpace(c.cfg.Referer); value != "" {
		return value
	}
	if marketType != "" {
		if value := strings.TrimSpace(DefaultReferers[marketType]); value != "" {
			return value
		}
	}
	return DefaultReferer
}

// applyHeader 应用 Config.Header：同名覆盖内置头，键为空白时跳过。
func (c *Client) applyHeader(req *http.Request) {
	for key, value := range c.cfg.Header {
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}
		req.Header.Set(name, value)
	}
}

// marketTypeParam 从有序查询参数中取出 marketType，供选择 referer 使用。
func marketTypeParam(params []Param) string {
	for _, p := range params {
		if p.Name == "marketType" {
			return strings.TrimSpace(p.Value)
		}
	}
	return ""
}

// decode 检查业务码；非成功时按服务端返回的 message 构造错误，成功时返回 nil。
//
// path 与 url 只用于构造错误信息，便于定位是哪个接口、哪组参数出的问题。
func decode(body []byte, path string, url string, code int, message string) error {
	if code != CodeOK {
		return &APIError{
			Path: path, URL: url, Code: code,
			Message: message, Body: truncate(string(body), 500),
		}
	}
	return nil
}

// FlexNum 兼容「字符串数字」与「数字」两种 JSON 表示。
type FlexNum string

// UnmarshalJSON 接受字符串、数字与 null 三种形式。
func (n *FlexNum) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "" || text == "null" {
		*n = ""
		return nil
	}
	if text[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*n = FlexNum(s)
		return nil
	}
	*n = FlexNum(text)
	return nil
}

// String 返回原始字符串形式。
func (n FlexNum) String() string { return string(n) }

// Float 转为 float64；空串或非法值返回 0（接口对缺失量给 0 而非 null）。
func (n FlexNum) Float() float64 {
	if n == "" {
		return 0
	}
	value, err := strconv.ParseFloat(string(n), 64)
	if err != nil {
		return 0
	}
	return value
}

// truncate 截断过长的响应体，避免日志膨胀。
func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}
