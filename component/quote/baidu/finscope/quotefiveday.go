package finscope

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// GetQuoteFiveDayRequest 是 getquotation（`group=quotation_fiveday_ab`）的请求参数。
//
// 与 K 线接口同路径、同参数集，但没有 ktype / count：周期固定为分钟，
// 窗口固定为近五个交易日。
type GetQuoteFiveDayRequest struct {
	// Code 证券代码：A 股 / 指数 6 位数字，港股 5 位，美股字母代码。
	Code string
	// MarketType 市场：MarketA / MarketHK / MarketUS，为空取 MarketA。
	MarketType string
	// Name 证券简称，可空。
	Name string
	// IsKC 注册制板块标记：科创板、创业板、北交所为 1，其它为 0。
	IsKC int
}

// withDefaults 补齐可省略字段并做必填校验。
func (r GetQuoteFiveDayRequest) withDefaults() (GetQuoteFiveDayRequest, error) {
	r.Code = strings.TrimSpace(r.Code)
	r.MarketType = strings.TrimSpace(r.MarketType)
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
	if r.IsKC != 0 && r.IsKC != 1 {
		return r, fmt.Errorf("百度 getquotation 参数 is_kc 只能是 0 或 1，实际 %d", r.IsKC)
	}
	return r, nil
}

// params 按文档中的参数顺序构造查询参数。
func (r GetQuoteFiveDayRequest) params() []Param {
	params := []Param{
		{Name: "srcid", Value: DefaultSrcID},
		{Name: "pointType", Value: DefaultPointType},
		{Name: "group", Value: GroupFiveDay},
		{Name: "newFormat", Value: "1"},
		{Name: "finClientType", Value: DefaultClientType},
		{Name: "code", Value: r.Code},
		{Name: "query", Value: r.Code},
		{Name: "market_type", Value: r.MarketType},
	}
	if r.Name != "" {
		params = append(params, Param{Name: "name", Value: r.Name})
	}
	params = append(params, Param{Name: "is_kc", Value: strconv.Itoa(r.IsKC)})
	return params
}

// FiveDayPoint 是五日分时里的一个分钟点。
type FiveDayPoint struct {
	// Timestamp 秒级时间戳，接口原值。
	Timestamp int64
	// Time 接口原值，形如 "11-09 09:31"，**不含年份**。
	Time string
	// Price 该分钟价格。
	Price float64
	// ChangePrice 涨跌额。
	ChangePrice float64
	// ChangeRatio 涨跌幅（百分数）。
	ChangeRatio float64
	// Volume 成交量。
	Volume float64
	// Amount 成交额。
	Amount float64
}

// FiveDayItem 是一个交易日的分时点集合。
type FiveDayItem struct {
	// Date 交易日，形如 "2026-11-09"。
	Date string
	// Points 该日的分钟点，按接口返回顺序。
	Points []FiveDayPoint
	// PreClose 昨收，实测响应未携带，故可能为 nil。
	PreClose *float64
}

// FiveDayResult 是五日分时的解析结果。
type FiveDayResult struct {
	// Headers 中文列名。
	Headers []string
	// Keys 字段名，决定 `p` 每条的列顺序。
	Keys []string
	// Days 按交易日分组的分钟点。
	Days []FiveDayItem
	// Skipped 因字段数不足被跳过的残缺记录数。
	Skipped int
	// MaxPoints 单日最大点数（接口原值，字符串）。
	MaxPoints string
	// CX 横轴日期标签，如 ["11-09", "11-10"]。
	CX []string
}

// GetQuoteFiveDay 拉取近五个交易日的分钟分时。
//
// 返回的分钟点按交易日分组。`time` 列不带年份，年份取自同组的 `date`；
// 需要完整时间时把两者拼接，或直接用 Timestamp。
func (c *Client) GetQuoteFiveDay(ctx context.Context, req GetQuoteFiveDayRequest) (*FiveDayResult, error) {
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
	return parseFiveDay(path, c.cfg.BaseURL+path+"?"+encodeQuery(params), body)
}

// fiveDayData 是五日分时的 newMarketData。
type fiveDayData struct {
	Headers    []string `json:"headers"`
	Keys       []string `json:"keys"`
	MarketData []struct {
		Date     string          `json:"date"`
		P        json.RawMessage `json:"p"`
		PreClose *float64        `json:"preClose"`
	} `json:"marketData"`
	MaxPoints string   `json:"maxPoints"`
	CX        []string `json:"cx"`
}

// parseFiveDay 解析五日分时响应。
func parseFiveDay(path, targetURL string, body []byte) (*FiveDayResult, error) {
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
		NewMarketData fiveDayData `json:"newMarketData"`
	}
	if err := json.Unmarshal(resultRaw, &wrapper); err != nil {
		return nil, &APIError{
			Path: path, URL: targetURL, Body: truncate(string(body), errorBodyLimit),
			Message: "响应 Result 解析失败: " + err.Error(),
		}
	}
	return decodeFiveDay(wrapper.NewMarketData)
}

// decodeFiveDay 按 keys 把每日的 `p` 拆成分钟点。
func decodeFiveDay(data fiveDayData) (*FiveDayResult, error) {
	result := &FiveDayResult{
		Headers:   data.Headers,
		Keys:      data.Keys,
		MaxPoints: data.MaxPoints,
		CX:        data.CX,
	}
	if len(data.MarketData) == 0 {
		return result, nil
	}
	if len(data.Keys) == 0 {
		return nil, fmt.Errorf("百度五日分时响应缺少 keys，无法确定列顺序")
	}
	for _, day := range data.MarketData {
		item := FiveDayItem{Date: day.Date, PreClose: day.PreClose}
		text, err := fiveDayPointsText(day.P)
		if err != nil {
			return nil, err
		}
		for _, record := range strings.Split(text, ";") {
			record = strings.TrimSpace(record)
			if record == "" {
				continue
			}
			fields := strings.Split(record, ",")
			if len(fields) < len(data.Keys) {
				result.Skipped++
				continue
			}
			item.Points = append(item.Points, buildFiveDayPoint(data.Keys, fields))
		}
		result.Days = append(result.Days, item)
	}
	return result, nil
}

// fiveDayPointsText 取出每日 `p` 的字符串形态。
//
// 实测为字符串；旧数据可能是数组，本实现对数组形态返回未支持的错误，
// 而不是静默丢弃——宁可报错，也不要给出不完整的分时。
func fiveDayPointsText(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return "", nil
	}
	if trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return "", fmt.Errorf("百度五日分时的 p 字段解析失败: %w", err)
		}
		return text, nil
	}
	return "", fmt.Errorf("百度五日分时的 p 字段不是字符串（实测形态为字符串），当前实现不支持该形态")
}

// buildFiveDayPoint 按 keys 把一条记录映射成分钟点。
func buildFiveDayPoint(keys, fields []string) FiveDayPoint {
	var point FiveDayPoint
	for i, key := range keys {
		value := strings.TrimSpace(fields[i])
		switch key {
		case "timestamp":
			point.Timestamp, _ = strconv.ParseInt(value, 10, 64)
		case "time":
			point.Time = value
		case "price":
			point.Price = parseFloat(value)
		case "range":
			point.ChangePrice = parseFloat(value)
		case "ratio":
			point.ChangeRatio = parseFloat(value)
		case "volume":
			point.Volume = parseFloat(value)
		case "amount":
			point.Amount = parseFloat(value)
		}
	}
	return point
}
