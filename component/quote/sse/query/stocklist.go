package query

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/acnxo/MetaExt/component/quote/internal/fetch"
)

// GetStockListRequest 是证券列表的请求参数。
type GetStockListRequest struct {
	// PageSize 每页条数，为空或非正时取 DefaultPageSize。
	// 该接口一次返回全部沪市 A 股，不需要翻页。
	PageSize int
}

// StockItem 是一条沪市证券。
type StockItem struct {
	// Code 证券代码，六位字符串。
	Code string
	// Name 证券简称，可能含 ST / 退 标记。
	Name string
	// ListDate 上市日期，yyyyMMdd；非法值已在校验阶段丢弃。
	ListDate string
}

// GetStockList 拉取沪市 A 股列表（含上市日期）。
//
// 返回结果保持站点顺序，未做板块过滤：沪市主板为 600/601/603/605 前缀，
// 科创板为 688 前缀，调用方按需过滤。
//
// 网络错误与 5xx/429 会自动重试；响应不是合法 JSON 或缺少 result 字段时返回 *APIError。
func (c *Client) GetStockList(ctx context.Context, req GetStockListRequest) ([]StockItem, error) {
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}

	query := url.Values{}
	query.Set("sqlId", DefaultSQLID)
	query.Set("type", "inParams")
	query.Set("STOCK_TYPE", "1")
	query.Set("COMPANY_STATUS", "2,4,5,7,8")
	query.Set("pageHelp.pageSize", strconv.Itoa(pageSize))
	query.Set("pageHelp.pageNo", "1")
	query.Set("pageHelp.beginPage", "1")
	query.Set("pageHelp.endPage", "1")
	query.Set("pageHelp.cacheSize", "1")
	apiURL := c.cfg.BaseURL + c.cfg.StockListPath + "?" + query.Encode()

	raw, err := c.req.Get(ctx, apiURL, map[string]string{"referer": c.cfg.Referer})
	if err != nil {
		return nil, wrapError(c.cfg.StockListPath, apiURL, "", err)
	}

	var payload stockListPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &APIError{
			Path:    c.cfg.StockListPath,
			URL:     apiURL,
			Message: "响应解析失败: " + err.Error(),
			Body:    truncateBody(raw),
			Err:     err,
		}
	}

	items := make([]StockItem, 0, len(payload.Result))
	for _, row := range payload.Result {
		code := strings.TrimSpace(row.Code)
		name := strings.TrimSpace(row.Name)
		date := strings.TrimSpace(row.ListDate)

		// 上市日期必须是八位数字，否则宁可丢弃也不要写入错误值。
		if code == "" || name == "" || !isDate(date) {
			continue
		}
		items = append(items, StockItem{Code: code, Name: name, ListDate: date})
	}
	return items, nil
}

// stockListPayload 是证券列表响应的外层结构。
type stockListPayload struct {
	Result []stockListRow `json:"result"`
}

// stockListRow 是响应中的一条记录。
type stockListRow struct {
	Code     string `json:"A_STOCK_CODE"`
	Name     string `json:"SEC_NAME_CN"`
	ListDate string `json:"LIST_DATE"`
}

// isDate 判断是否为八位纯数字日期。
func isDate(text string) bool {
	if len(text) != 8 {
		return false
	}
	for _, ch := range text {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// wrapError 把请求层的错误包装成本包的 *APIError。
func wrapError(path, apiURL, body string, err error) error {
	var statusErr *fetch.StatusError
	if errors.As(err, &statusErr) {
		return &APIError{
			Path:       path,
			URL:        apiURL,
			HTTPStatus: statusErr.Status,
			Message:    statusErr.Body,
			Body:       statusErr.Body,
			Err:        err,
		}
	}
	return &APIError{Path: path, URL: apiURL, Message: err.Error(), Body: body, Err: err}
}

// truncateBody 截断响应体，避免错误信息过大。
func truncateBody(raw []byte) string {
	const limit = 300
	text := strings.TrimSpace(string(raw))
	if len(text) <= limit {
		return text
	}
	return fmt.Sprintf("%s...", text[:limit])
}
