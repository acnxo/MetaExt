package quoteapiv2

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// GetQuoteMinuteRequest 分钟级分时请求参数（对应 get-quote-minute 接口）。
//
// 六个定位参数（StockID / MarketType / Type / MarketCode / InstrumentType /
// SubInstrumentType）全部必填，且必须与 stockId 匹配，否则服务端返回 500。
//
// ReqSection 是唯一使用 snake_case 的可选参数：为空时整键省略，服务端返回全天数据；
// 非空时插在 subInstrumentType 之后、时间戳之前参与签名。
type GetQuoteMinuteRequest struct {
	// StockID 服务端内部股票 ID，必填。
	StockID string
	// MarketType 市场类型，必填；本接口中 1 为 A 股、2 为港股。
	MarketType string
	// Type 行情类型，必填。
	Type string
	// MarketCode 市场代码，必填。
	MarketCode string
	// InstrumentType 证券类型，必填。
	InstrumentType string
	// SubInstrumentType 证券子类型，必填。
	SubInstrumentType string
	// ReqSection 请求时段，观测到 1 表示盘中；为空时省略该参数，返回全天数据。
	ReqSection string
	// TimestampMillis 毫秒时间戳，为空则省略该参数。
	TimestampMillis string
}

// QuoteMinute 分钟级分时数据：当日概览、分钟级明细与交易时段。
type QuoteMinute struct {
	// StockID 证券 ID。
	StockID int64
	// ServerTime 服务端时间戳，毫秒。
	ServerTime int64
	// LastClosePrice 昨收价。
	LastClosePrice float64
	// HighestPrice 最高价。
	HighestPrice float64
	// LowestPrice 最低价。
	LowestPrice float64
	// Ticks 分钟级明细。
	Ticks []QuoteMinuteTick
	// Sections 当日交易时段。
	Sections []QuoteMinuteSection
}

// QuoteMinuteTick 一条分钟级分时数据。
//
// 价格、成交量与成交额按接口原始类型（JSON number）用 float64 承接；涨跌幅是带百分号的
// 字符串，保持原样以免丢失格式。
type QuoteMinuteTick struct {
	// Time 分钟时间戳，秒级 UNIX 时间戳。
	Time int64 `json:"time"`
	// Price 成交价格。
	Price float64 `json:"price"`
	// CCPrice 复权价格。
	CCPrice float64 `json:"cc_price"`
	// Volume 成交量。
	Volume float64 `json:"volume"`
	// Turnover 成交额。
	Turnover float64 `json:"turnover"`
	// Ratio 涨跌幅，含百分号。
	Ratio string `json:"ratio"`
	// ChangePrice 涨跌额。
	ChangePrice float64 `json:"change_price"`
}

// QuoteMinuteSection 一个交易时段。
type QuoteMinuteSection struct {
	// Begin 时段开始时间戳，秒级 UNIX 时间戳。
	Begin int64 `json:"begin"`
	// End 时段结束时间戳，秒级 UNIX 时间戳。
	End int64 `json:"end"`
}

// GetQuoteMinute 获取分钟级分时行情（对应 get-quote-minute 接口）。
//
// ReqSection 为空时返回全天数据；指定时段时只返回该时段。六个定位参数必须与 stockId
// 匹配，本方法只做非空校验，不做语义换算。
func (c *Client) GetQuoteMinute(ctx context.Context, req GetQuoteMinuteRequest) (QuoteMinute, error) {
	required := []struct {
		name  string
		value string
	}{
		{"stockId", req.StockID},
		{"marketType", req.MarketType},
		{"type", req.Type},
		{"marketCode", req.MarketCode},
		{"instrumentType", req.InstrumentType},
		{"subInstrumentType", req.SubInstrumentType},
	}
	trimmed := make([]string, len(required))
	for i, item := range required {
		trimmed[i] = strings.TrimSpace(item.value)
		if trimmed[i] == "" {
			return QuoteMinute{}, errors.New("moomoo GetQuoteMinute：" + item.name + " 不能为空")
		}
	}

	// 参数顺序即签名键序；req_section 是唯一的 snake_case 参数，位置固定在
	// subInstrumentType 之后、时间戳之前。
	params := []Param{
		{Name: "stockId", Value: trimmed[0]},
		{Name: "marketType", Value: trimmed[1]},
		{Name: "type", Value: trimmed[2]},
		{Name: "marketCode", Value: trimmed[3]},
		{Name: "instrumentType", Value: trimmed[4]},
		{Name: "subInstrumentType", Value: trimmed[5]},
	}
	if section := strings.TrimSpace(req.ReqSection); section != "" {
		params = append(params, Param{Name: "req_section", Value: section})
	}
	if timestamp := strings.TrimSpace(req.TimestampMillis); timestamp != "" {
		params = append(params, Param{Name: "_", Value: timestamp})
	}

	path := c.cfg.QuoteMinutePath
	body, err := c.get(ctx, path, params)
	if err != nil {
		return QuoteMinute{}, err
	}
	var payload quoteMinutePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return QuoteMinute{}, &APIError{
			Path: path, URL: c.url(path, params), Body: truncate(string(body), 500),
			Message: "响应解析失败: " + err.Error(),
		}
	}
	if err := decode(body, path, c.url(path, params), payload.Code, payload.Message); err != nil {
		return QuoteMinute{}, err
	}
	return QuoteMinute{
		StockID:        payload.Data.StockID,
		ServerTime:     payload.Data.ServerTime,
		LastClosePrice: payload.Data.LastClosePrice,
		HighestPrice:   payload.Data.HighestPrice,
		LowestPrice:    payload.Data.LowestPrice,
		Ticks:          payload.Data.List,
		Sections:       payload.Data.TimeSection,
	}, nil
}

// quoteMinutePayload 分时行情响应外层包裹。
type quoteMinutePayload struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    quoteMinuteData `json:"data"`
}

// quoteMinuteData 响应 data 字段。
type quoteMinuteData struct {
	StockID        int64                `json:"stockId"`
	ServerTime     int64                `json:"server_time"`
	LastClosePrice float64              `json:"last_close_price"`
	HighestPrice   float64              `json:"highest_price"`
	LowestPrice    float64              `json:"lowest_price"`
	List           []QuoteMinuteTick    `json:"list"`
	TimeSection    []QuoteMinuteSection `json:"time_section"`
}
