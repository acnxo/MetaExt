package quoteapiv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// OptionQuoteID 一个待查询的期权合约。
type OptionQuoteID struct {
	// ID 期权合约 ID，字符串形式，不能为空。
	ID string
	// SpreadTableCode 价差表代码，整数形式；写成字符串会导致签名与服务端不一致。
	SpreadTableCode int
}

// GetOptionQuoteRequest 期权批量报价请求参数（对应 get-option-quote 接口）。
type GetOptionQuoteRequest struct {
	// IDs 待查询的合约列表，至少一个元素。
	IDs []OptionQuoteID
}

// OptionQuote 单个期权合约的实时报价。
//
// 全部字段均为字符串，价格与数量需要调用方自行转换；这与接口原始口径一致，
// 也避免了单位与精度损失。
type OptionQuote struct {
	// PriceBid 买盘价格。
	PriceBid string `json:"priceBid"`
	// VolumeBid 买盘数量。
	VolumeBid string `json:"volumeBid"`
	// PriceAsk 卖盘价格。
	PriceAsk string `json:"priceAsk"`
	// VolumeAsk 卖盘数量。
	VolumeAsk string `json:"volumeAsk"`
	// Price 最新成交价。
	Price string `json:"price"`
	// ChangeRatio 涨跌幅，含百分号。
	ChangeRatio string `json:"changeRatio"`
	// Change 涨跌额。
	Change string `json:"change"`
	// Volume 成交量。
	Volume string `json:"volume"`
	// Turnover 成交额。
	Turnover string `json:"turnover"`
	// PriceDirect 价格方向，如 up、down。
	PriceDirect string `json:"priceDirect"`
}

// GetOptionQuote 批量获取期权合约报价（对应 get-option-quote 接口）。
//
// 返回以合约 ID 为键的报价表。已过期或已摘牌的合约不会报错，对应键不会出现在结果中，
// 因此调用方需要按请求的 ID 逐个检查返回：值为零值 OptionQuote 说明该合约没有报价。
//
// 该接口为 POST，签名输入是请求体的原始字节，所以请求体必须一次构造完成、
// 保持紧凑格式与固定键序，不能在签名后再做格式化。
func (c *Client) GetOptionQuote(ctx context.Context, req GetOptionQuoteRequest) (map[string]OptionQuote, error) {
	body, err := buildOptionQuoteBody(req.IDs)
	if err != nil {
		return nil, err
	}

	path := c.cfg.OptionQuotePath
	respBody, err := c.post(ctx, path, body)
	if err != nil {
		return nil, err
	}
	url := c.cfg.BaseURL + path
	var payload optionQuotePayload
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, &APIError{
			Path: path, URL: url, Body: truncate(string(respBody), 500),
			Message: "响应解析失败: " + err.Error(),
		}
	}
	if err := decode(respBody, path, url, payload.Code, payload.Message); err != nil {
		return nil, err
	}
	if payload.Data == nil {
		// 接口正常时返回对象，为空对象表示没有有效报价；null 也统一成空表。
		return map[string]OptionQuote{}, nil
	}
	return payload.Data, nil
}

// buildOptionQuoteBody 构造请求体：紧凑 JSON，元素键序固定为 id、spreadTableCode。
//
// 这里手工拼装而不用 encoding/json：请求体字节即签名输入，标准库的转义与字段处理
// 一旦发生变化就会导致签名不一致。ID 走与签名相同的转义函数，其余部分是数字。
func buildOptionQuoteBody(ids []OptionQuoteID) ([]byte, error) {
	if len(ids) == 0 {
		return nil, errors.New("moomoo GetOptionQuote：ids 至少要有一个元素")
	}
	var b strings.Builder
	b.WriteString(`{"ids":[`)
	for i, item := range ids {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			return nil, fmt.Errorf("moomoo GetOptionQuote：第 %d 个合约的 id 不能为空", i+1)
		}
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"id":"`)
		b.WriteString(escapeJSONString(id))
		b.WriteString(`","spreadTableCode":`)
		b.WriteString(strconv.Itoa(item.SpreadTableCode))
		b.WriteString(`}`)
	}
	b.WriteString(`]}`)
	return []byte(b.String()), nil
}

// optionQuotePayload 期权报价响应外层包裹。
type optionQuotePayload struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]OptionQuote `json:"data"`
}
