package quoteapiv2

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// GetNewsListRequest 个股新闻列表请求参数（对应 get-news-list 接口）。
//
// 该接口的参数命名风格是混用的：stock_id、market_type 为 snake_case，
// subType 为 camelCase。必须原样保留，不能为了统一风格而改名。
type GetNewsListRequest struct {
	// StockID 服务端内部股票 ID，snake_case 的 stock_id，必填。
	StockID string
	// MarketType 市场类型，为空取 DefaultNewsMarketType。
	MarketType string
	// Type 新闻类型，为空取 DefaultNewsType；缺该参数服务端返回 404。
	Type string
	// SubType 新闻子类型，为空取 DefaultNewsSubType。
	SubType string
	// TimestampMillis 毫秒时间戳（查询串 _ 参数），为空则省略该参数。
	TimestampMillis string
}

// NewsListPage 新闻列表结果：分页标记与当前页条目。
type NewsListPage struct {
	// ServerTime 服务端时间戳，毫秒。
	ServerTime int64
	// HasMore 是否还有更多新闻。
	HasMore bool
	// SeqMark 分页标记，翻页请求时回传。
	SeqMark string
	// Items 当前页新闻条目。
	Items []NewsListItem
}

// NewsListItem 一条新闻。
type NewsListItem struct {
	// ID 新闻唯一 ID。
	ID string `json:"id"`
	// Title 新闻标题。
	Title string `json:"title"`
	// Time 发布时间戳，秒级 UNIX 时间戳。
	Time int64 `json:"time"`
	// Source 新闻来源。
	Source string `json:"source"`
	// URL 新闻详情链接。
	URL string `json:"url"`
	// Summary 新闻摘要。
	Summary string `json:"summary"`
}

// GetNewsList 获取指定股票的新闻列表（对应 get-news-list 接口）。
//
// 结果按服务端顺序返回，是否还有更多由 HasMore 判断。响应中的 SeqMark 用于翻页，
// 但翻页时对应的查询参数名尚未实测确认，请求侧暂不透传，避免编造参数。
func (c *Client) GetNewsList(ctx context.Context, req GetNewsListRequest) (NewsListPage, error) {
	stockID := strings.TrimSpace(req.StockID)
	if stockID == "" {
		return NewsListPage{}, errors.New("moomoo GetNewsList：stock_id 不能为空")
	}

	// 参数顺序即签名键序，键名保持接口原始的 snake_case / camelCase 混用风格。
	params := []Param{
		{Name: "stock_id", Value: stockID},
		{Name: "market_type", Value: defaultIfEmpty(req.MarketType, DefaultNewsMarketType)},
		{Name: "type", Value: defaultIfEmpty(req.Type, DefaultNewsType)},
		{Name: "subType", Value: defaultIfEmpty(req.SubType, DefaultNewsSubType)},
	}
	if timestamp := defaultIfEmpty(req.TimestampMillis, ""); timestamp != "" {
		params = append(params, Param{Name: "_", Value: timestamp})
	}

	path := c.cfg.NewsListPath
	body, err := c.get(ctx, path, params)
	if err != nil {
		return NewsListPage{}, err
	}
	var payload newsListPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return NewsListPage{}, &APIError{
			Path: path, URL: c.url(path, params), Body: truncate(string(body), 500),
			Message: "响应解析失败: " + err.Error(),
		}
	}
	if err := decode(body, path, c.url(path, params), payload.Code, payload.Message); err != nil {
		return NewsListPage{}, err
	}
	return NewsListPage{
		ServerTime: payload.Data.ServerTime,
		HasMore:    payload.Data.HasMore,
		SeqMark:    payload.Data.SeqMark,
		Items:      payload.Data.List,
	}, nil
}

// newsListPayload 新闻列表响应外层包裹。
type newsListPayload struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Data    newsListData `json:"data"`
}

// newsListData 响应 data 字段。
type newsListData struct {
	ServerTime int64          `json:"server_time"`
	HasMore    bool           `json:"has_more"`
	SeqMark    string         `json:"seq_mark"`
	List       []NewsListItem `json:"list"`
}
