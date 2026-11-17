package gtimg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// klinePath 日线接口路径。
const klinePath = "/appstock/app/fqkline/get"

// GetKLineRequest 是日线的请求参数。
type GetKLineRequest struct {
	// Symbol 带市场前缀的代码（如 `sh600000`）；非空时优先使用。
	Symbol string
	// Code 六位代码；Symbol 为空时按首位推断市场（6 → sh，其余 → sz）。
	Code string
	// Begin 起始日期 yyyy-MM-dd；为空时由站点决定。
	Begin string
	// End 结束日期 yyyy-MM-dd；为空时由站点决定。
	End string
	// Count 返回根数上限，为空或非正时取 1000。
	Count int
	// Adjust 为 true 时请求前复权（qfq），否则不复权。
	Adjust bool
}

// KLineBar 是一根日线。
//
// 价格单位随标的（股票为元、指数为点），**成交量原始单位是「手」**。
type KLineBar struct {
	// Date 交易日，yyyy-MM-dd。
	Date string
	// Open 开盘。
	Open float64
	// Close 收盘。
	Close float64
	// High 最高。
	High float64
	// Low 最低。
	Low float64
	// Volume 成交量（**手**）。
	Volume float64
}

// KLineResult 是一段日线。
type KLineResult struct {
	// Symbol 带市场前缀的代码。
	Symbol string
	// Bars 日线，按站点返回顺序（通常升序）。
	Bars []KLineBar
}

// GetKLine 拉取日线。
//
// 请求前复权时站点返回 `qfqday`，否则返回 `day`；两者都缺失时视为无数据，
// 返回空 Bars 而不是错误。
func (c *Client) GetKLine(ctx context.Context, req GetKLineRequest) (*KLineResult, error) {
	symbol, err := resolveSymbol(req)
	if err != nil {
		return nil, err
	}
	count := req.Count
	if count <= 0 {
		count = 1000
	}
	adjust := ""
	if req.Adjust {
		adjust = "qfq"
	}

	param := strings.Join([]string{
		symbol, "day", strings.TrimSpace(req.Begin), strings.TrimSpace(req.End),
		strconv.Itoa(count), adjust,
	}, ",")
	apiURL := c.cfg.KLineBaseURL + klinePath + "?param=" + url.QueryEscape(param)

	raw, err := c.req.Get(ctx, apiURL, map[string]string{"referer": c.cfg.Referer})
	if err != nil {
		return nil, wrapError(klinePath, apiURL, err)
	}

	var payload klinePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &APIError{
			Path:    klinePath,
			URL:     apiURL,
			Message: "响应解析失败: " + err.Error(),
			Body:    truncateBody(raw),
			Err:     err,
		}
	}
	node, ok := payload.Data[symbol]
	if !ok {
		return &KLineResult{Symbol: symbol, Bars: []KLineBar{}}, nil
	}
	rows := node.QFQDay
	if !req.Adjust || len(rows) == 0 {
		if len(node.Day) > 0 {
			rows = node.Day
		}
	}

	bars := make([]KLineBar, 0, len(rows))
	for _, row := range rows {
		bar, ok := decodeKLineRow(row)
		if !ok {
			continue
		}
		bars = append(bars, bar)
	}
	return &KLineResult{Symbol: symbol, Bars: bars}, nil
}

// klinePayload 是日线响应的外层结构。
type klinePayload struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data map[string]struct {
		QFQDay [][]json.RawMessage `json:"qfqday"`
		Day    [][]json.RawMessage `json:"day"`
	} `json:"data"`
}

// decodeKLineRow 解析一行 `[日期, 开, 收, 高, 低, 量]`。
func decodeKLineRow(row []json.RawMessage) (KLineBar, bool) {
	if len(row) < 6 {
		return KLineBar{}, false
	}
	bar := KLineBar{
		Date:   cellText(row[0]),
		Open:   parseNum(cellText(row[1])),
		Close:  parseNum(cellText(row[2])),
		High:   parseNum(cellText(row[3])),
		Low:    parseNum(cellText(row[4])),
		Volume: parseNum(cellText(row[5])),
	}
	if len(bar.Date) < 10 || bar.Close <= 0 {
		return KLineBar{}, false
	}
	return bar, true
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

// resolveSymbol 计算请求用的带前缀代码。
func resolveSymbol(req GetKLineRequest) (string, error) {
	if symbol := strings.TrimSpace(req.Symbol); symbol != "" {
		return symbol, nil
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return "", errors.New("腾讯 GetKLine：Symbol 与 Code 不能同时为空")
	}
	if symbol := symbolOf(code); symbol != "" {
		return symbol, nil
	}
	return "", fmt.Errorf("腾讯 GetKLine：无法从 %q 推断市场前缀", code)
}
