package quoteapiv2

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// GetStockListRequest 板块股票列表请求参数（对应 get-stock-list 接口）。
//
// 可选参数为空时整键省略，不要用空串占位：rankType、page、pageSize 与时间戳缺省时，
// 签名输入与查询串中都不出现该键。plateType 是例外，缺省时按 DefaultPlateType 补值。
type GetStockListRequest struct {
	// MarketType 市场类型，必填，原样透传。
	MarketType string
	// PlateType 板块类型，为空取 DefaultPlateType。
	PlateType string
	// RankType 排行类型，为空则省略该参数，由服务端决定默认排序。
	RankType string
	// Page 页码，从 0 开始（0 即第一页）；为空则省略，服务端按第 0 页处理。
	Page string
	// PageSize 每页数量；为空则省略，服务端使用默认值（观测为 50）。
	PageSize string
	// TimestampMillis 毫秒时间戳（查询串 _ 参数），为空则省略该参数。
	TimestampMillis string
}

// Pagination 分页元信息。页码从 0 开始，与本接口一致。
type Pagination struct {
	// Page 当前页码。
	Page int `json:"page"`
	// PageSize 每页数量。
	PageSize int `json:"pageSize"`
	// PageCount 总页数。
	PageCount int `json:"pageCount"`
	// Total 总条数。
	Total int `json:"total"`
}

// StockListPage 股票列表结果：分页信息与当前页条目。
type StockListPage struct {
	// Pagination 分页元信息。
	Pagination Pagination
	// Items 当前页的股票条目。
	Items []StockListItem
}

// StockListItem 股票列表条目。
//
// 行情类字段（涨跌幅、价格、成交量额、市值、股本等）一律保留接口原始字符串：
// 它们带单位（如「亿」「万」）或百分号，转成数值会丢失口径。少数接口字段名与常规拼写
// 不一致（tradeTrunover、tradeVolumn、tradeChangeraio），tag 保持原样以便解析。
type StockListItem struct {
	// StockID 标的编号。
	StockID int64
	// Name 名称。
	Name string
	// StockCode 代码。
	StockCode string
	// MarketLabel 市场标识，如 HK。
	MarketLabel string
	// InstrumentType 标的类型编号，普通股通常为 3。
	InstrumentType int
	// ChangeRatio 涨跌幅，含百分号，如 -4.30%。
	ChangeRatio string
	// PriceDirect 价格方向，如 up、down。
	PriceDirect string
	// Change 涨跌额。
	Change string
	// PriceNominal 最新价。
	PriceNominal string
	// TradeTurnover 成交额，接口字段名为 tradeTrunover。
	TradeTurnover string
	// TradeVolume 成交量，接口字段名为 tradeVolumn。
	TradeVolume string
	// MarketValue 总市值，接口字段名为 marketVal。
	MarketValue string
	// CirculationMarketValue 流通市值。
	CirculationMarketValue string
	// TotalShares 总股本。
	TotalShares string
	// CirculationTotalShares 流通股本。
	CirculationTotalShares string
	// Change5Days 5 日涨跌幅，接口字段名为 c_5Days。
	Change5Days string
	// Change5DaysPriceDirect 5 日价格方向，接口字段名为 c_5Days_priceDirect。
	Change5DaysPriceDirect string
	// Change10Days 10 日涨跌幅，接口字段名为 c_10Days。
	Change10Days string
	// Change10DaysPriceDirect 10 日价格方向，接口字段名为 c_10Days_priceDirect。
	Change10DaysPriceDirect string
	// Change20Days 20 日涨跌幅，接口字段名为 c_20Days。
	Change20Days string
	// Change20DaysPriceDirect 20 日价格方向，接口字段名为 c_20Days_priceDirect。
	Change20DaysPriceDirect string
	// Change60Days 60 日涨跌幅，接口字段名为 c_60Days。
	Change60Days string
	// Change60DaysPriceDirect 60 日价格方向，接口字段名为 c_60Days_priceDirect。
	Change60DaysPriceDirect string
	// Change120Days 120 日涨跌幅，接口字段名为 c_120Days。
	Change120Days string
	// Change120DaysPriceDirect 120 日价格方向，接口字段名为 c_120Days_priceDirect。
	Change120DaysPriceDirect string
	// Change250Days 250 日涨跌幅，接口字段名为 c_250Days。
	Change250Days string
	// Change250DaysPriceDirect 250 日价格方向，接口字段名为 c_250Days_priceDirect。
	Change250DaysPriceDirect string
	// ChangeYearDays 一年涨跌幅，接口字段名为 c_YearDays。
	ChangeYearDays string
	// ChangeYearDaysPriceDirect 一年价格方向，接口字段名为 c_YearDays_priceDirect。
	ChangeYearDaysPriceDirect string
	// TrustAUM 信托资产规模，接口字段名为 trustAum。
	TrustAUM string
	// Leverage 杠杆率，接口可能返回 null，故用指针区分「无值」与「0」。
	Leverage *FlexNum
	// TradeChangeRatio 换手率，接口字段名为 tradeChangeraio。
	TradeChangeRatio string
	// PriceAmplitude 振幅，含百分号。
	PriceAmplitude string
	// VolumeRatio 量比，接口字段名为 volumnRatio。
	VolumeRatio string
	// BuySellRatio 买卖盘比例，接口字段名为 buysellRatio。
	BuySellRatio string
}

