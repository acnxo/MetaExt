package indexapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/acnxo/MetaExt/component/quote/internal/fetch"
)

// GetIndexPerfRequest 是指数日行情的请求参数。
type GetIndexPerfRequest struct {
	// IndexCode 指数代码（六位数字），如 `932000`、`000985`。
	IndexCode string
	// Begin 起始日期，yyyyMMdd。
	Begin string
	// End 结束日期，yyyyMMdd。
	End string
}

// IndexBar 是一根指数日线；价格为点。
type IndexBar struct {
	// Date 交易日，yyyy-MM-dd（已归一化）。
	Date string
	// Open 开盘（点）。
	Open float64
	// High 最高（点）。
	High float64
	// Low 最低（点）。
	Low float64
	// Close 收盘（点）。
	Close float64
	// Change 涨跌点数。
	Change float64
	// ChangeRatio 涨跌幅（%）。
	ChangeRatio float64
}

// IndexPerfResult 是指数日行情结果。
type IndexPerfResult struct {
	// IndexCode 请求的指数代码。
	IndexCode string
	// Bars 日线，按日期升序、已去重。
	Bars []IndexBar
}

// GetIndexPerf 拉取指数日行情。
//
// 区间早于指数发布日时返回空 Bars 而不是错误。响应缺少 data 或不是合法 JSON 时
// 返回 *APIError；官网重复行按日期去重，保留最后一条。
func (c *Client) GetIndexPerf(ctx context.Context, req GetIndexPerfRequest) (*IndexPerfResult, error) {
	indexCode := strings.TrimSpace(req.IndexCode)
	if indexCode == "" {
		return nil, errors.New("中证指数 GetIndexPerf：IndexCode 不能为空")
	}

	query := url.Values{}
	query.Set("indexCode", indexCode)
	if begin := compactDate(req.Begin); begin != "" {
		query.Set("startDate", begin)
	}
	if end := compactDate(req.End); end != "" {
		query.Set("endDate", end)
	}
	apiURL := c.cfg.BaseURL + c.cfg.IndexPerfPath + "?" + query.Encode()

	raw, err := c.req.Get(ctx, apiURL, map[string]string{"referer": c.cfg.Referer})
	if err != nil {
		return nil, wrapError(c.cfg.IndexPerfPath, apiURL, err)
	}

	var payload perfPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &APIError{
			Path:    c.cfg.IndexPerfPath,
			URL:     apiURL,
			Message: "响应解析失败: " + err.Error(),
			Body:    truncateBody(raw),
			Err:     err,
		}
	}

	// 同一交易日可能有多行，保留最后一条。
	byDate := make(map[string]IndexBar, len(payload.Data))
	for _, row := range payload.Data {
		date := normalizeDate(row.TradeDate)
		if date == "" {
			continue
		}
		bar := IndexBar{
			Date:        date,
			Open:        parseNum(row.Open),
			High:        parseNum(row.High),
			Low:         parseNum(row.Low),
			Close:       parseNum(row.Close),
			Change:      parseNum(row.Change),
			ChangeRatio: parseNum(row.ChangeRatio),
		}
		if bar.Close <= 0 {
			continue
		}
		byDate[date] = bar
	}

	bars := make([]IndexBar, 0, len(byDate))
	for _, bar := range byDate {
		bars = append(bars, bar)
	}
	sort.Slice(bars, func(i, j int) bool { return bars[i].Date < bars[j].Date })
	return &IndexPerfResult{IndexCode: indexCode, Bars: bars}, nil
}

// perfPayload 是指数日行情的响应外层。
type perfPayload struct {
	Code int `json:"code"`
	Data []struct {
		TradeDate   json.RawMessage `json:"tradeDate"`
		Open        json.RawMessage `json:"open"`
		High        json.RawMessage `json:"high"`
		Low         json.RawMessage `json:"low"`
		Close       json.RawMessage `json:"close"`
		Change      json.RawMessage `json:"change"`
		ChangeRatio json.RawMessage `json:"changeRatio"`
	} `json:"data"`
}

// normalizeDate 把 `20261113` / `2026-11-13` / `2026/11/13` 统一成 `2026-11-13`。
func normalizeDate(value json.RawMessage) string {
	text := cellText(value)
	digits := make([]byte, 0, 8)
	for i := 0; i < len(text); i++ {
		if text[i] >= '0' && text[i] <= '9' {
			digits = append(digits, text[i])
		}
	}
	if len(digits) < 8 {
		return ""
	}
	return string(digits[:4]) + "-" + string(digits[4:6]) + "-" + string(digits[6:8])
}

// compactDate 把 `2026-11-13` 压成 `20261113`；已是紧凑格式时原样返回。
func compactDate(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	digits := make([]byte, 0, 8)
	for i := 0; i < len(text); i++ {
		if text[i] >= '0' && text[i] <= '9' {
			digits = append(digits, text[i])
		}
	}
	if len(digits) < 8 {
		return text
	}
	return string(digits[:8])
}

// cellText 把 JSON 原始值转成文本：字符串去引号，数字取原文。
func cellText(value json.RawMessage) string {
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
func parseNum(value json.RawMessage) float64 {
	text := strings.TrimSpace(cellText(value))
	if text == "" || text == "null" || text == "-" {
		return 0
	}
	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0
	}
	return parsed
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
