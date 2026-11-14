package finscope

import (
	"context"
	"fmt"
	"strings"
)

// getquotation 的期货分组与期货五日路径。
const (
	// GroupFuturesMinute getquotation 的期货分钟分组。
	GroupFuturesMinute = "quotation_futures_minute"
	// GroupFuturesFiveDay 期货五日分时的分组名。
	GroupFuturesFiveDay = "quotation_futures_fiveday"
)

// GetQuoteFuturesMinuteRequest 是 getquotation（`group=quotation_futures_minute`）的请求参数。
//
// 与股票 K 线的差异：多出 All、EProp、FinanceType 三个期货专用参数；
// 同一个 `market_type=ab` 在期货语义下指国内商品期货，而不是 A 股。
type GetQuoteFuturesMinuteRequest struct {
	// Code 期货合约代码：国内为「品种字母 + 合约数字」（AU888 主力连续、AU2612），
	// 外盘为「品种代码 + 两位年份 + 月份码」（GC26Z）。
	Code string
	// MarketType 市场：MarketA（国内商品期货）、MarketUS（美国市场期货），为空取 MarketA。
	MarketType string
}

// withDefaults 做必填校验并补齐默认市场。
func (r GetQuoteFuturesMinuteRequest) withDefaults() (GetQuoteFuturesMinuteRequest, error) {
	r.Code = strings.ToUpper(strings.TrimSpace(r.Code))
	r.MarketType = strings.TrimSpace(r.MarketType)
	if r.Code == "" {
		return r, fmt.Errorf("百度期货分钟参数 code 不能为空")
	}
	switch r.MarketType {
	case "":
		r.MarketType = MarketA
	case MarketA, MarketUS:
	default:
		return r, fmt.Errorf("百度期货分钟参数 market_type 只能为 ab 或 us，实际 %q", r.MarketType)
	}
	return r, nil
}

// params 按文档中的参数顺序构造查询参数。
func (r GetQuoteFuturesMinuteRequest) params() []Param {
	return []Param{
		{Name: "srcid", Value: DefaultSrcID},
		{Name: "all", Value: "1"},
		{Name: "code", Value: r.Code},
		{Name: "query", Value: r.Code},
		{Name: "eprop", Value: "min"},
		{Name: "financeType", Value: FinanceTypeFutures},
		{Name: "group", Value: GroupFuturesMinute},
		{Name: "market_type", Value: r.MarketType},
		{Name: "finClientType", Value: DefaultClientType},
	}
}

// GetQuoteFuturesMinute 拉取期货合约的分钟行情。
//
// 响应结构**未验证**：本机对该 group 持续风控，Result 是否为 newMarketData 结构、
// keys 是否含持仓量与结算价等期货特有字段，都需真实样本确认，故返回 RawQuoteResult。
func (c *Client) GetQuoteFuturesMinute(ctx context.Context, req GetQuoteFuturesMinuteRequest) (*RawQuoteResult, error) {
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
	return parseRawQuote(path, c.cfg.BaseURL+path+"?"+encodeQuery(params), body)
}

// GetQuoteFuturesFiveDayRequest 是期货五日分时接口的请求参数。
//
// 该接口路径与股票侧不同（`/selfselect/getstockquotation`），品种靠 5 个 is* 开关指定，
// **必须且只能** IsFutures=true，其余四个为 false；市场仍由 MarketType 指定。
type GetQuoteFuturesFiveDayRequest struct {
	// Code 期货合约代码。
	Code string
	// MarketType 市场：MarketA（国内商品期货）、MarketUS（美国市场期货），为空取 MarketA。
	MarketType string
}

// withDefaults 做必填校验并补齐默认市场。
func (r GetQuoteFuturesFiveDayRequest) withDefaults() (GetQuoteFuturesFiveDayRequest, error) {
	r.Code = strings.ToUpper(strings.TrimSpace(r.Code))
	r.MarketType = strings.TrimSpace(r.MarketType)
	if r.Code == "" {
		return r, fmt.Errorf("百度期货五日参数 code 不能为空")
	}
	switch r.MarketType {
	case "":
		r.MarketType = MarketA
	case MarketA, MarketUS:
	default:
		return r, fmt.Errorf("百度期货五日参数 market_type 只能为 ab 或 us，实际 %q", r.MarketType)
	}
	return r, nil
}

// params 按文档中的参数顺序构造查询参数。
//
// 5 个 is* 开关必须且只能 isFutures=true，其余为 false。
func (r GetQuoteFuturesFiveDayRequest) params() []Param {
	return []Param{
		{Name: "all", Value: "1"},
		{Name: "code", Value: r.Code},
		{Name: "isIndex", Value: "false"},
		{Name: "isBk", Value: "false"},
		{Name: "isBlock", Value: "false"},
		{Name: "isFutures", Value: "true"},
		{Name: "isStock", Value: "false"},
		{Name: "newFormat", Value: "1"},
		{Name: "market_type", Value: r.MarketType},
		{Name: "group", Value: GroupFuturesFiveDay},
		{Name: "finClientType", Value: DefaultClientType},
	}
}

// GetQuoteFuturesFiveDay 拉取期货合约近五个交易日的分钟分时。
//
// 响应结构**未验证**：本机请求返回风控，且该接口风控时 QueryID 可能为 "0"。
// 它与股票五日是否同为 newMarketData 结构、是否包含夜盘时段，都需真实样本确认，
// 因此返回 RawQuoteResult，不套用股票五日的字段表。
func (c *Client) GetQuoteFuturesFiveDay(ctx context.Context, req GetQuoteFuturesFiveDayRequest) (*RawQuoteResult, error) {
	normalized, err := req.withDefaults()
	if err != nil {
		return nil, err
	}
	path := c.cfg.FuturesFiveDayPath
	params := normalized.params()
	body, err := c.get(ctx, path, params)
	if err != nil {
		return nil, err
	}
	return parseRawQuote(path, c.cfg.BaseURL+path+"?"+encodeQuery(params), body)
}
