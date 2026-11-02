// Package quoteapiv2 封装 moomoo（富途）网页行情接口 /quote-api/quote-v2/*。
//
// 本包只做「HTTP 协议 → 统一结构」的转换，不涉及存储与业务编排，结果交给调用方
// 自行处理。当前实现 get-kline 一个接口，其余接口可按同样的方式在 Client 上扩展：
// 构造有序参数、签名、发请求、解析响应四步。
//
// 对外方法名与接口路径保持映射关系：取路径末段按驼峰命名，例如 /quote-api/quote-v2/
// get-kline 对应 GetKLine，要新增接口时沿用同一规则，以便从方法名直接看出调用的
// 是哪个接口。请求参数与返回类型同样带上接口前缀，避免不同接口之间用泛泛的通用名。
//
// 请求流程上有两点必须遵守：
//
//   - 查询串的键序就是签名输入的键序，不能用 map 或 url.Values 排序后发送；
//   - 所有参数值都要先字符串化再参与签名，数值类型的差异会导致 token 不一致。
package quoteapiv2

import (
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
	// DefaultLocale Cookie 中的语言标记。
	DefaultLocale = "zh-cn"
	// DefaultReferer 请求来源页。
	DefaultReferer = "https://www.moomoo.com/"
	// DefaultUserAgent 浏览器 UA；站点对非浏览器 UA 可能直接拒绝。
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"
	// DefaultTimeout 单次请求超时。
	DefaultTimeout = 20 * time.Second
	// KLineTypeDay 日 K 线类型编号。
	KLineTypeDay = "2"
)

// 业务错误码。
const (
	// CodeOK 业务成功。
	CodeOK = 0
	// CodeTokenInvalid quote-token 被服务端判定无效。
	CodeTokenInvalid = -403
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

// Config 客户端配置；零值代表使用协议默认值。
type Config struct {
	// BaseURL 接口站点，默认 https://www.moomoo.com，结尾斜杠会被去掉。
	BaseURL string
	// Path 接口路径，默认 /quote-api/quote-v2/get-kline。
	Path string
	// CSRF 对应请求头 futu-x-csrf-token 与 Cookie 中的 csrfToken。
	// 实测 get-kline 不校验该值，但请求中必须存在，故默认使用占位值。
	CSRF string
	// Locale Cookie 语言标记，默认 zh-cn。
	Locale string
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
	if strings.TrimSpace(c.Path) == "" {
		c.Path = DefaultKLinePath
	}
	if !strings.HasPrefix(c.Path, "/") {
		c.Path = "/" + c.Path
	}
	if strings.TrimSpace(c.CSRF) == "" {
		// 服务端不校验，但请求头必须存在；用占位值避免误以为需要登录态。
		c.CSRF = "0000000000"
	}
	if strings.TrimSpace(c.Locale) == "" {
		c.Locale = DefaultLocale
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	return c
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

// GetKLineRequest K 线请求参数（对应 get-kline 接口）。
//
// 五个定位参数（StockID / MarketType / MarketCode / InstrumentType /
// SubInstrumentType）原样透传，不做语义换算：同一编号在不同接口含义不同，
// 换算会引入难以排查的错位。
type GetKLineRequest struct {
	// StockID 标的编号，必填。
	StockID string
	// MarketType 市场类型。
	MarketType string
	// KLineType 周期类型，为空取 KLineTypeDay。
	KLineType string
	// MarketCode 市场代码。
	MarketCode string
	// InstrumentType 标的类型。
	InstrumentType string
	// SubInstrumentType 标的子类型。
	SubInstrumentType string
	// TimestampMillis 毫秒时间戳（查询串 _ 参数），为空自动取当前时间。
	TimestampMillis string
}

// KLineBar 一根 K 线，保持接口原始口径：时间为秒级 UNIX 时间戳，价格保留字符串
// 以免浮点精度损失，数值仅在明确是数字时使用 float64 承载。由 GetKLine 返回。
type KLineBar struct {
	// Time 秒级 UNIX 时间戳。
	Time int64
	// Open 开盘价原始字符串，可能为空。
	Open string
	// High 最高价原始字符串，可能为空。
	High string
	// Low 最低价原始字符串，可能为空。
	Low string
	// Close 收盘价原始字符串，可能为空。
	Close string
	// Volume 成交量。
	Volume float64
	// Turnover 成交额。
	Turnover float64
	// ChangeRatio 涨跌幅，接口可能返回 null，故用指针区分「无值」与「0」。
	ChangeRatio *float64
	// LastClose 昨收价。
	LastClose float64
	// ChangePrice 涨跌额原始字符串，含符号。
	ChangePrice string
}

// GetKLine 拉取全量 K 线序列（对应 get-kline 接口）。
//
// 该接口无分页与时间范围参数，一次返回全部历史数据；返回值按时间升序（最早在前），
// 与接口顺序一致，区间过滤由调用方完成。
func (c *Client) GetKLine(ctx context.Context, req GetKLineRequest) ([]KLineBar, error) {
	if strings.TrimSpace(req.StockID) == "" {
		return nil, errors.New("moomoo GetKLine：stockId 不能为空")
	}
	klineType := strings.TrimSpace(req.KLineType)
	if klineType == "" {
		klineType = KLineTypeDay
	}
	timestamp := strings.TrimSpace(req.TimestampMillis)
	if timestamp == "" {
		timestamp = strconv.FormatInt(c.cfg.Now().UnixMilli(), 10)
	}
	// 参数顺序即签名键序，不可随意调整。
	params := []Param{
		{Name: "stockId", Value: strings.TrimSpace(req.StockID)},
		{Name: "marketType", Value: strings.TrimSpace(req.MarketType)},
		{Name: "type", Value: klineType},
		{Name: "marketCode", Value: strings.TrimSpace(req.MarketCode)},
		{Name: "instrumentType", Value: strings.TrimSpace(req.InstrumentType)},
		{Name: "subInstrumentType", Value: strings.TrimSpace(req.SubInstrumentType)},
		{Name: "_", Value: timestamp},
	}

	body, err := c.get(ctx, params)
	if err != nil {
		return nil, err
	}
	var payload klinePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, &APIError{
			Path: c.cfg.Path, URL: c.url(params), Body: truncate(string(body), 500),
			Message: "响应解析失败: " + err.Error(),
		}
	}
	if payload.Code != CodeOK {
		return nil, &APIError{
			Path: c.cfg.Path, URL: c.url(params), Code: payload.Code,
			Message: payload.Message, Body: truncate(string(body), 500),
		}
	}
	return payload.Data.Bars(), nil
}

// get 按有序参数发一次 GET 请求，返回响应体；非 200 视为失败。
func (c *Client) get(ctx context.Context, params []Param) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(params), nil)
	if err != nil {
		return nil, fmt.Errorf("构造 moomoo 请求失败: %w", err)
	}
	req.Header.Set("quote-token", SignParams(params))
	req.Header.Set("futu-x-csrf-token", c.cfg.CSRF)
	req.Header.Set("cookie", fmt.Sprintf("csrfToken=%s; locale=%s", c.cfg.CSRF, c.cfg.Locale))
	req.Header.Set("accept", "application/json")
	req.Header.Set("referer", DefaultReferer)
	req.Header.Set("user-agent", DefaultUserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &APIError{Path: c.cfg.Path, URL: req.URL.String(), Message: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, &APIError{Path: c.cfg.Path, URL: req.URL.String(), HTTPStatus: resp.StatusCode, Message: err.Error()}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{
			Path: c.cfg.Path, URL: req.URL.String(), HTTPStatus: resp.StatusCode,
			Body: truncate(string(body), 500),
		}
	}
	return body, nil
}

// url 拼接完整请求地址。
func (c *Client) url(params []Param) string {
	return c.cfg.BaseURL + c.cfg.Path + "?" + SignedQuery(params)
}

// klinePayload K 线响应外层包裹。
type klinePayload struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Data    klineData `json:"data"`
}

