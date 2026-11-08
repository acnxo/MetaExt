package quoteapiv2

import (
	"context"
	"encoding/json"
	"strings"
)

// GetFxRateListRequest 汇率列表请求参数（对应 get-fx-rate-list 接口）。
//
// 四个参数在服务端都有默认值：本实现缺省时显式补默认值（listId=1、page=0、
// pageSize=50），与「缺省整键省略」的接口不同，这里始终把它们送入签名。
type GetFxRateListRequest struct {
	// ListID 清单分组 ID，为空取 DefaultFxListID（全部汇率，观测 282 条）。
	// 取值 1 / 10 / 11 / 12 对应前端的全部、欧美、亚太、其他汇率四组；
	// 2~9 也可用，但前端没有对应路由。传 0 服务端返回 code=3，本方法不做本地拦截。
	ListID string
	// Page 页码，从 0 开始（0 即第 1 页）；为空取 DefaultFxRatePage。越界页返回空列表。
	Page string
	// PageSize 每页数量，为空取 DefaultFxRatePageSize。
	// 服务端上限未逐值定位：观测 300 可用、500 返回 code=500。
	PageSize string
	// TimestampMillis 毫秒时间戳（查询串 _ 参数），为空则省略该参数。
	TimestampMillis string
}

// FxRateListPage 汇率列表结果：分组 ID、分页信息与当前页条目。
type FxRateListPage struct {
	// ListID 服务端回显的清单分组 ID；页码越界时该字段可能缺失，此处为空串。
	ListID string
	// Pagination 分页元信息（page 从 0 开始）。
	Pagination Pagination
	// Items 当前页的汇率条目；越界页为空。
	Items []FxRateListItem
}

// FxRateListItem 汇率列表条目。
//
// 价格、涨跌与振幅一律保留接口原始字符串：小数位随货币对精度变化
// （USDJPY 3 位、USDCNY 4 位、EURUSD 5 位），涨跌幅与振幅是服务端已格式化
// 的百分比文本，转数值会丢失口径。
type FxRateListItem struct {
	// StockID 证券内部 ID，外汇标的集中在 72000000 段。
	StockID int64 `json:"stockId"`
	// StockCode 货币对代码，如 EURUSD；指数类为 USDindex、EURindex。
	StockCode string `json:"stockCode"`
	// MarketLabel 市场标签，外汇固定为 FX。
	MarketLabel string `json:"marketLabel"`
	// InstrumentType 证券类型，外汇固定为 10。
	InstrumentType int `json:"instrumentType"`
	// Name 展示名称，如 EUR/USD；指数类为 USD、Euro。
	Name string `json:"name"`
	// PriceDirect 最新价涨跌方向：up / down / flat。
	PriceDirect string `json:"priceDirect"`
	// PriceNominal 最新价。
	PriceNominal string `json:"priceNominal"`
	// ChangeAmount 涨跌额，带符号。
	ChangeAmount string `json:"changeAmount"`
	// ChangePercent 涨跌幅，含百分号。
	ChangePercent string `json:"changePercent"`
	// PriceBid 买盘价。
	PriceBid string `json:"priceBid"`
	// PriceBidDirect 买盘价方向：up / down / flat。
	PriceBidDirect string `json:"priceBidDirect"`
	// PriceAsk 卖盘价。
	PriceAsk string `json:"priceAsk"`
	// PriceAskDirect 卖盘价方向：up / down / flat。
	PriceAskDirect string `json:"priceAskDirect"`
	// PriceOpen 今开。
	PriceOpen string `json:"priceOpen"`
	// PriceLastClose 昨收。
	PriceLastClose string `json:"priceLastClose"`
	// PriceHighest 今日最高。
	PriceHighest string `json:"priceHighest"`
	// PriceLowest 今日最低。
	PriceLowest string `json:"priceLowest"`
	// AmplitudePrice 振幅，含百分号。
	AmplitudePrice string `json:"amplitudePrice"`
}

// GetFxRateList 按清单分组拉取汇率列表（对应 get-fx-rate-list 接口）。
//
// 页码从 0 开始；同一 listId 下顺序稳定，本方法不做额外排序。
func (c *Client) GetFxRateList(ctx context.Context, req GetFxRateListRequest) (FxRateListPage, error) {
	page, err := intParam(defaultIfEmpty(req.Page, DefaultFxRatePage), "page", 0)
	if err != nil {
		return FxRateListPage{}, err
	}
	pageSize, err := intParam(defaultIfEmpty(req.PageSize, DefaultFxRatePageSize), "pageSize", 1)
	if err != nil {
		return FxRateListPage{}, err
	}

	// 参数顺序即签名键序：listId → page → pageSize → _。
	params := []Param{
		{Name: "listId", Value: defaultIfEmpty(req.ListID, DefaultFxListID)},
		{Name: "page", Value: page},
		{Name: "pageSize", Value: pageSize},
	}
	if timestamp := strings.TrimSpace(req.TimestampMillis); timestamp != "" {
		params = append(params, Param{Name: "_", Value: timestamp})
	}

	path := c.cfg.FxRateListPath
	// 汇率没有 marketType，来源页走全局默认值。
	body, err := c.get(ctx, path, params)
	if err != nil {
		return FxRateListPage{}, err
	}
	var payload fxRateListPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return FxRateListPage{}, &APIError{
			Path: path, URL: c.url(path, params), Body: truncate(string(body), 500),
			Message: "响应解析失败: " + err.Error(),
		}
	}
	if err := decode(body, path, c.url(path, params), payload.Code, payload.Message); err != nil {
		return FxRateListPage{}, err
	}
	return FxRateListPage{
		ListID:     payload.Data.ListID,
		Pagination: payload.Data.Pagination,
		Items:      payload.Data.List,
	}, nil
}

// fxRateListPayload 汇率列表响应外层包裹。
type fxRateListPayload struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    fxRateListData `json:"data"`
}

// fxRateListData 响应 data 字段。
type fxRateListData struct {
	ListID     string           `json:"listId"`
	Pagination Pagination       `json:"pagination"`
	List       []FxRateListItem `json:"list"`
}
