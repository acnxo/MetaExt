package finscope

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// 成分类型（参数 financeType）。
const (
	// FinanceTypeIndex 指数成分股。
	FinanceTypeIndex = "index"
	// FinanceTypeBlock 概念 / 行业板块成分股。
	FinanceTypeBlock = "block"
)

// 市场编码（参数 market）。
const (
	// MarketA A 股（沪 / 深 / 北）。
	MarketA = "ab"
	// MarketHK 港股。
	MarketHK = "hk"
	// MarketUS 美股。
	MarketUS = "us"
)

// 排序字段（参数 sortKey）。
const (
	// SortByMarketValue 按总市值排序。
	SortByMarketValue = "marketValue"
	// SortByAmount 按成交额排序。
	SortByAmount = "amount"
)

// 排序方向（参数 sortType）。
const (
	// SortDesc 降序。
	SortDesc = "desc"
	// SortAsc 升序。
	SortAsc = "asc"
)

// maxCollectPages 是 GetAllConstituents 的翻页上限，防止服务端忽略偏移量时死循环。
const maxCollectPages = 200

// GetConstituentsRequest 是 constituents 接口的请求参数。
//
// 全部参数都会写进查询串。`Offset` / `Limit` 对应文档里的 `pn` / `rn`：
// **`pn` 是偏移量而非页码**，翻页必须 `pn += rn`，按页码自增会严重漏采，
// 因此本包用 Offset 命名并只接受偏移量语义。
type GetConstituentsRequest struct {
	// FinanceType 成分类型，必填，取 FinanceTypeIndex 或 FinanceTypeBlock。
	FinanceType string
	// Market 市场编码，必填，取 MarketA / MarketHK / MarketUS；目前只验证了 A 股。
	Market string
	// Code 来源标识码，必填：指数为 6 位代码（000300），板块为 GNxxxx（GN2309）。
	Code string
	// SortKey 排序字段，必填，取 SortByMarketValue 或 SortByAmount。
	SortKey string
	// SortType 排序方向，必填，取 SortDesc 或 SortAsc。
	SortType string
	// Style 返回样式，为空取 DefaultStyle（tablelist）。
	Style string
	// Offset 偏移量，首屏为 0，下一页为上一次的 Offset 加 Limit；负数按 0 处理。
	Offset int
	// Limit 单页条数，为空或 0 取 DefaultPageSize；实测没有 500 硬上限。
	Limit int
	// ClientType 客户端类型，为空取 DefaultClientType（pc）。
	ClientType string
}

// withDefaults 补齐可省略的字段并做必填校验。
func (r GetConstituentsRequest) withDefaults() (GetConstituentsRequest, error) {
	r.FinanceType = strings.TrimSpace(r.FinanceType)
	r.Market = strings.TrimSpace(r.Market)
	r.Code = strings.TrimSpace(r.Code)
	r.SortKey = strings.TrimSpace(r.SortKey)
	r.SortType = strings.TrimSpace(r.SortType)
	r.Style = strings.TrimSpace(r.Style)
	r.ClientType = strings.TrimSpace(r.ClientType)

	for name, value := range map[string]string{
		"financeType": r.FinanceType,
		"market":      r.Market,
		"code":        r.Code,
		"sortKey":     r.SortKey,
		"sortType":    r.SortType,
	} {
		if value == "" {
			return r, fmt.Errorf("百度 constituents 参数 %s 不能为空", name)
		}
	}
	if r.FinanceType != FinanceTypeIndex && r.FinanceType != FinanceTypeBlock {
		return r, fmt.Errorf("百度 constituents 参数 financeType 取值非法：%q", r.FinanceType)
	}
	switch r.Market {
	case MarketA, MarketHK, MarketUS:
	default:
		return r, fmt.Errorf("百度 constituents 参数 market 取值非法：%q", r.Market)
	}
	if r.SortKey != SortByMarketValue && r.SortKey != SortByAmount {
		return r, fmt.Errorf("百度 constituents 参数 sortKey 取值非法：%q", r.SortKey)
	}
	if r.SortType != SortDesc && r.SortType != SortAsc {
		return r, fmt.Errorf("百度 constituents 参数 sortType 取值非法：%q", r.SortType)
	}
	if r.Style == "" {
		r.Style = DefaultStyle
	}
	if r.ClientType == "" {
		r.ClientType = DefaultClientType
	}
	if r.Offset < 0 {
		r.Offset = 0
	}
	if r.Limit <= 0 {
		r.Limit = DefaultPageSize
	}
	return r, nil
}

// params 按文档中的参数顺序构造查询参数。
func (r GetConstituentsRequest) params() []Param {
	return []Param{
		{Name: "financeType", Value: r.FinanceType},
		{Name: "market", Value: r.Market},
		{Name: "code", Value: r.Code},
		{Name: "sortKey", Value: r.SortKey},
		{Name: "sortType", Value: r.SortType},
		{Name: "style", Value: r.Style},
		{Name: "pn", Value: fmt.Sprintf("%d", r.Offset)},
		{Name: "rn", Value: fmt.Sprintf("%d", r.Limit)},
		{Name: "finClientType", Value: r.ClientType},
	}
}

