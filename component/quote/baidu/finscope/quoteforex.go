package finscope

import (
	"context"
	"fmt"
	"strings"
)

// GroupForexMinute getquotation 的外汇分钟分组。
const GroupForexMinute = "huilv_minute"

// GetQuoteForexMinuteRequest 是 getquotation（`group=huilv_minute`）的请求参数。
//
// 外汇没有 market / market_type 概念：货币对直接写在 Code 里，URL 中也没有
// srcid、pointType、newFormat。
type GetQuoteForexMinuteRequest struct {
	// Code 货币对，6 位大写字母，形如 USDCNY（美元兑人民币）、BTCUSD（比特币兑美元）。
	Code string
	// NeedReverseReal 固定 1；字面为「需要反向真实值」，具体口径未验证，按原样传递。
	NeedReverseReal int
}

// withDefaults 校验货币对格式并补齐固定参数。
func (r GetQuoteForexMinuteRequest) withDefaults() (GetQuoteForexMinuteRequest, error) {
	r.Code = strings.ToUpper(strings.TrimSpace(r.Code))
	if r.Code == "" {
		return r, fmt.Errorf("百度外汇分钟参数 code 不能为空")
	}
	if len(r.Code) != 6 {
		return r, fmt.Errorf("百度外汇分钟参数 code 应为 6 位货币对，实际 %q", r.Code)
	}
	if r.NeedReverseReal == 0 {
		r.NeedReverseReal = 1
	}
	if r.NeedReverseReal != 1 {
		return r, fmt.Errorf("百度外汇分钟参数 need_reverse_real 只能为 1，实际 %d", r.NeedReverseReal)
	}
	return r, nil
}

// params 按文档中的参数顺序构造查询参数。
func (r GetQuoteForexMinuteRequest) params() []Param {
	return []Param{
		{Name: "group", Value: GroupForexMinute},
		{Name: "need_reverse_real", Value: "1"},
		{Name: "code", Value: r.Code},
		{Name: "finClientType", Value: DefaultClientType},
	}
}

// GetQuoteForexMinute 拉取货币对的分钟行情。
//
// 响应结构**未验证**：本机对该 group 持续返回风控，返回的是 RawQuoteResult，
// 其 Result 为原始 JSON，字段含义待拿到真实样本后确认。不要把股票 K 线的
// newMarketData 结构套到这里。
func (c *Client) GetQuoteForexMinute(ctx context.Context, req GetQuoteForexMinuteRequest) (*RawQuoteResult, error) {
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
