package finscope

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// K 线周期（参数 ktype）。
const (
	// KTypeMin1 1 分钟线，默认值；配合 Count=241 取一个交易日。
	KTypeMin1 = "min1"
	// KTypeMin5 5 分钟线。
	KTypeMin5 = "min5"
	// KTypeDay 日线。
	KTypeDay = "day"
)

// GetQuoteKLineRequest 是 getquotation（`group=quotation_kline_ab`）的请求参数。
//
// 该接口风控比 constituents 严格得多：本机对指数、个股、各个 group 的请求都稳定返回
// `{"code":403,"isCaptchaEnabled":true,"msg":"hit risk"}`，换 UA、Referer、出口均无效。
// 封装保留完整参数是为了拿到可用出口后能直接使用。
type GetQuoteKLineRequest struct {
	// Code 证券代码：A 股 / 指数 6 位数字，港股 5 位，美股字母代码。
	Code string
	// MarketType 市场：MarketA / MarketHK / MarketUS。
	MarketType string
	// Name 证券简称，可空；传了更贴近浏览器真实请求。
	Name string
	// IsKC 注册制板块标记：科创板（688/689）、创业板（30）、北交所（8/4）为 1，其它为 0。
	IsKC int
	// KType 周期，取 KTypeMin1 / KTypeMin5 / KTypeDay，为空取 KTypeMin1。
	KType string
	// Count 返回条数，为空取 DefaultKLineCount（241，覆盖一个交易日的分钟线）。
	Count int
}

// withDefaults 补齐可省略字段并做必填校验。
func (r GetQuoteKLineRequest) withDefaults() (GetQuoteKLineRequest, error) {
	r.Code = strings.TrimSpace(r.Code)
	r.MarketType = strings.TrimSpace(r.MarketType)
	r.KType = strings.TrimSpace(r.KType)
	if r.Code == "" {
		return r, fmt.Errorf("百度 getquotation 参数 code 不能为空")
	}
	switch r.MarketType {
	case MarketA, MarketHK, MarketUS:
	case "":
		r.MarketType = MarketA
	default:
		return r, fmt.Errorf("百度 getquotation 参数 market_type 取值非法：%q", r.MarketType)
	}
	switch r.KType {
	case "":
		r.KType = KTypeMin1
	case KTypeMin1, KTypeMin5, KTypeDay:
	default:
		return r, fmt.Errorf("百度 getquotation 参数 ktype 取值非法：%q", r.KType)
	}
	if r.Count <= 0 {
		r.Count = DefaultKLineCount
	}
	if r.IsKC != 0 && r.IsKC != 1 {
		return r, fmt.Errorf("百度 getquotation 参数 is_kc 只能是 0 或 1，实际 %d", r.IsKC)
	}
	return r, nil
}

// params 按文档中的参数顺序构造查询参数。
func (r GetQuoteKLineRequest) params() []Param {
	params := []Param{
		{Name: "srcid", Value: DefaultSrcID},
		{Name: "pointType", Value: DefaultPointType},
		{Name: "group", Value: GroupKLine},
		{Name: "newFormat", Value: "1"},
		{Name: "finClientType", Value: DefaultClientType},
		{Name: "code", Value: r.Code},
		{Name: "query", Value: r.Code},
		{Name: "market_type", Value: r.MarketType},
	}
	if r.Name != "" {
		params = append(params, Param{Name: "name", Value: r.Name})
	}
	params = append(params,
		Param{Name: "is_kc", Value: strconv.Itoa(r.IsKC)},
		Param{Name: "ktype", Value: r.KType},
		Param{Name: "count", Value: strconv.Itoa(r.Count)},
	)
	return params
}

// KLineBar 是一根 K 线。
//
// 字段按响应里的 `keys` 逐列还原，而不是按下标硬编码，因此百度调整列顺序时不需要改代码。
// `keys` 中未识别的列保留在 Extra 中，便于新字段出现时不丢数据。
type KLineBar struct {
	// Timestamp 秒级时间戳，接口原值。
	Timestamp int64
	// Time 时间，形如 "2026-11-12 10:07"；日线为日期。
	Time string
	// Open 开盘价。
	Open float64
	// Close 收盘价。
	Close float64
	// High 最高价。
	High float64
	// Low 最低价。
	Low float64
	// Volume 成交量。
	Volume float64
	// Amount 成交额。
	Amount float64
	// ChangePrice 涨跌额。
	ChangePrice float64
	// ChangeRatio 涨跌幅（百分数）。
	ChangeRatio float64
	// TurnoverRatio 换手率（百分数），分钟线上常缺省，故用指针区分「无值」。
	TurnoverRatio *float64
	// LastClose 昨收，可能缺省。
	LastClose *float64
	// Extra 未识别的列，键为 `keys` 中的原始字段名。
	Extra map[string]string
}

// KLineResult 是 K 线响应解析后的结果。
type KLineResult struct {
	// Headers 中文列名，仅供展示。
	Headers []string
	// Keys 字段名，决定 marketData 的列顺序。
	Keys []string
	// Bars 解析后的 K 线，按接口返回顺序（默认时间升序）。
	Bars []KLineBar
	// Skipped 因字段数不足被跳过的残缺记录数。
	Skipped int
}