// ConstituentsHeader 是响应里的列定义，仅用于还原表格列语义。
type ConstituentsHeader struct {
	// Key 字段名，对应条目里的 JSON 字段。
	Key string `json:"key"`
	// Name 中文列名。
	Name string `json:"name"`
	// CanSort 是否可排序，1 / 0。
	CanSort int `json:"canSort"`
}

// ConstituentRawData 是条目的原始数值。
//
// 顶层的同名字段是带单位的展示字符串（`15.42亿`、`1.33%`），本结构才是可直接参与计算的
// 数值，未做单位换算（`15.42亿` 对应 1542319057）。取数值请用本结构。
type ConstituentRawData struct {
	// LastPrice 最新价（元），接口字段名为 lastPx。
	LastPrice float64 `json:"lastPx"`
	// ChangePrice 涨跌额（元），可为负，接口字段名为 pxChange。
	ChangePrice float64 `json:"pxChange"`
	// ChangeRatio 涨跌幅（百分数，0.83 表示 +0.83%），接口字段名为 pxChangeRate。
	ChangeRatio float64 `json:"pxChangeRate"`
	// Volume 成交量（手）。
	Volume float64 `json:"volume"`
	// Amount 成交额（元）。
	Amount float64 `json:"amount"`
	// TurnoverRatio 换手率（百分数）。
	TurnoverRatio float64 `json:"turnoverRatio"`
	// MarketValue 总市值（元）。
	MarketValue float64 `json:"marketValue"`
}

// ConstituentBlock 是条目所属的行业板块。
type ConstituentBlock struct {
	// Code 板块代码。
	Code string `json:"code"`
	// Name 板块名称。
	Name string `json:"name"`
	// Market 市场编码。
	Market string `json:"market"`
	// FinanceType 品种类型，板块通常为 block。
	FinanceType string `json:"financeType"`
}

// ConstituentLogo 是条目 Logo 信息，采集场景可忽略。
type ConstituentLogo struct {
	// Type Logo 类型，如 normal。
	Type string `json:"type"`
	// Logo Logo 图地址。
	Logo string `json:"logo"`
}

// Constituent 是成分股条目。
//
// 同一指标有两份：带单位的展示字符串（本结构的中文名短字段）与 RawData 中的原始数值。
// 计算、入库一律用 RawData；展示字符串可能带「万」「亿」「%」，直接解析会丢口径。
type Constituent struct {
	// Code 证券代码，去重主键。
	Code string `json:"code"`
	// Name 证券简称。
	Name string `json:"name"`
	// Exchange 交易所，SH / SZ / BJ。
	Exchange string `json:"exchange"`
	// Market 市场标识，取值同请求参数 market。
	Market string `json:"market"`
	// FinanceType 品种类型；本接口对股票与 ETF 都返回 stock。
	FinanceType string `json:"financeType"`
	// HasQuote 是否有行情，1 / 0。
	HasQuote int `json:"hasQuote"`
	// LastPrice 最新价的展示字符串，接口字段名为 lastPx。
	LastPrice string `json:"lastPx"`
	// ChangePrice 涨跌额的展示字符串，接口字段名为 pxChange。
	ChangePrice string `json:"pxChange"`
	// ChangeRatio 涨跌幅的展示字符串，接口字段名为 pxChangeRate。
	ChangeRatio string `json:"pxChangeRate"`
	// Volume 成交量展示字符串，带中文单位。
	Volume string `json:"volume"`
	// Amount 成交额展示字符串，带中文单位。
	Amount string `json:"amount"`
	// TurnoverRatio 换手率展示字符串，带百分号。
	TurnoverRatio string `json:"turnoverRatio"`
	// MarketValue 总市值展示字符串，带中文单位。
	MarketValue string `json:"marketValue"`
	// RawData 原始数值，取数值一律用这里。
	RawData ConstituentRawData `json:"rawData"`
	// Block 所属行业板块。
	Block ConstituentBlock `json:"block"`
	// Logo Logo 信息。
	Logo ConstituentLogo `json:"logo"`
	// PCURL 个股页面地址，形如 https://finance.baidu.com/stock/ab-601939。
	PCURL string `json:"pcUrl"`
}

// ConstituentsPage 是 constituents 的单页结果。
type ConstituentsPage struct {
	// Headers 列定义，可据此还原表格列语义。
	Headers []ConstituentsHeader
	// Items 当前页条目，可能为空（偏移量越界）。
	Items []Constituent
	// PCURL 该来源的 PC 页面地址，采集可忽略。
	PCURL string
	// Offset 本次请求的偏移量。
	Offset int
	// Limit 本次请求的单页条数。
	Limit int
	// HasMore 是否可能还有下一页：返回条数等于 Limit 时为 true。
	//
	// 该判断是保守估计——服务端没有游标，只能以「条数小于 Limit」判定末页。
	HasMore bool
}

