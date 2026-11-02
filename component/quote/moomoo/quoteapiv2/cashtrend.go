package quoteapiv2

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// GetHistoryCashTrendRequest 历史资金流水请求参数（对应 get-history-cash-trend 接口）。
//
// Type 是必填项，且属于「非校验型参数」：省略它时服务端既不报错也不返回数据，
// 只会给出空列表，因此实现会在本地直接拒绝空的 Type，避免静默失败。
type GetHistoryCashTrendRequest struct {
	// StockID 服务端内部股票 ID，必填。
	StockID string
	// Type 资金流粒度，必填，取 CashTrendTypeDay / CashTrendTypeWeek / CashTrendTypeMonth。
	Type string
	// BaseTime 分页基准时间戳（秒），为空取 DefaultCashTrendBaseTime（0 表示最新一段）。
	// 向前翻页时传上一页返回记录中的 time。
	BaseTime string
	// TimestampMillis 毫秒时间戳，为空则省略该参数。
	TimestampMillis string
}

// CashTrendPage 资金流水结果：是否还有更早的数据与当前段记录。
type CashTrendPage struct {
	// HasMore 是否还有更早一段数据。
	HasMore bool
	// Items 资金流水记录。
	Items []CashTrendItem
}

// CashTrendItem 一段资金流水记录。
//
// 字段全部为字符串：金额是最小货币单位的整数字符串，本身不含小数位信息；
// 比率字段是百分数乘以 100 后的结果，展示时需要除以 100（如 "117" 表示 +1.17%）。
type CashTrendItem struct {
	// Time 秒级时间戳，字符串形式；翻页时作为下次的 baseTime。
	Time string `json:"time"`
	// TotalNetIn 总净流入。
	TotalNetIn string `json:"totalNetIn"`
	// SuperNetIn 特大单净流入。
	SuperNetIn string `json:"superNetIn"`
	// BigNetIn 大单净流入。
	BigNetIn string `json:"bigNetIn"`
	// MidNetIn 中单净流入。
	MidNetIn string `json:"midNetIn"`
	// SmallNetIn 小单净流入。
	SmallNetIn string `json:"smlNetIn"`
	// MainNetIn 主力净流入，即特大单加大单。
	MainNetIn string `json:"mainNetIn"`
	// PriceChangeRatio 涨跌幅，百分数乘以 100，如 "117" 表示 +1.17%。
	PriceChangeRatio string `json:"priceChangeRatio"`
	// ClosePrice 该期收盘价。
	ClosePrice string `json:"closePrice"`
	// MainDealRatio 主力成交占比，百分数乘以 100，如 "2897" 表示 28.97%。
	MainDealRatio string `json:"mainDealRatio"`
	// AccMainNetIn 累计主力净流入。
	AccMainNetIn string `json:"accMainNetIn"`
}

// GetHistoryCashTrend 获取个股历史资金流水（对应 get-history-cash-trend 接口）。
//
// 支持日、周、月三档粒度，以 baseTime 作为游标向前翻页。空列表是正常响应：
// 标的没有数据、粒度取值无数据、stockId 无效都会返回 code=0 + 空列表，因此不能凭空列表
// 判断参数是否正确，需要先用日粒度取一次非空数据作对照。
func (c *Client) GetHistoryCashTrend(ctx context.Context, req GetHistoryCashTrendRequest) (CashTrendPage, error) {
	stockID := strings.TrimSpace(req.StockID)
	if stockID == "" {
		return CashTrendPage{}, errors.New("moomoo GetHistoryCashTrend：stockId 不能为空")
	}
	trendType := strings.TrimSpace(req.Type)
	if trendType == "" {
		// 省略 type 时服务端静默返回空列表，本地必须先拦住。
		return CashTrendPage{}, errors.New("moomoo GetHistoryCashTrend：type 不能为空，省略会导致服务端静默返回空列表")
	}
	baseTime := defaultIfEmpty(req.BaseTime, DefaultCashTrendBaseTime)

	// 参数顺序即签名键序。
	params := []Param{
		{Name: "stockId", Value: stockID},
		{Name: "type", Value: trendType},
		{Name: "baseTime", Value: baseTime},
	}
	if timestamp := strings.TrimSpace(req.TimestampMillis); timestamp != "" {
		params = append(params, Param{Name: "_", Value: timestamp})
	}

	path := c.cfg.HistoryCashTrendPath
	body, err := c.get(ctx, path, params)
	if err != nil {
		return CashTrendPage{}, err
	}
	var payload cashTrendPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return CashTrendPage{}, &APIError{
			Path: path, URL: c.url(path, params), Body: truncate(string(body), 500),
			Message: "响应解析失败: " + err.Error(),
		}
	}
	if err := decode(body, path, c.url(path, params), payload.Code, payload.Message); err != nil {
		return CashTrendPage{}, err
	}
	return CashTrendPage{
		HasMore: payload.Data.HasMore,
		Items:   payload.Data.List,
	}, nil
}

// cashTrendPayload 资金流水响应外层包裹。
type cashTrendPayload struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    cashTrendData `json:"data"`
}

// cashTrendData 响应 data 字段。
type cashTrendData struct {
	HasMore bool            `json:"hasMore"`
	List    []CashTrendItem `json:"list"`
}