// GetQuoteKLine 拉取 K 线。
//
// 状态：blocked —— 请求参数已按文档构造，但本机持续被站点风控 403，字段解析只用文档样本的单测
// 覆盖过。复测命令见 TestLive，通过后把 verification.go 里的状态改为 StatusVerified。
//
// 风控命中时返回 *APIError（Code=403），可用 IsRiskControl 判断；不会把风控响应当成空结果。
func (c *Client) GetQuoteKLine(ctx context.Context, req GetQuoteKLineRequest) (*KLineResult, error) {
	normalized, err := req.withDefaults()
	if err != nil {
		return nil, err
	}
	path := c.cfg.QuotePath
	params := normalized.params()
	body, err := c.get(ctx, path, params)
	if err != nil {
		return nil, err
	}
	return parseKLine(path, c.cfg.BaseURL+path+"?"+encodeQuery(params), body)
}

// kLineEnvelope 是 K 线响应外层。
type kLineEnvelope struct {
	QueryID    string          `json:"QueryID"`
	ResultNum  int             `json:"ResultNum"`
	ResultCode int             `json:"ResultCode"`
	Result     json.RawMessage `json:"Result"`
}

// kLineMarketData 是正常响应里的 K 线容器。
type kLineMarketData struct {
	Headers    []string `json:"headers"`
	Keys       []string `json:"keys"`
	MarketData string   `json:"marketData"`
}

// parseKLine 解析 K 线响应；风控与结构异常转成 *APIError。
func parseKLine(path, targetURL string, body []byte) (*KLineResult, error) {
	var envelope kLineEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, &APIError{
			Path: path, URL: targetURL, Body: truncate(string(body), errorBodyLimit),
			Message: "响应解析失败: " + err.Error(),
		}
	}
	resultRaw := bytes.TrimSpace(envelope.Result)
	if len(resultRaw) == 0 || string(resultRaw) == "null" {
		return nil, &APIError{
			Path: path, URL: targetURL, Body: truncate(string(body), errorBodyLimit),
			Message: "响应缺少 Result 字段",
		}
	}
	if resultRaw[0] != '{' {
		return nil, &APIError{
			Path: path, URL: targetURL, Body: truncate(string(body), errorBodyLimit),
			Message: "响应的 Result 不是对象",
		}
	}
	var risk riskControl
	if err := json.Unmarshal(resultRaw, &risk); err == nil && risk.Code != 0 {
		return nil, &APIError{
			Path: path, URL: targetURL, Code: risk.Code, Message: risk.Message,
			Body: truncate(string(body), errorBodyLimit),
		}
	}
	var wrapper struct {
		NewMarketData kLineMarketData `json:"newMarketData"`
	}
	if err := json.Unmarshal(resultRaw, &wrapper); err != nil {
		return nil, &APIError{
			Path: path, URL: targetURL, Body: truncate(string(body), errorBodyLimit),
			Message: "响应 Result 解析失败: " + err.Error(),
		}
	}
	return decodeKLine(wrapper.NewMarketData)
}

// decodeKLine 按 keys 把 marketData 字符串拆成 K 线序列。
//
// 记录用 `;` 分隔、字段用 `,` 分隔，缺失值写 `--`；字段数不足 len(keys) 的记录跳过。
func decodeKLine(data kLineMarketData) (*KLineResult, error) {
	result := &KLineResult{Headers: data.Headers, Keys: data.Keys}
	text := strings.TrimSpace(data.MarketData)
	if text == "" {
		return result, nil
	}
	if len(data.Keys) == 0 {
		return nil, fmt.Errorf("百度 K 线响应缺少 keys，无法确定列顺序")
	}
	for _, record := range strings.Split(text, ";") {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}
		fields := strings.Split(record, ",")
		if len(fields) < len(data.Keys) {
			// 末尾常有带空字段的残缺记录，跳过而不报错。
			result.Skipped++
			continue
		}
		result.Bars = append(result.Bars, buildKLineBar(data.Keys, fields))
	}
	return result, nil
}

// buildKLineBar 把一条记录按 keys 映射成 KLineBar。
//
// 已知列映射到具名字段，其余列（均线等）原样保留到 Extra，新增字段不会丢。
func buildKLineBar(keys, fields []string) KLineBar {
	var bar KLineBar
	for i, key := range keys {
		value := strings.TrimSpace(fields[i])
		switch key {
		case "timestamp":
			bar.Timestamp, _ = strconv.ParseInt(value, 10, 64)
		case "time":
			bar.Time = value
		case "open":
			bar.Open = parseFloat(value)
		case "close":
			bar.Close = parseFloat(value)
		case "high":
			bar.High = parseFloat(value)
		case "low":
			bar.Low = parseFloat(value)
		case "volume":
			bar.Volume = parseFloat(value)
		case "amount":
			bar.Amount = parseFloat(value)
		case "range":
			bar.ChangePrice = parseFloat(value)
		case "ratio":
			bar.ChangeRatio = parseFloat(value)
		case "turnoverratio":
			bar.TurnoverRatio = parseOptionalFloat(value)
		case "preClose":
			bar.LastClose = parseOptionalFloat(value)
		default:
			bar.Extra = setExtra(bar.Extra, key, value)
		}
	}
	return bar
}

// setExtra 在 Extra 非空时复用 map，避免每列都判断。
func setExtra(extra map[string]string, key, value string) map[string]string {
	if extra == nil {
		extra = make(map[string]string, 4)
	}
	extra[key] = value
	return extra
}

// parseFloat 解析数值，空值与 `--` 一律按 0 处理。
func parseFloat(value string) float64 {
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return number
}

// parseOptionalFloat 解析可缺省的数值，`--`、空串与非法值返回 nil。
func parseOptionalFloat(value string) *float64 {
	if value == "" || value == "--" {
		return nil
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	return &number
}
