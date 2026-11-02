package quoteapiv2

import (
	"context"
	"encoding/json"
	"strings"
)

// GetEtfListRequest ETF 列表请求参数（对应 get-etf-list 接口）。
//
// 与其他列表接口不同，本接口的各参数都有服务端默认值，且 leverage 即使为空也**必须**
// 作为空串参与签名：省略该键会导致签名不匹配，服务端返回 -403。
type GetEtfListRequest struct {
	// RouteType 站点路由类型，为空取 DefaultEtfRouteType。
	RouteType string
	// PlateID 板块 ID，为空取 DefaultEtfPlateID。
	PlateID string
	// RankType 排行类型，为空取 DefaultEtfRankType。
	RankType string
	// Leverage 杠杆筛选，为空时仍以空串参与签名，不能省略该参数。
	Leverage string
	// Page 页码，从 0 开始（0 即第 1 页）；为空取 DefaultEtfPage。
	Page string
	// PageSize 每页数量，为空取 DefaultEtfPageSize。
	PageSize string
	// TimestampMillis 毫秒时间戳，为空则省略该参数。
	TimestampMillis string
}

// EtfListPage ETF 列表结果：分页信息与当前页条目。
//
// 条目结构与股票列表一致，复用 StockListItem；本接口中 InstrumentType 为 4，
// Leverage 为字符串形式（如 "1"）。
type EtfListPage struct {
	// Pagination 分页元信息。
	Pagination Pagination
	// Items 当前页的 ETF 条目。
	Items []StockListItem
}

// GetEtfList 按板块拉取 ETF 列表（对应 get-etf-list 接口）。
//
// 默认按资产规模降序并分页，页码从 0 开始；列表顺序由服务端决定，本方法不做额外排序。
func (c *Client) GetEtfList(ctx context.Context, req GetEtfListRequest) (EtfListPage, error) {
	page, err := intParam(defaultIfEmpty(req.Page, DefaultEtfPage), "page", 0)
	if err != nil {
		return EtfListPage{}, err
	}
	pageSize, err := intParam(defaultIfEmpty(req.PageSize, DefaultEtfPageSize), "pageSize", 1)
	if err != nil {
		return EtfListPage{}, err
	}

	// 参数顺序即签名键序；leverage 即使为空也保留，不可省略。
	params := []Param{
		{Name: "routeType", Value: defaultIfEmpty(req.RouteType, DefaultEtfRouteType)},
		{Name: "plateId", Value: defaultIfEmpty(req.PlateID, DefaultEtfPlateID)},
		{Name: "rankType", Value: defaultIfEmpty(req.RankType, DefaultEtfRankType)},
		{Name: "leverage", Value: strings.TrimSpace(req.Leverage)},
		{Name: "page", Value: page},
		{Name: "pageSize", Value: pageSize},
	}
	if timestamp := strings.TrimSpace(req.TimestampMillis); timestamp != "" {
		params = append(params, Param{Name: "_", Value: timestamp})
	}

	path := c.cfg.EtfListPath
	body, err := c.get(ctx, path, params)
	if err != nil {
		return EtfListPage{}, err
	}
	var payload etfListPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return EtfListPage{}, &APIError{
			Path: path, URL: c.url(path, params), Body: truncate(string(body), 500),
			Message: "响应解析失败: " + err.Error(),
		}
	}
	if err := decode(body, path, c.url(path, params), payload.Code, payload.Message); err != nil {
		return EtfListPage{}, err
	}
	return EtfListPage{
		Pagination: payload.Data.Pagination,
		Items:      payload.Data.List,
	}, nil
}

// etfListPayload ETF 列表响应外层包裹。
type etfListPayload struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    etfListData `json:"data"`
}

// etfListData 响应 data 字段。
type etfListData struct {
	Pagination Pagination      `json:"pagination"`
	List       []StockListItem `json:"list"`
}
