package push2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// 历史 K 线接口的固定参数。
const (
	// klinePath 历史 K 线接口路径。
	klinePath = "/api/qt/stock/kline/get"
	// DefaultUT 站点前端的公开固定参数，不是用户凭据。
	DefaultUT = "fa5fd1943c7b386f172d6893dbfba10b"
	// DefaultCalendarSecID 取交易日历时使用的证券（上证指数）。
	DefaultCalendarSecID = "1.000001"
	// DefaultCalendarLimit 交易日历默认返回的根数。
	DefaultCalendarLimit = 260
	// AdjustNone 不复权。
	AdjustNone = 0
	// klineFields 请求的 K 线字段：日期、开、收、高、低、量(手)、额(元)。
	klineFields = "f51,f52,f53,f54,f55,f56,f57"
)

// GetKLineRequest 是日线的请求参数。
type GetKLineRequest struct {
	// SecID 证券标识，形如 `1.000001`；非空时优先使用。
	SecID string
	// Code 六位代码；仅在 SecID 为空时使用，市场按首位推断（6 → 沪市，其余 → 深市）。
	// 指数等「代码无法区分市场」的标的请直接用 SecID，例如上证指数 `1.000001`。
	Code string
	// Begin 起始日期 yyyyMMdd；为空时不传。
	Begin string
	// End 结束日期 yyyyMMdd；为空时不传。
	End string
	// Limit 返回条数上限；为空或非正时不传。
	Limit int
	// Adjust 复权方式，`AdjustNone`(0) 为不复权；其它取值本包未验证。
	Adjust int
	// UT 站点公开固定参数，为空时取 DefaultUT。
	UT string
}

// KLineBar 是一根日线。
//
// 价格与成交额为元，**成交量原始单位是「手」**，与本包保持一致、不做 ×100 换算。
type KLineBar struct {
	// Date 交易日，yyyy-MM-dd。
	Date string
	// Open 开盘（元）。
	Open float64
	// Close 收盘（元）。
	Close float64
	// High 最高（元）。
	High float64
	// Low 最低（元）。
	Low float64
	// Volume 成交量（**手**）。
	Volume float64
	// Amount 成交额（元）。
	Amount float64
}

// KLineResult 是一段日线。
type KLineResult struct {
	// Code 证券代码。
	Code string
	// Name 证券名称。
	Name string
	// Bars 日线，按日期升序。
	Bars []KLineBar
}

// GetKLine 拉取日线。
//
// 只返回站点给出的区间内的数据；区间内无数据时 Bars 为空切片而不是错误。
// 镜像域名会依次轮换，`data` 为 null 时视为被限频并换下一个镜像。
func (c *Client) GetKLine(ctx context.Context, req GetKLineRequest) (*KLineResult, error) {
	secID, err := resolveSecID(req)
	if err != nil {
		return nil, err
	}
	adjust := req.Adjust
	ut := strings.TrimSpace(req.UT)
	if ut == "" {
		ut = DefaultUT
	}

	query := url.Values{}
	query.Set("secid", secID)
	query.Set("ut", ut)
	query.Set("fields1", "f1,f2,f3,f4,f5,f6")
	query.Set("fields2", klineFields)
	query.Set("klt", "101")
	query.Set("fqt", strconv.Itoa(adjust))
	if begin := strings.TrimSpace(req.Begin); begin != "" {
		query.Set("beg", begin)
	}
	if end := strings.TrimSpace(req.End); end != "" {
		query.Set("end", end)
	}
	if req.Limit > 0 {
		query.Set("lmt", strconv.Itoa(req.Limit))
	}

	raw, apiURL, err := c.getRaw(ctx, c.cfg.KLineHosts, klinePath, query, hasData)
	if err != nil {
		return nil, err
	}

	var payload klinePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, wrapParseError(klinePath, apiURL, raw, err)
	}
	if payload.Data == nil {
		return nil, &APIError{Path: klinePath, URL: apiURL, Message: "响应缺少 data", Body: truncateBody(raw)}
	}

	bars := make([]KLineBar, 0, len(payload.Data.KLines))
	for _, line := range payload.Data.KLines {
		bar, ok := parseKLine(line)
		if !ok {
			continue
		}
		bars = append(bars, bar)
	}
	return &KLineResult{Code: payload.Data.Code, Name: payload.Data.Name, Bars: bars}, nil
}

// GetTradingCalendar 用上证指数日线取交易日序列（升序，yyyy-MM-dd）。
//
// limit 为空或非正时取 DefaultCalendarLimit。该接口比打包节假日表更可靠，
// 代价是每次多一次请求，调用方可自行缓存。
func (c *Client) GetTradingCalendar(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = DefaultCalendarLimit
	}
	begin := time.Now().AddDate(-3, 0, 0).Format("20060102")
	result, err := c.GetKLine(ctx, GetKLineRequest{
		SecID: DefaultCalendarSecID,
		Begin: begin,
		End:   "20500101",
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}
	dates := make([]string, 0, len(result.Bars))
	for _, bar := range result.Bars {
		if len(bar.Date) == 10 {
			dates = append(dates, bar.Date)
		}
	}
	if len(dates) == 0 {
		return nil, &APIError{Path: klinePath, Message: "交易日历为空"}
	}
	return dates, nil
}

// klinePayload 是日线响应的外层结构。
type klinePayload struct {
	RC   int `json:"rc"`
	Data *struct {
		Code   string   `json:"code"`
		Name   string   `json:"name"`
		KLines []string `json:"klines"`
	} `json:"data"`
}

// parseKLine 解析一行 `日期,开,收,高,低,量,额`。
func parseKLine(line string) (KLineBar, bool) {
	fields := strings.Split(line, ",")
	if len(fields) < 7 {
		return KLineBar{}, false
	}
	bar := KLineBar{
		Date:   strings.TrimSpace(fields[0]),
		Open:   parseNum(fields[1]),
		Close:  parseNum(fields[2]),
		High:   parseNum(fields[3]),
		Low:    parseNum(fields[4]),
		Volume: parseNum(fields[5]),
		Amount: parseNum(fields[6]),
	}
	if bar.Date == "" || bar.Close <= 0 {
		return KLineBar{}, false
	}
	return bar, true
}

// resolveSecID 计算请求用的 secid。
//
// 东财的市场号：1 为沪市，0 为深市与北交所（北交所在东财同属 0 市场）。
func resolveSecID(req GetKLineRequest) (string, error) {
	if secID := strings.TrimSpace(req.SecID); secID != "" {
		return secID, nil
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return "", errors.New("东财 GetKLine：SecID 与 Code 不能同时为空")
	}
	market := 0
	if code[0] == '6' {
		market = 1
	}
	return fmt.Sprintf("%d.%s", market, code), nil
}