// GetConstituents 拉取一页成分股。
//
// 偏移量越界时服务端返回空列表（`Result.list` 为 `{}` 或 `[]`），本方法如实返回
// 空 Items 与 HasMore=false，不视为错误。
//
// 风控命中时外层 ResultCode 仍为 0，本方法会返回 *APIError（Code=403），
// 可用 IsRiskControl 判断；不要把它当成「本页没有数据」继续翻页。
func (c *Client) GetConstituents(ctx context.Context, req GetConstituentsRequest) (*ConstituentsPage, error) {
	normalized, err := req.withDefaults()
	if err != nil {
		return nil, err
	}
	path := c.cfg.ConstituentsPath
	params := normalized.params()
	body, err := c.get(ctx, path, params)
	if err != nil {
		return nil, err
	}
	page, err := parseConstituents(path, c.cfg.BaseURL+path+"?"+encodeQuery(params), body)
	if err != nil {
		return nil, err
	}
	page.Offset = normalized.Offset
	page.Limit = normalized.Limit
	page.HasMore = len(page.Items) >= normalized.Limit
	return page, nil
}

// ConstituentsResult 是 GetAllConstituents 的汇总结果。
type ConstituentsResult struct {
	// Headers 列定义，取首次成功响应的值。
	Headers []ConstituentsHeader
	// Items 按 code 去重后的全部条目，保持服务端返回顺序。
	Items []Constituent
	// Pages 实际请求的页数。
	Pages int
}

// GetAllConstituents 按偏移量翻页，直到取完或触达上限。
//
// 翻页规则与站点一致：`Offset += Limit`，返回条数小于 Limit 时结束。
// 另外加了两道保护：按 code 去重；某页没有任何新条目时立即结束（服务端忽略偏移量时
// 不会死循环）。ctx 取消会中断翻页并返回已取到的结果与错误。
//
// 风控命中时立即返回错误，不把风控响应当成空页继续翻页。
func (c *Client) GetAllConstituents(ctx context.Context, req GetConstituentsRequest) (*ConstituentsResult, error) {
	base, err := req.withDefaults()
	if err != nil {
		return nil, err
	}
	result := &ConstituentsResult{}
	seen := make(map[string]struct{}, base.Limit)
	offset := base.Offset
	for page := 0; page < maxCollectPages; page++ {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		current := base
		current.Offset = offset
		got, err := c.GetConstituents(ctx, current)
		if err != nil {
			return result, err
		}
		result.Pages++
		if len(result.Headers) == 0 && len(got.Headers) > 0 {
			result.Headers = got.Headers
		}
		added := 0
		for _, item := range got.Items {
			if _, ok := seen[item.Code]; ok {
				continue
			}
			seen[item.Code] = struct{}{}
			result.Items = append(result.Items, item)
			added++
		}
		if !got.HasMore || added == 0 {
			return result, nil
		}
		offset += base.Limit
	}
	return result, fmt.Errorf("百度 constituents 翻页超过上限 %d 页，已返回部分结果", maxCollectPages)
}

// constituentsEnvelope 是响应外层：业务成败看 Result 一层，ResultCode 只表示网关处理情况。
type constituentsEnvelope struct {
	QueryID    string          `json:"QueryID"`
	ResultCode int             `json:"ResultCode"`
	Result     json.RawMessage `json:"Result"`
}

// constituentsResult 是正常响应的 Result。
type constituentsResult struct {
	List         json.RawMessage `json:"list"`
	PCURL        string          `json:"pcUrl"`
	WebStatusURL string          `json:"webStatusUrl"`
}

// constituentsList 是正常响应 Result.list 的对象形态。
type constituentsList struct {
	Headers []ConstituentsHeader `json:"headers"`
	Body    []Constituent        `json:"body"`
}

// riskControl 是风控响应形态：外层 ResultCode 为 0，错误码在 Result 层。
type riskControl struct {
	Code             int    `json:"code"`
	Message          string `json:"msg"`
	IsCaptchaEnabled bool   `json:"isCaptchaEnabled"`
}

// parseConstituents 解析响应体；风控与结构异常转成 *APIError。
func parseConstituents(path, targetURL string, body []byte) (*ConstituentsPage, error) {
	var envelope constituentsEnvelope
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
	// Result 只能是对象；被降级成数组或标量同样视为失败。
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

	var result constituentsResult
	if err := json.Unmarshal(resultRaw, &result); err != nil {
		return nil, &APIError{
			Path: path, URL: targetURL, Body: truncate(string(body), errorBodyLimit),
			Message: "响应 Result 解析失败: " + err.Error(),
		}
	}
	page := &ConstituentsPage{PCURL: result.PCURL}
	listRaw := bytes.TrimSpace(result.List)
	if len(listRaw) == 0 || string(listRaw) == "null" {
		return page, nil
	}
	// 偏移量越界时 list 可能是空数组，视为没有更多数据。
	if listRaw[0] != '{' {
		return page, nil
	}
	var list constituentsList
	if err := json.Unmarshal(listRaw, &list); err != nil {
		return nil, &APIError{
			Path: path, URL: targetURL, Body: truncate(string(body), errorBodyLimit),
			Message: "响应 Result.list 解析失败: " + err.Error(),
		}
	}
	page.Headers = list.Headers
	page.Items = list.Body
	return page, nil
}