// UnmarshalJSON 按接口原始字段名解析。
func (s *StockListItem) UnmarshalJSON(data []byte) error {
	var raw stockListItem
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*s = StockListItem(raw)
	return nil
}

// MarshalJSON 按接口原始字段名输出，便于在测试与日志中复用同一结构。
func (s StockListItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(stockListItem(s))
}

// stockListItem 与 StockListItem 同构，仅用于承载接口原始字段名的 json tag。
type stockListItem struct {
	StockID                   int64    `json:"stockId"`
	Name                      string   `json:"name"`
	StockCode                 string   `json:"stockCode"`
	MarketLabel               string   `json:"marketLabel"`
	InstrumentType            int      `json:"instrumentType"`
	ChangeRatio               string   `json:"changeRatio"`
	PriceDirect               string   `json:"priceDirect"`
	Change                    string   `json:"change"`
	PriceNominal              string   `json:"priceNominal"`
	TradeTurnover             string   `json:"tradeTrunover"`
	TradeVolume               string   `json:"tradeVolumn"`
	MarketValue               string   `json:"marketVal"`
	CirculationMarketValue    string   `json:"circulationMarketValue"`
	TotalShares               string   `json:"totalShares"`
	CirculationTotalShares    string   `json:"circulationTotalShares"`
	Change5Days               string   `json:"c_5Days"`
	Change5DaysPriceDirect    string   `json:"c_5Days_priceDirect"`
	Change10Days              string   `json:"c_10Days"`
	Change10DaysPriceDirect   string   `json:"c_10Days_priceDirect"`
	Change20Days              string   `json:"c_20Days"`
	Change20DaysPriceDirect   string   `json:"c_20Days_priceDirect"`
	Change60Days              string   `json:"c_60Days"`
	Change60DaysPriceDirect   string   `json:"c_60Days_priceDirect"`
	Change120Days             string   `json:"c_120Days"`
	Change120DaysPriceDirect  string   `json:"c_120Days_priceDirect"`
	Change250Days             string   `json:"c_250Days"`
	Change250DaysPriceDirect  string   `json:"c_250Days_priceDirect"`
	ChangeYearDays            string   `json:"c_YearDays"`
	ChangeYearDaysPriceDirect string   `json:"c_YearDays_priceDirect"`
	TrustAUM                  string   `json:"trustAum"`
	Leverage                  *FlexNum `json:"leverage"`
	TradeChangeRatio          string   `json:"tradeChangeraio"`
	PriceAmplitude            string   `json:"priceAmplitude"`
	VolumeRatio               string   `json:"volumnRatio"`
	BuySellRatio              string   `json:"buysellRatio"`
}

// GetStockList 按市场与板块拉取股票列表（对应 get-stock-list 接口）。
//
// 支持排行榜排序与分页，页码从 0 开始，0 即第一页（传 1 会跳过第一页的 pageSize 条）。页内条目按服务端给出的顺序返回，
// 本方法不做额外排序。
func (c *Client) GetStockList(ctx context.Context, req GetStockListRequest) (StockListPage, error) {
	marketType := strings.TrimSpace(req.MarketType)
	if marketType == "" {
		return StockListPage{}, errors.New("moomoo GetStockList：marketType 不能为空")
	}
	plateType := strings.TrimSpace(req.PlateType)
	if plateType == "" {
		plateType = DefaultPlateType
	}

	// 参数顺序即签名键序，不可随意调整。
	params := []Param{
		{Name: "marketType", Value: marketType},
		{Name: "plateType", Value: plateType},
	}
	if value := strings.TrimSpace(req.RankType); value != "" {
		params = append(params, Param{Name: "rankType", Value: value})
	}
	page, err := intParam(req.Page, "page", 0)
	if err != nil {
		return StockListPage{}, err
	}
	if page != "" {
		params = append(params, Param{Name: "page", Value: page})
	}
	pageSize, err := intParam(req.PageSize, "pageSize", 1)
	if err != nil {
		return StockListPage{}, err
	}
	if pageSize != "" {
		params = append(params, Param{Name: "pageSize", Value: pageSize})
	}
	// 该接口的 _ 与 K 线不同：缺省时整键省略，不自动补当前时间。
	if timestamp := strings.TrimSpace(req.TimestampMillis); timestamp != "" {
		params = append(params, Param{Name: "_", Value: timestamp})
	}

	path := c.cfg.StockListPath
	body, err := c.getList(ctx, path, params)
	if err != nil {
		return StockListPage{}, err
	}
	var payload stockListPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return StockListPage{}, &APIError{
			Path: path, URL: c.url(path, params), Body: truncate(string(body), 500),
			Message: "响应解析失败: " + err.Error(),
		}
	}
	if err := decode(body, path, c.url(path, params), payload.Code, payload.Message); err != nil {
		return StockListPage{}, err
	}
	return StockListPage{
		Pagination: payload.Data.Pagination,
		Items:      payload.Data.List,
	}, nil
}

// stockListPayload 股票列表响应外层包裹。
type stockListPayload struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    stockListData `json:"data"`
}

// stockListData 响应 data 字段。
type stockListData struct {
	Pagination Pagination      `json:"pagination"`
	List       []StockListItem `json:"list"`
}
