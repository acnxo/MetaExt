package indexapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/acnxo/MetaExt/component/quote/internal/fetch"
)

// cstOffset 东八区偏移（秒）。
const cstOffset = 8 * 3600

// GetIndexKLineRequest 是指数日线的请求参数。
type GetIndexKLineRequest struct {
	// IndexCode windCode，如 `8841431.WI`；用于在 IndexIDs 中查官网编码。
	IndexCode string
	// IndexID 官网内部编码；非空时优先使用，可绕过映射表。
	IndexID string
	// Language 语言，为空取 DefaultLanguage。
	Language string
}

// IndexBar 是一根指数日线。
type IndexBar struct {
	// Date 交易日，yyyy-MM-dd。
	Date string
	// Open 开盘（点）。
	Open float64
	// High 最高（点）。
	High float64
	// Low 最低（点）。
	Low float64
	// Close 收盘（点）。
	Close float64
	// Volume 成交量（站点原始口径）。
	Volume float64
	// Amount 成交额（元）。
	Amount float64
	// ChangeRatio 涨跌幅（%）。
	ChangeRatio float64
}

// IndexKLineResult 是指数日线结果。
type IndexKLineResult struct {
	// IndexCode 请求的 windCode。
	IndexCode string
	// IndexID 实际使用的官网内部编码。
	IndexID string
	// Bars 日线，按日期升序、已去重。
	Bars []IndexBar
}

// GetIndexKLine 拉取指数日线。
//
// 站点只返回最近约 60 个交易日；调用方需自行与已有数据按日期合并。
// `Result` 为空时返回空 Bars 而不是错误（站点对部分编码会返回空结果）。
func (c *Client) GetIndexKLine(ctx context.Context, req GetIndexKLineRequest) (*IndexKLineResult, error) {
	indexCode := strings.TrimSpace(req.IndexCode)
	indexID := strings.TrimSpace(req.IndexID)
	if indexID == "" {
		if indexCode == "" {
			return nil, errors.New("万得官网 GetIndexKLine：IndexCode 与 IndexID 不能同时为空")
		}
		mapped, ok := c.cfg.IndexIDs[indexCode]
		if !ok {
			return nil, fmt.Errorf("万得官网 GetIndexKLine：未收录的 windCode %q，请显式传 IndexID", indexCode)
		}
		indexID = mapped
	}
	language := strings.TrimSpace(req.Language)
	if language == "" {
		language = DefaultLanguage
	}

	query := url.Values{}
	query.Set("indexId", indexID)
	query.Set("lan", language)
	apiURL := c.cfg.BaseURL + c.cfg.IndexKLinePath + "?" + query.Encode()

	raw, err := c.req.Get(ctx, apiURL, nil)
	if err != nil {
		return nil, wrapError(c.cfg.IndexKLinePath, apiURL, err)
	}

	var payload klinePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &APIError{
			Path:    c.cfg.IndexKLinePath,
			URL:     apiURL,
			Message: "响应解析失败: " + err.Error(),
			Body:    truncateBody(raw),
			Err:     err,
		}
	}

	byDate := make(map[string]IndexBar, len(payload.Result))
	for _, row := range payload.Result {
		date := dateOfMillis(parseNum(row.TradeDate))
		if date == "" {
			continue
		}
		bar := IndexBar{
			Date:  date,
			Open:  parseNum(row.Open),
			High:  parseNum(row.Hight), // 站点拼写为 hight
			Low:   parseNum(row.Low),
			Close: parseNum(row.Close),
			// 成交量与成交额用整数承载，缺失时按 0。
			Volume:      parseNum(row.Volume),
			Amount:      parseNum(row.Amount),
			ChangeRatio: parseNum(row.PctChange),
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
	return &IndexKLineResult{IndexCode: indexCode, IndexID: indexID, Bars: bars}, nil
}

// klinePayload 是指数日线的响应外层。
type klinePayload struct {
	Result []struct {
		TradeDate json.RawMessage `json:"tradeDate"`
		Open      json.RawMessage `json:"open"`
		Hight     json.RawMessage `json:"hight"`
		Low       json.RawMessage `json:"low"`
		Close     json.RawMessage `json:"close"`
		Volume    json.RawMessage `json:"volume"`
		Amount    json.RawMessage `json:"amount"`
		PctChange json.RawMessage `json:"pctChange"`
	} `json:"Result"`
}

// dateOfMillis 把毫秒时间戳转成东八区的 yyyy-MM-dd。
func dateOfMillis(ms float64) string {
	if ms <= 0 {
		return ""
	}
	moment := time.UnixMilli(int64(ms)).In(time.FixedZone("CST", cstOffset))
	return moment.Format("2006-01-02")
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

// truncateBody 截断响应体，避免错误信息过大。
func truncateBody(raw []byte) string {
	const limit = 300
	text := strings.TrimSpace(string(raw))
	if len(text) <= limit {
		return text
	}
	return fmt.Sprintf("%s...", text[:limit])
}
