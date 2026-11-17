package kline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/acnxo/MetaExt/component/quote/internal/fetch"
)

// GetKLineRequest 是日线的请求参数。
type GetKLineRequest struct {
	// Symbol 带市场前缀的代码（如 `sz399303`）；非空时优先使用。
	Symbol string
	// Code 六位代码；Symbol 为空时按首位补前缀（6 → sh，其余 → sz）。
	Code string
	// Scale 周期（分钟数），为空或非正时取 DefaultScale。
	Scale int
	// DataLen 返回根数，为空或非正时取 DefaultDataLen。
	DataLen int
}

// KLineBar 是一根日线。
//
// 价格单位随标的（股票为元、指数为点）；**成交量原始单位是「股」**，不做换算。
type KLineBar struct {
	// Date 交易日，yyyy-MM-dd。
	Date string
	// Open 开盘。
	Open float64
	// High 最高。
	High float64
	// Low 最低。
	Low float64
	// Close 收盘。
	Close float64
	// Volume 成交量（**股**）。
	Volume float64
}

// KLineResult 是一段日线。
type KLineResult struct {
	// Symbol 带市场前缀的代码。
	Symbol string
	// Bars 日线，按返回顺序（通常升序）。
	Bars []KLineBar
}

// GetKLine 拉取日线。
//
// 该接口不返回成交额。返回空数组时视为无数据，Bars 为空切片而不是错误；
// 响应不是合法 JSON（站点限频返回 `null` 或错误页）时返回 *APIError。
func (c *Client) GetKLine(ctx context.Context, req GetKLineRequest) (*KLineResult, error) {
	symbol, err := resolveSymbol(req)
	if err != nil {
		return nil, err
	}
	scale := req.Scale
	if scale <= 0 {
		scale = DefaultScale
	}
	dataLen := req.DataLen
	if dataLen <= 0 {
		dataLen = DefaultDataLen
	}

	query := url.Values{}
	query.Set("symbol", symbol)
	query.Set("scale", strconv.Itoa(scale))
	query.Set("ma", "no")
	query.Set("datalen", strconv.Itoa(dataLen))
	apiURL := c.cfg.BaseURL + c.cfg.KLinePath + "?" + query.Encode()

	raw, err := c.req.Get(ctx, apiURL, map[string]string{"referer": c.cfg.Referer})
	if err != nil {
		return nil, wrapError(c.cfg.KLinePath, apiURL, err)
	}

	// 只在响应确实是数组时才接受：站点限频会返回 `null`，不能当成「无数据」。
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, &APIError{
			Path:    c.cfg.KLinePath,
			URL:     apiURL,
			Message: "响应不是数组（可能被限频）",
			Body:    truncateBody(raw),
		}
	}

	var rows []klineRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, &APIError{
			Path:    c.cfg.KLinePath,
			URL:     apiURL,
			Message: "响应解析失败: " + err.Error(),
			Body:    truncateBody(raw),
			Err:     err,
		}
	}

	bars := make([]KLineBar, 0, len(rows))
	for _, row := range rows {
		bar := KLineBar{
			Date:   strings.TrimSpace(row.Day),
			Open:   parseNum(row.Open),
			High:   parseNum(row.High),
			Low:    parseNum(row.Low),
			Close:  parseNum(row.Close),
			Volume: parseNum(row.Volume),
		}
		if bar.Date == "" || bar.Close <= 0 {
			continue
		}
		bars = append(bars, bar)
	}
	return &KLineResult{Symbol: symbol, Bars: bars}, nil
}

// klineRow 是响应数组中的一个元素；数值以字符串返回。
type klineRow struct {
	Day    string `json:"day"`
	Open   string `json:"open"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Close  string `json:"close"`
	Volume string `json:"volume"`
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
		return "", errors.New("新浪 GetKLine：Symbol 与 Code 不能同时为空")
	}
	switch code[0] {
	case '6':
		return "sh" + code, nil
	case '4', '8':
		return "bj" + code, nil
	default:
		return "sz" + code, nil
	}
}

// parseNum 解析数值文本，缺失或非法时返回 0。
func parseNum(text string) float64 {
	text = strings.TrimSpace(text)
	if text == "" || text == "null" {
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
