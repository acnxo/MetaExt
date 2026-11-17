package dataapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/acnxo/MetaExt/component/quote/internal/fetch"
)

// StockItem 是一条深市证券。
type StockItem struct {
	// Code 证券代码，六位字符串。
	Code string
	// Name 证券简称，可能含 ST / 退 标记。
	Name string
}

// GetStockList 拉取深市全部证券的代码与简称。
//
// 一次请求返回全部，无分页；结果包含创业板、B 股等，调用方按代码前缀过滤。
// 该接口不返回上市日期，StockItem 中也没有该字段。
func (c *Client) GetStockList(ctx context.Context) ([]StockItem, error) {
	apiURL := c.cfg.BaseURL + c.cfg.StockListPath

	raw, err := c.req.Get(ctx, apiURL, map[string]string{"referer": c.cfg.Referer})
	if err != nil {
		return nil, wrapError(c.cfg.StockListPath, apiURL, err)
	}

	items, err := parseStockList(raw)
	if err != nil {
		return nil, &APIError{
			Path:    c.cfg.StockListPath,
			URL:     apiURL,
			Message: "响应解析失败: " + err.Error(),
			Body:    truncateBody(raw),
			Err:     err,
		}
	}
	if len(items) == 0 {
		return nil, &APIError{
			Path:    c.cfg.StockListPath,
			URL:     apiURL,
			Message: "响应不含任何证券",
			Body:    truncateBody(raw),
		}
	}
	return items, nil
}

// cninfoRow 是响应中的一条记录。
type cninfoRow struct {
	Code string `json:"code"`
	Name string `json:"zwjc"`
}

// parseStockList 兼容两种响应形态：`{"stockList": [...]}` 与直接返回数组。
func parseStockList(raw []byte) ([]StockItem, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, errors.New("空响应")
	}

	var rows []cninfoRow
	if trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &rows); err != nil {
			return nil, err
		}
	} else {
		var payload struct {
			StockList []cninfoRow `json:"stockList"`
		}
		if err := json.Unmarshal(trimmed, &payload); err != nil {
			return nil, err
		}
		rows = payload.StockList
	}

	items := make([]StockItem, 0, len(rows))
	for _, row := range rows {
		code := strings.TrimSpace(row.Code)
		if code == "" {
			continue
		}
		items = append(items, StockItem{Code: code, Name: strings.TrimSpace(row.Name)})
	}
	return items, nil
}

// wrapError 把请求层的错误包装成本包的 *APIError。
func wrapError(path, apiURL string, err error) error {
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
	return &APIError{Path: path, URL: apiURL, Message: err.Error(), Err: err}
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
