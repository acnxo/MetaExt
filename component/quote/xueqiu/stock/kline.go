package stock

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/acnxo/MetaExt/component/quote/internal/fetch"
)

// 日线接口的固定参数。
const (
	// klinePath 日线接口路径。
	klinePath = "/v5/stock/chart/kline.json"
	// DefaultCount 单次返回的根数；负数表示向前取，8000 根覆盖 1999 年至今。
	DefaultCount = 8000
	// cstOffset 东八区偏移（秒）。
	cstOffset = 8 * 3600
)

// GetKLineRequest 是日线的请求参数。
type GetKLineRequest struct {
	// Symbol 带市场前缀的大写代码（如 `SH600000`）；非空时优先使用。
	Symbol string
	// Code 六位代码；Symbol 为空时按首位补前缀（6 → SH，其余 → SZ）。
	Code string
	// Count 返回根数，为空或非正时取 DefaultCount。
	Count int
}

// KLineBar 是一根日线。
//
// 价格与成交额单位为元（指数为点与元）；**成交量单位是「股」**，无需换算。
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
	// Volume 成交量（**股**）。
	Volume float64
	// Amount 成交额（**元**）。
	Amount float64
}

// KLineResult 是一段日线。
type KLineResult struct {
	// Symbol 带市场前缀的大写代码。
	Symbol string
	// Bars 日线，按日期升序。
	Bars []KLineBar
}

// GetKLine 拉取日线。
//
// 首次调用会先预热会话拿 cookie。若返回的 `item` 为空（典型是 cookie 失效），
// 会重建会话并重试一次；仍为空则返回错误，避免把「会话失效」误判成「无数据」。
func (c *Client) GetKLine(ctx context.Context, req GetKLineRequest) (*KLineResult, error) {
	symbol, err := resolveSymbol(req)
	if err != nil {
		return nil, err
	}
	count := req.Count
	if count <= 0 {
		count = DefaultCount
	}

	query := url.Values{}
	query.Set("symbol", symbol)
	query.Set("begin", strconv.FormatInt(time.Now().UnixMilli(), 10))
	query.Set("period", "day")
	query.Set("count", strconv.Itoa(-count))
	query.Set("indicator", "kline")
	query.Set("type", "normal")
	apiURL := c.cfg.BaseURL + klinePath + "?" + query.Encode()

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if err := c.warm(ctx); err != nil {
			return nil, wrapError(klinePath, apiURL, err)
		}
		raw, err := c.req.Get(ctx, apiURL, map[string]string{"referer": c.cfg.Referer})
		if err != nil {
			c.invalidateSession()
			lastErr = wrapError(klinePath, apiURL, err)
			continue
		}

		result, err := parseKLine(raw, symbol)
		if err != nil {
			c.invalidateSession()
			lastErr = err
			continue
		}
		if len(result.Bars) == 0 {
			c.invalidateSession()
			lastErr = &APIError{
				Path:    klinePath,
				URL:     apiURL,
				Message: "响应不含 K 线（会话可能失效）",
				Body:    truncateBody(raw),
			}
			continue
		}
		return result, nil
	}
	return nil, lastErr
}

// klinePayload 是日线响应的外层结构。
type klinePayload struct {
	ErrorCode        int    `json:"error_code"`
	ErrorDescription string `json:"error_description"`
	Data             struct {
		Symbol string   `json:"symbol"`
		Period string   `json:"period"`
		Column []string `json:"column"`
		Item   [][]any  `json:"item"`
	} `json:"data"`
}

// parseKLine 解析列式响应。
func parseKLine(raw []byte, symbol string) (*KLineResult, error) {
	var payload klinePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &APIError{
			Path:    klinePath,
			Message: "响应解析失败: " + err.Error(),
			Body:    truncateBody(raw),
			Err:     err,
		}
	}
	if payload.ErrorCode != 0 {
		return nil, &APIError{
			Path:    klinePath,
			Code:    payload.ErrorCode,
			Message: payload.ErrorDescription,
			Body:    truncateBody(raw),
		}
	}

	column := make(map[string]int, len(payload.Data.Column))
	for i, name := range payload.Data.Column {
		column[name] = i
	}

	bars := make([]KLineBar, 0, len(payload.Data.Item))
	for _, row := range payload.Data.Item {
		closePrice := cellNum(row, column, "close")
		if closePrice <= 0 {
			continue
		}
		bar := KLineBar{
			Date:   dateOf(cellNum(row, column, "timestamp")),
			Open:   fallbackNum(cellNum(row, column, "open"), closePrice),
			Close:  closePrice,
			High:   fallbackNum(cellNum(row, column, "high"), closePrice),
			Low:    fallbackNum(cellNum(row, column, "low"), closePrice),
			Volume: cellNum(row, column, "volume"),
			Amount: cellNum(row, column, "amount"),
		}
		if bar.Date == "" {
			continue
		}
		bars = append(bars, bar)
	}
	return &KLineResult{Symbol: symbol, Bars: bars}, nil
}

// cellValue 按列名取单元格，缺失时返回 nil。
func cellValue(row []any, column map[string]int, name string) any {
	index, ok := column[name]
	if !ok || index >= len(row) {
		return nil
	}
	return row[index]
}

// cellNum 按列名取数值，缺失或非法时返回 0。
func cellNum(row []any, column map[string]int, name string) float64 {
	switch value := cellValue(row, column, name).(type) {
	case float64:
		return value
	case json.Number:
		if parsed, err := value.Float64(); err == nil {
			return parsed
		}
	case string:
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
			return parsed
		}
	}
	return 0
}

// fallbackNum 为 0 时取默认值，用于 open/high/low 缺失的场景。
func fallbackNum(value, fallback float64) float64 {
	if value > 0 {
		return value
	}
	return fallback
}

// dateOf 把毫秒时间戳转成东八区的 yyyy-MM-dd。
func dateOf(ms float64) string {
	if ms <= 0 {
		return ""
	}
	moment := time.UnixMilli(int64(ms)).In(time.FixedZone("CST", cstOffset))
	return moment.Format("2006-01-02")
}

// resolveSymbol 计算请求用的带前缀代码。
//
// 前缀规则：6 开头为沪市、4/8 开头为北交所、其余为深市。
// 920 段（北交所新代码）与 B 股无法按首位区分，需要调用方显式传 Symbol。
func resolveSymbol(req GetKLineRequest) (string, error) {
	if symbol := strings.TrimSpace(req.Symbol); symbol != "" {
		return symbol, nil
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return "", errors.New("雪球 GetKLine：Symbol 与 Code 不能同时为空")
	}
	switch code[0] {
	case '6':
		return "SH" + code, nil
	case '4', '8':
		return "BJ" + code, nil
	default:
		return "SZ" + code, nil
	}
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
