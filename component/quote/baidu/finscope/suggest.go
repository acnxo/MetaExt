package finscope

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// 搜索结果的条目类型（type 字段）。
const (
	// ItemTypeIndex 指数。
	ItemTypeIndex = "index"
	// ItemTypeStock 股票。
	ItemTypeStock = "stock"
	// ItemTypeEtf ETF。
	ItemTypeEtf = "etf"
	// ItemTypeFund 基金。
	ItemTypeFund = "fund"
	// ItemTypeBlock 概念 / 行业板块。
	ItemTypeBlock = "block"
	// ItemTypeBond 债券。
	ItemTypeBond = "bond"
)

// SuggestItem 是搜索接口（`/vapi/v1/sug`）返回的一条候选。
//
// 该接口只做关键词匹配，不要求完整名称；同名的指数、板块、ETF、基金可能并列出现，
// 因此要按 Type 与 Exchange 过滤。它顺带返回条目**当前的行情快照**（Price / Ratio），
// 在行情接口被风控时，可用它取指数的点位与涨跌幅。
type SuggestItem struct {
	// Code 代码。
	Code string `json:"code"`
	// Name 名称。
	Name string `json:"name"`
	// Market 市场编码，如 ab。
	Market string `json:"market"`
	// Type 条目类型，取 ItemTypeIndex / ItemTypeStock / ItemTypeBlock 等。
	Type string `json:"type"`
	// Price 最新价的展示字符串，指数为点位。
	Price string `json:"price"`
	// Ratio 涨跌幅的展示字符串，带百分号，如 "-3.50%"。
	Ratio string `json:"ratio"`
	// Exchange 所属交易所或类别，指数为 BJ / SZ / NQ，板块为 BK，基金为 FD。
	Exchange string `json:"exchange"`
	// SubType 子类型，无值时为 "--"。
	SubType string `json:"subType"`
	// Status 状态标记，接口原值。
	Status string `json:"status"`
}

// PriceValue 把 Price 转成数值；无法解析时返回 false。
func (s SuggestItem) PriceValue() (float64, bool) {
	return parseDisplayFloat(s.Price)
}

// RatioPercent 把 Ratio 转成百分数数值（-3.50% → -3.5）；无法解析时返回 false。
func (s SuggestItem) RatioPercent() (float64, bool) {
	return parseDisplayFloat(strings.TrimSuffix(strings.TrimSpace(s.Ratio), "%"))
}

// parseDisplayFloat 解析展示型字符串，容忍 + 前缀与千分位。
func parseDisplayFloat(value string) (float64, bool) {
	text := strings.TrimSpace(value)
	text = strings.TrimPrefix(text, "+")
	text = strings.ReplaceAll(text, ",", "")
	if text == "" || text == "--" {
		return 0, false
	}
	number, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, false
	}
	return number, true
}

// Suggest 用关键词搜索证券、指数与板块。
//
// 状态：已验证（用真实响应核对过结构）。
//
// 该接口不参与 getquotation 的风控，实测可用；返回结果中同名条目较多（指数、ETF、
// 联接基金常常同名），调用方应按 Type 过滤后再取行情字段。
func (c *Client) Suggest(ctx context.Context, wd string) ([]SuggestItem, error) {
	keyword := strings.TrimSpace(wd)
	if keyword == "" {
		return nil, fmt.Errorf("百度 sug 参数 wd 不能为空")
	}
	path := c.cfg.SuggestPath
	params := []Param{{Name: "wd", Value: keyword}}
	body, err := c.get(ctx, path, params)
	if err != nil {
		return nil, err
	}
	return parseSuggest(path, c.cfg.BaseURL+path+"?"+encodeQuery(params), body)
}

// FindSuggest 搜索并只保留指定类型的条目。
//
// 常用于「已知名称查代码」：传入 ItemTypeIndex 可过滤掉同名的 ETF、基金与板块。
func (c *Client) FindSuggest(ctx context.Context, wd, itemType string) ([]SuggestItem, error) {
	items, err := c.Suggest(ctx, wd)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(itemType) == "" {
		return items, nil
	}
	matched := make([]SuggestItem, 0, len(items))
	for _, item := range items {
		if item.Type == itemType {
			matched = append(matched, item)
		}
	}
	return matched, nil
}

// suggestEnvelope 是搜索响应外层。
type suggestEnvelope struct {
	QueryID    string `json:"QueryID"`
	ResultCode int    `json:"ResultCode"`
	Result     struct {
		List []SuggestItem `json:"list"`
	} `json:"Result"`
}

// parseSuggest 解析搜索响应。
func parseSuggest(path, targetURL string, body []byte) ([]SuggestItem, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, &APIError{Path: path, URL: targetURL, Message: "响应为空"}
	}
	var envelope suggestEnvelope
	if err := json.Unmarshal(trimmed, &envelope); err != nil {
		return nil, &APIError{
			Path: path, URL: targetURL, Body: truncate(string(trimmed), errorBodyLimit),
			Message: "响应解析失败: " + err.Error(),
		}
	}
	// Result 为 null 表示没有匹配项，不是错误。
	return envelope.Result.List, nil
}
