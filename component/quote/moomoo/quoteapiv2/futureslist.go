package quoteapiv2

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// GetFuturesListRequest 期货合约列表请求参数（对应 get-futures-list 接口）。
//
// marketType 与 futuresType 必须成对使用，两者都是必填；常用组合见接口文档：
// 美股商品期货（2/4）、美股股指期货（2/1）、港股股指期货（1/8）。
type GetFuturesListRequest struct {
	// MarketType 市场类型，必填，原样透传。
	MarketType string
	// FuturesType 期货分类，必填，原样透传。
	FuturesType string
	// Page 页码，从 0 开始（0 即第 1 页）；为空取 DefaultFuturesPage。
	Page string
	// PageSize 每页数量，为空取 DefaultFuturesPageSize。
	PageSize string
	// TimestampMillis 毫秒时间戳，为空则省略该参数。
	TimestampMillis string
}

// FuturesListPage 期货列表结果：分页信息与当前页条目。
type FuturesListPage struct {
	// Pagination 分页元信息。
	Pagination Pagination
	// Items 当前页的期货条目。
	Items []FuturesListItem
}

// FuturesListItem 一条期货合约。
//
// 该接口的响应是位置化结构：Fields 是按字段编号给出的展示值，FieldList 是按位置给出的
// 字段定义与原始值。两者互补，一般直接取 Fields 即可；需要自己换算数值时才看 FieldList。
type FuturesListItem struct {
	// StockID 服务端内部证券唯一标识。
	// 该字段线上以字符串返回（如 "70000294"），解码时兼容字符串与数字两种形式，
	// 对外仍是 int64；缺失或空串为 0（表示无 ID）。
	StockID int64 `json:"stockId"`
	// PriceDirect 价格方向，如 up、down。
	PriceDirect string `json:"priceDirect"`
	// FieldList 字段定义数组，含原始值与类型。
	FieldList []FuturesField `json:"fieldList"`
	// Fields 字段展示值，键为字段编号的字符串形式（如 "3" 为价格）。
	// 比 FieldList 多一个键 "19"（市场标签），该键在 FieldList 中没有对应项。
	// 值可能以数字返回，解码时统一成文本。
	Fields map[string]string `json:"fields"`
}

// UnmarshalJSON 兼容 stockId 与 fields 值的两种编码。
//
// 旧版本把 stockId 声明为 int64 且未做兼容，线上返回字符串时 json.Unmarshal 在第一个
// 不匹配字段即失败，整页数据被丢弃。这里用 FlexInt64 / FlexNum 兜住两种形式，对外的
// 字段类型（int64、map[string]string）保持不变，调用方无需改动。
//
// 匿名结构体中显式声明的字段深度为 0，内嵌的 plain 深度为 1；encoding/json 的字段选择
// 规则是「浅者优先」，因此 stockId 与 fields 只由显式字段解析，其余字段仍走原结构体的
// json tag，不需要复制一份完整结构。
func (item *FuturesListItem) UnmarshalJSON(data []byte) error {
	type plain FuturesListItem
	aux := &struct {
		StockID FlexInt64          `json:"stockId"`
		Fields  map[string]FlexNum `json:"fields"`
		*plain
	}{plain: (*plain)(item)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	item.StockID = aux.StockID.Int64()
	if aux.Fields == nil {
		item.Fields = nil
		return nil
	}
	fields := make(map[string]string, len(aux.Fields))
	for key, value := range aux.Fields {
		fields[key] = value.String()
	}
	item.Fields = fields
	return nil
}

// FuturesField 位置化字段中的一项。
type FuturesField struct {
	// FieldID 字段位置，从 1 开始，各品种位置固定。
	FieldID int `json:"fieldId"`
	// FieldName 字段名，语种由服务端决定，可能中英切换，不要按名称硬编码逻辑。
	FieldName string `json:"fieldName"`
	// Value 数值类字段的原始值，通常是放大后的整数，仅数值类字段返回。
	Value string `json:"value"`
	// Text 文本类字段的值，如合约代码、名称，仅文本类字段返回。
	Text string `json:"text"`
	// Precision 小数位：价格类为放大偏移，数量类为展示小数位。
	Precision int `json:"precision"`
	// ContentType 内容类型，决定数值的还原方式。
	ContentType int `json:"contentType"`
}

// UnmarshalJSON 兼容整型字段的两种编码。
//
// fieldId / precision / contentType 在线上偶发以字符串返回（如 "3"），按 int 声明会让
// 整条数据解码失败。这里统一用 FlexInt64 兜住，对外字段类型保持 int 不变。
func (field *FuturesField) UnmarshalJSON(data []byte) error {
	type plain FuturesField
	aux := &struct {
		FieldID     FlexInt64 `json:"fieldId"`
		Precision   FlexInt64 `json:"precision"`
		ContentType FlexInt64 `json:"contentType"`
		*plain
	}{plain: (*plain)(field)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	field.FieldID = int(aux.FieldID)
	field.Precision = int(aux.Precision)
	field.ContentType = int(aux.ContentType)
	return nil
}

// GetFuturesList 按市场与分类拉取期货合约列表（对应 get-futures-list 接口）。
//
// 页码从 0 开始，列表顺序由服务端决定。响应中的 Fields 已是展示值（如 "4413.8"、
// "-0.25%"），直接使用即可；原始 Value 的还原公式依赖站点字段字典，本项目未收录，
// 因此本包不做数值换算，避免按猜测的口径算出错误结果。
func (c *Client) GetFuturesList(ctx context.Context, req GetFuturesListRequest) (FuturesListPage, error) {
	marketType := strings.TrimSpace(req.MarketType)
	if marketType == "" {
		return FuturesListPage{}, errors.New("moomoo GetFuturesList：marketType 不能为空")
	}
	futuresType := strings.TrimSpace(req.FuturesType)
	if futuresType == "" {
		return FuturesListPage{}, errors.New("moomoo GetFuturesList：futuresType 不能为空")
	}
	page, err := intParam(defaultIfEmpty(req.Page, DefaultFuturesPage), "page", 0)
	if err != nil {
		return FuturesListPage{}, err
	}
	pageSize, err := intParam(defaultIfEmpty(req.PageSize, DefaultFuturesPageSize), "pageSize", 1)
	if err != nil {
		return FuturesListPage{}, err
	}

	// 参数顺序即签名键序。
	params := []Param{
		{Name: "marketType", Value: marketType},
		{Name: "futuresType", Value: futuresType},
		{Name: "page", Value: page},
		{Name: "pageSize", Value: pageSize},
	}
	if timestamp := strings.TrimSpace(req.TimestampMillis); timestamp != "" {
		params = append(params, Param{Name: "_", Value: timestamp})
	}

	path := c.cfg.FuturesListPath
	body, err := c.getList(ctx, path, params)
	if err != nil {
		return FuturesListPage{}, err
	}
	var payload futuresListPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return FuturesListPage{}, &APIError{
			Path: path, URL: c.url(path, params), Body: truncate(string(body), 500),
			Message: "响应解析失败: " + err.Error(),
		}
	}
	if err := decode(body, path, c.url(path, params), payload.Code, payload.Message); err != nil {
		return FuturesListPage{}, err
	}
	return FuturesListPage{
		Pagination: payload.Data.Pagination,
		Items:      payload.Data.List,
	}, nil
}

// futuresListPayload 期货列表响应外层包裹。
type futuresListPayload struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    futuresListData `json:"data"`
}

// futuresListData 响应 data 字段。
type futuresListData struct {
	Pagination Pagination        `json:"pagination"`
	List       []FuturesListItem `json:"list"`
}