// klineItem 接口原始字段：开高低收与涨跌额为字符串，k/v/t/lc 为数值，
// 类型混用是接口行为，这里统一用 flexNum 兼容两种写法。
type klineItem struct {
	K  int64    `json:"k"`
	O  flexNum  `json:"o"`
	H  flexNum  `json:"h"`
	L  flexNum  `json:"l"`
	C  flexNum  `json:"c"`
	V  flexNum  `json:"v"`
	T  flexNum  `json:"t"`
	R  *flexNum `json:"r"`
	LC flexNum  `json:"lc"`
	CP flexNum  `json:"cp"`
}

// klineData 响应 data 字段。
type klineData struct {
	List []klineItem `json:"list"`
}

// Bars 把原始条目转换为 KLineBar 切片。
func (p klineData) Bars() []KLineBar {
	bars := make([]KLineBar, 0, len(p.List))
	for _, item := range p.List {
		bar := KLineBar{
			Time:        item.K,
			Open:        item.O.String(),
			High:        item.H.String(),
			Low:         item.L.String(),
			Close:       item.C.String(),
			Volume:      item.V.Float(),
			Turnover:    item.T.Float(),
			LastClose:   item.LC.Float(),
			ChangePrice: item.CP.String(),
		}
		if item.R != nil {
			ratio := item.R.Float()
			bar.ChangeRatio = &ratio
		}
		bars = append(bars, bar)
	}
	return bars
}

// flexNum 兼容「字符串数字」与「数字」两种 JSON 表示。
type flexNum string

// UnmarshalJSON 接受字符串、数字与 null 三种形式。
func (n *flexNum) UnmarshalJSON(data []byte) error {
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
		*n = flexNum(s)
		return nil
	}
	*n = flexNum(text)
	return nil
}

// String 返回原始字符串形式。
func (n flexNum) String() string { return string(n) }

// Float 转为 float64；空串或非法值返回 0（接口对缺失量给 0 而非 null）。
func (n flexNum) Float() float64 {
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
