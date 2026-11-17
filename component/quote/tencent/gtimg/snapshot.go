package gtimg

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"

	"github.com/acnxo/MetaExt/component/quote/internal/fetch"
)

// 快照响应的字段下标（按 `~` 拆分后的位置）。
//
//	1 名称 / 2 代码 / 3 最新价 / 5 今开 / 6 成交量(手) / 30 时间戳
//	37 成交额(万元) / 44 流通市值(亿元) / 45 总市值(亿元)
const (
	idxName        = 1
	idxCode        = 2
	idxLast        = 3
	idxOpen        = 5
	idxVolume      = 6
	idxTimestamp   = 30
	idxAmount      = 37
	idxFloatMV     = 44
	idxTotalMV     = 45
	minSnapshotSeg = 46
)

// GetQuoteSnapshotRequest 是批量快照的请求参数。
type GetQuoteSnapshotRequest struct {
	// Codes 六位证券代码；空值会被剔除，重复项只保留首次出现。
	Codes []string
	// Workers 分批并发数，为空或非正时取配置值。
	Workers int
}

// QuoteItem 是一条行情快照。
//
// 市值由「亿元」换算成元，成交额由「万元」换算成元；
// **成交量原始单位是「手」**，不做换算。
type QuoteItem struct {
	// Code 证券代码，六位。
	Code string
	// Name 证券简称，可能是 GBK 解码后的中文。
	Name string
	// Last 最新价（元）。
	Last float64
	// Open 今开（元）。
	Open float64
	// Volume 成交量（**手**）。
	Volume float64
	// Amount 成交额（元）。
	Amount float64
	// TotalMV 总市值（元）。
	TotalMV float64
	// FloatMV 流通市值（元）。
	FloatMV float64
	// Timestamp 行情时间戳，yyyyMMddHHmmss。
	Timestamp string
}

// QuoteSnapshot 是一次批量快照的结果。
type QuoteSnapshot struct {
	// Items 行情条目，顺序与请求的代码顺序无关。
	Items []QuoteItem
}

// GetQuoteSnapshot 批量拉取行情快照。
//
// 代码按 Batch 分批并发请求；某批失败（典型是 URL 太长触发 414）时自动二分，
// 直到单批不超过 100 只；仍失败则整体返回错误，不返回残缺结果。
//
// 返回顺序按批次与批内出现顺序排列，调用方如需按请求顺序对齐请自行建索引。
func (c *Client) GetQuoteSnapshot(ctx context.Context, req GetQuoteSnapshotRequest) (*QuoteSnapshot, error) {
	codes := normalizeCodes(req.Codes)
	if len(codes) == 0 {
		return nil, errors.New("腾讯 GetQuoteSnapshot：codes 不能为空")
	}
	workers := req.Workers
	if workers <= 0 {
		workers = c.cfg.Workers
	}

	chunks := splitCodes(codes, c.cfg.Batch)
	results := make([][]QuoteItem, len(chunks))
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
		sem      = make(chan struct{}, workers)
	)
	for i, chunk := range chunks {
		wg.Add(1)
		go func(index int, batch []string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			items, err := c.fetchChunk(ctx, batch)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			results[index] = items
		}(i, chunk)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}

	total := 0
	for _, items := range results {
		total += len(items)
	}
	out := make([]QuoteItem, 0, total)
	for _, items := range results {
		out = append(out, items...)
	}
	return &QuoteSnapshot{Items: out}, nil
}

// fetchChunk 请求一批代码，失败时二分重试。
func (c *Client) fetchChunk(ctx context.Context, codes []string) ([]QuoteItem, error) {
	if len(codes) == 0 {
		return nil, nil
	}
	symbols := make([]string, 0, len(codes))
	for _, code := range codes {
		if symbol := symbolOf(code); symbol != "" {
			symbols = append(symbols, symbol)
		}
	}
	apiURL := c.cfg.BaseURL + "/q=" + strings.Join(symbols, ",")

	text, err := c.req.GetText(ctx, apiURL, map[string]string{"referer": c.cfg.Referer}, fetch.GBK)
	if err != nil {
		// URL 过长（414）或临时失败：二分后重试，直到单批不超过阈值。
		if len(codes) <= c.cfg.MinSplitCodes {
			return nil, wrapError("/q", apiURL, err)
		}
		mid := len(codes) / 2
		left, leftErr := c.fetchChunk(ctx, codes[:mid])
		if leftErr != nil {
			return nil, leftErr
		}
		right, rightErr := c.fetchChunk(ctx, codes[mid:])
		if rightErr != nil {
			return nil, rightErr
		}
		return append(left, right...), nil
	}
	return parseSnapshotText(text), nil
}

// parseSnapshotText 解析 `v_sz000001="...";` 形式的文本。
func parseSnapshotText(text string) []QuoteItem {
	items := make([]QuoteItem, 0, 64)
	for _, line := range strings.Split(text, ";") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, `="`) {
			continue
		}
		value := line[strings.Index(line, `="`)+2:]
		value = strings.TrimSuffix(value, `"`)
		fields := strings.Split(value, "~")
		if len(fields) < minSnapshotSeg || strings.TrimSpace(fields[idxCode]) == "" {
			continue
		}

		item := QuoteItem{
			Code:      strings.TrimSpace(fields[idxCode]),
			Name:      strings.TrimSpace(fields[idxName]),
			Last:      parseNum(fields[idxLast]),
			Open:      parseNum(fields[idxOpen]),
			Volume:    parseNum(fields[idxVolume]),
			Amount:    parseNum(fields[idxAmount]) * 1e4,  // 万元 → 元
			TotalMV:   parseNum(fields[idxTotalMV]) * 1e8, // 亿元 → 元
			FloatMV:   parseNum(fields[idxFloatMV]) * 1e8, // 亿元 → 元
			Timestamp: strings.TrimSpace(fields[idxTimestamp]),
		}
		if item.Last <= 0 || item.TotalMV <= 0 {
			continue
		}
		items = append(items, item)
	}
	return items
}

// normalizeCodes 去空白、去空值、去重（保留首次出现）。
func normalizeCodes(codes []string) []string {
	out := make([]string, 0, len(codes))
	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out
}

// splitCodes 按批大小切分。
func splitCodes(codes []string, batch int) [][]string {
	if batch <= 0 || len(codes) <= batch {
		return [][]string{codes}
	}
	chunks := make([][]string, 0, (len(codes)+batch-1)/batch)
	for start := 0; start < len(codes); start += batch {
		end := start + batch
		if end > len(codes) {
			end = len(codes)
		}
		chunks = append(chunks, codes[start:end])
	}
	return chunks
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
