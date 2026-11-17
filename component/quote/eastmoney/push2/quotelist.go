package push2

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

// stockListPath 行情列表接口路径。
const stockListPath = "/api/qt/clist/get"

// listFields 请求的字段码，与 QuoteListItem 的成员一一对应。
//
//	f2 最新价 / f5 成交量(手) / f6 成交额(元) / f12 代码 / f13 市场(0 深 1 沪)
//	f14 名称 / f15 最高 / f16 最低 / f17 今开 / f20 总市值(元) / f21 流通市值(元)
//	f26 上市日期 / f124 行情时间戳(秒)
const listFields = "f2,f5,f6,f12,f13,f14,f15,f16,f17,f20,f21,f26,f124"

// GetQuoteListRequest 是行情列表的请求参数。
type GetQuoteListRequest struct {
	// Page 页码，从 1 开始；为空或非正时取 1。
	Page int
	// PageSize 每页条数，为空或非正时取 DefaultPageSize。
	PageSize int
	// SortField 排序字段：SortByTotalMV（默认）或 SortByListDate。
	SortField string
	// Desc 为 true 时按 SortField 降序，否则升序。
	Desc bool
	// Universe 市场过滤串，默认为 UniverseMainBoard。
	Universe string
}

// QuoteListItem 是一条行情快照。
//
// 价格与金额统一为 float64，单位为元（成交量为手）；需要原始字符串时查 Raw。
type QuoteListItem struct {
	// Code 证券代码，六位字符串。
	Code string
	// Name 证券简称，可能含 ST / 退 标记。
	Name string
	// Last 最新价（元），无行情时为 0。
	Last float64
	// Open 今开（元）。
	Open float64
	// High 最高（元）。
	High float64
	// Low 最低（元）。
	Low float64
	// Volume 成交量（手）。
	Volume float64
	// Amount 成交额（元）。
	Amount float64
	// TotalMV 总市值（元）。
	TotalMV float64
	// FloatMV 流通市值（元）。
	FloatMV float64
	// ListDate 上市日期，yyyy-MM-dd；缺失时为空串。
	ListDate string
	// Market 市场，0 深市、1 沪市。
	Market int
	// Timestamp 行情时间戳（秒）。
	Timestamp int64
	// Raw 原始字段码 → 原始文本，便于排障与后续扩展。
	Raw map[string]string
}

// QuoteListPage 是行情列表的一页。
type QuoteListPage struct {
	// Items 当前页条目，已剔除无行情的记录。
	Items []QuoteListItem
	// Total 符合条件的总条数。
	Total int
	// Page 请求的页码。
	Page int
	// PageSize 请求的每页条数。
	PageSize int
	// SortField 请求的排序字段。
	SortField string
	// Desc 是否为降序。
	Desc bool
}

// GetQuoteList 拉取一页行情快照。
//
// 站点按 SortField 排序后返回，配合同一排序翻页可做「按总市值升序扫描」。
// 无行情（退市、长期停牌）的记录会被剔除，因此返回条数可能少于 PageSize。
//
// 镜像域名会依次轮换；`data` 为 null 时视为被限频，自动换下一个镜像。
func (c *Client) GetQuoteList(ctx context.Context, req GetQuoteListRequest) (*QuoteListPage, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	sortField := strings.TrimSpace(req.SortField)
	if sortField == "" {
		sortField = SortByTotalMV
	}
	universe := strings.TrimSpace(req.Universe)
	if universe == "" {
		universe = UniverseMainBoard
	}
	order := "0"
	if req.Desc {
		order = "1"
	}

	query := url.Values{}
	query.Set("pn", strconv.Itoa(page))
	query.Set("pz", strconv.Itoa(pageSize))
	query.Set("po", order)
	query.Set("np", "1")
	query.Set("fltt", "2")
	query.Set("invt", "2")
	query.Set("fid", sortField)
	query.Set("fs", universe)
	query.Set("fields", listFields)

	raw, apiURL, err := c.getRaw(ctx, c.cfg.ListHosts, stockListPath, query, hasData)
	if err != nil {
		return nil, err
	}

	var payload quoteListPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, wrapParseError(stockListPath, apiURL, raw, err)
	}
	if payload.Data == nil {
		return nil, &APIError{Path: stockListPath, URL: apiURL, Message: "响应缺少 data", Body: truncateBody(raw)}
	}

	items := make([]QuoteListItem, 0, len(payload.Data.Diff))
	for _, row := range payload.Data.Diff {
		item := decodeQuoteRow(row)
		if item.Code == "" || item.Last <= 0 || item.TotalMV <= 0 {
			continue
		}
		items = append(items, item)
	}
	return &QuoteListPage{
		Items:     items,
		Total:     payload.Data.Total,
		Page:      page,
		PageSize:  pageSize,
		SortField: sortField,
		Desc:      req.Desc,
	}, nil
}

// quoteListPayload 是行情列表响应的外层结构。
type quoteListPayload struct {
	RC   int `json:"rc"`
	Data *struct {
		Total int                          `json:"total"`
		Diff  []map[string]json.RawMessage `json:"diff"`
	} `json:"data"`
}

// decodeQuoteRow 把一行原始字段转成 QuoteListItem。
func decodeQuoteRow(row map[string]json.RawMessage) QuoteListItem {
	raw := make(map[string]string, len(row))
	for code, value := range row {
		raw[code] = rawText(value)
	}
	return QuoteListItem{
		Code:      strings.TrimSpace(raw["f12"]),
		Name:      strings.TrimSpace(raw["f14"]),
		Last:      parseNum(raw["f2"]),
		Open:      parseNum(raw["f17"]),
		High:      parseNum(raw["f15"]),
		Low:       parseNum(raw["f16"]),
		Volume:    parseNum(raw["f5"]),
		Amount:    parseNum(raw["f6"]),
		TotalMV:   parseNum(raw["f20"]),
		FloatMV:   parseNum(raw["f21"]),
		ListDate:  strings.TrimSpace(raw["f26"]),
		Market:    int(parseNum(raw["f13"])),
		Timestamp: int64(parseNum(raw["f124"])),
		Raw:       raw,
	}
}

// rawText 把 JSON 原始值转成文本：字符串去引号，其它取原文。
func rawText(value json.RawMessage) string {
	text := strings.TrimSpace(string(value))
	if len(text) >= 2 && text[0] == '"' {
		var s string
		if err := json.Unmarshal(value, &s); err == nil {
			return s
		}
	}
	return text
}

// parseNum 解析数值文本，缺失或非法时返回 0。
func parseNum(text string) float64 {
	text = strings.TrimSpace(text)
	if text == "" || text == "-" || text == "null" {
		return 0
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0
	}
	return value
}
