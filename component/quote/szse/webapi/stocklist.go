package webapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/acnxo/MetaExt/component/quote/internal/fetch"
)

// GetStockListRequest 是证券列表的请求参数。
type GetStockListRequest struct {
	// All 为 true 时自动翻完全部页；为 false 时只取 Page 指定的那一页。
	All bool
	// Page 页码，从 1 开始；All 为 true 时忽略。
	Page int
	// Workers 并发翻页数，为空或非正时取 DefaultWorkers。
	// 只在 All 为 true 且总页数大于 1 时生效。
	Workers int
}

// StockItem 是一条深市证券。
type StockItem struct {
	// Code 证券代码，六位字符串。
	Code string
	// Name 证券简称，已去掉高亮标签，可能含 ST / 退 标记。
	Name string
	// ListDate 上市日期，yyyy-MM-dd；缺失时为空串。
	ListDate string
}

// StockListPage 是一页（或全部）证券列表。
type StockListPage struct {
	// Items 证券条目，按页序拼接。
	Items []StockItem
	// PageNo 当前页码；All 为 true 时为 1。
	PageNo int
	// PageCount 总页数。
	PageCount int
	// RecordCount 总记录数。
	RecordCount int
}

// GetStockList 拉取深市证券列表（含上市日期）。
//
// All 为 true 时先取第 1 页拿到总页数，再用 Workers 并发翻完其余页，按页序拼接；
// 任一页失败即返回错误，不返回残缺结果。本站点单页固定 20 条，全量约 146 次请求。
func (c *Client) GetStockList(ctx context.Context, req GetStockListRequest) (*StockListPage, error) {
	pageNo := req.Page
	if pageNo <= 0 {
		pageNo = 1
	}
	first, err := c.fetchPage(ctx, pageNo)
	if err != nil {
		return nil, err
	}
	if !req.All || first.PageCount <= pageNo {
		return first, nil
	}

	workers := req.Workers
	if workers <= 0 {
		workers = DefaultWorkers
	}
	if workers > first.PageCount {
		workers = first.PageCount
	}

	pages := make([][]StockItem, first.PageCount+1)
	pages[pageNo] = first.Items

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
		sem      = make(chan struct{}, workers)
	)
	for p := 1; p <= first.PageCount; p++ {
		if p == pageNo {
			continue
		}
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := c.fetchPage(ctx, page)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			pages[page] = result.Items
		}(p)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}

	items := make([]StockItem, 0, first.RecordCount)
	for p := 1; p <= first.PageCount; p++ {
		items = append(items, pages[p]...)
	}
	return &StockListPage{
		Items:       items,
		PageNo:      pageNo,
		PageCount:   first.PageCount,
		RecordCount: first.RecordCount,
	}, nil
}

// fetchPage 拉取并解析单页。
func (c *Client) fetchPage(ctx context.Context, page int) (*StockListPage, error) {
	query := url.Values{}
	query.Set("SHOWTYPE", "JSON")
	query.Set("CATALOGID", DefaultCatalogID)
	query.Set("TABKEY", DefaultTabKey)
	query.Set("PAGENO", strconv.Itoa(page))
	apiURL := c.cfg.BaseURL + c.cfg.StockListPath + "?" + query.Encode()

	raw, err := c.req.Get(ctx, apiURL, map[string]string{"referer": c.cfg.Referer})
	if err != nil {
		return nil, wrapError(c.cfg.StockListPath, apiURL, err)
	}

	var payload []rawPage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &APIError{
			Path:    c.cfg.StockListPath,
			URL:     apiURL,
			Message: "响应解析失败: " + err.Error(),
			Body:    truncateBody(raw),
			Err:     err,
		}
	}
	if len(payload) == 0 {
		return nil, &APIError{
			Path:    c.cfg.StockListPath,
			URL:     apiURL,
			Message: "响应缺少页签数据",
			Body:    truncateBody(raw),
		}
	}

	tab := payload[0]
	items := make([]StockItem, 0, len(tab.Data))
	for _, row := range tab.Data {
		code := strings.TrimSpace(row.Code)
		if code == "" {
			continue
		}
		items = append(items, StockItem{
			Code:     code,
			Name:     cleanName(row.Name),
			ListDate: strings.TrimSpace(row.ListDate),
		})
	}
	return &StockListPage{
		Items:       items,
		PageNo:      atoiOr(tab.Metadata.PageNo, page),
		PageCount:   atoiOr(tab.Metadata.PageCount, 1),
		RecordCount: atoiOr(tab.Metadata.RecordCount, len(items)),
	}, nil
}

// rawPage 是响应数组中的一个页签元素。
type rawPage struct {
	Metadata struct {
		PageNo      string `json:"pageno"`
		PageCount   string `json:"pagecount"`
		RecordCount string `json:"recordcount"`
	} `json:"metadata"`
	Data []struct {
		Code     string `json:"agdm"`
		Name     string `json:"agjc"`
		ListDate string `json:"agssrq"`
	} `json:"data"`
}

// cleanName 去掉简称里的高亮标签。
//
// 实测形态有 "<u>平安银行</u>" 与 "<a href='...' target='_blank'>XD平安银行</a>" 两种，
// 先去掉 <u>，再按 "'>" 取最后一段、按 "<" 取第一段。
func cleanName(raw string) string {
	text := strings.ReplaceAll(raw, "<u>", " ")
	text = strings.ReplaceAll(text, "</u>", " ")
	if i := strings.LastIndex(text, "'>"); i >= 0 {
		text = text[i+2:]
	}
	if i := strings.Index(text, "<"); i >= 0 {
		text = text[:i]
	}
	return strings.TrimSpace(text)
}

// atoiOr 解析整数，失败时返回默认值。
func atoiOr(text string, fallback int) int {
	if value, err := strconv.Atoi(strings.TrimSpace(text)); err == nil {
		return value
	}
	return fallback
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
