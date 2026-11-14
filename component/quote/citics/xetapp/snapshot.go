package xetapp

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// 数值解析与名称序号剥离用的正则。
var (
	// numUnitRe 匹配「数值 + 可选单位」，如 -2.08 与 10.37亿。
	numUnitRe = regexp.MustCompile(`^(-?\d+(?:\.\d+)?)(万亿|亿|万)?`)
	// seqRe 匹配名称前的递增序号前缀，如 "1.幸福蓝海" 的 "1."。
	seqRe = regexp.MustCompile(`^\d+\.`)
)

// unitMultiplier 是单位到倍数的映射。
var unitMultiplier = map[string]float64{"万": 1e4, "亿": 1e8, "万亿": 1e12}

// SnapshotRequest 是行情快照的请求参数。
type SnapshotRequest struct {
	// Pool 池别名（见 Pools）或板块内部代码；为空取 "all"（沪深京全A）。
	Pool string
	// Fields 完全自定义的字段码；为空使用 DefaultFields（45 个）。
	// 传入后输出只包含这些字段。
	Fields []int
	// ExtraFields 在 Fields（或默认集）基础上追加的字段码，重复的自动忽略。
	ExtraFields []int
	// MaxCount 返回行数上限，为空取 DefaultMaxCount。
	// 该值必须显式传：站点在不传时只返回 2 行。
	MaxCount int
	// StartPos 分页起点，从 1 开始，是**成分序号**而非行号；0 表示不传。
	StartPos int
}

// DepthLevel 是五档行情的一档。
type DepthLevel struct {
	// Price 价格（元），原始响应以厘为单位，已换算。
	Price float64
	// Volume 数量（手）。
	Volume int64
}

// Row 是一条证券快照。
//
// 默认 45 个字段有强类型成员，直接取用即可；缺失或不适用的字段为零值——
// 想区分「未返回」与「0」，用 Values 查原始解析结果（未返回时该 key 不存在）。
type Row struct {
	// Name 名称，已剥离 "1." 这类序号前缀。码 1。
	Name string
	// Code 证券代码。码 32。
	Code string
	// Last 最新价（元）。码 2。
	Last float64
	// ChangePrice 涨跌额（元）。码 3。
	ChangePrice float64
	// ChangeRatio 涨跌幅（%）。码 4。
	ChangeRatio float64
	// PreClose 昨收（元）。码 5。
	PreClose float64
	// Open 今开（元）。码 6。
	Open float64
	// High 最高（元）。码 7。
	High float64
	// Low 最低（元）。码 8。
	Low float64
	// AvgPrice 均价（元）。码 9。
	AvgPrice float64
	// Volume 成交量（手）。码 10。
	Volume int64
	// Amount 成交额（元）。码 11。
	Amount float64
	// LastVolume 现手（手）。码 12。
	LastVolume int64
	// TurnoverRate 换手率（%）。码 13。
	TurnoverRate float64
	// BidAskRatio 委比（%）。码 14。
	BidAskRatio float64
	// BidAskDiff 委差（手，可为小数）。码 15。
	BidAskDiff float64
	// Amplitude 振幅（%）。码 16。
	Amplitude float64
	// OuterVolume 外盘（手）。码 48。
	OuterVolume int64
	// InnerVolume 内盘（手）。码 49。
	InnerVolume int64
	// VolumeRatio 量比。码 50。
	VolumeRatio float64
	// PE 市盈率（静态，亏损为负）。码 92。
	PE float64
	// PETTM 市盈率 TTM。码 306。
	PETTM float64
	// DynamicPE 动态市盈率。码 227。
	DynamicPE float64
	// PB 市净率。码 208。
	PB float64
	// BVPS 每股净资产（元）。码 228。
	BVPS float64
	// EPS 每股收益（元）。码 233。
	EPS float64
	// ROE 净资产收益率（%）。码 256。
	ROE float64
	// NetInflow 当日净流入（元）。码 162。
	NetInflow float64
	// SuspendFlag 停牌标志（0/1），语义以 TradeStatus 为准。码 209。
	SuspendFlag int64
	// TradeStatus 停牌状态，如「正常交易」。码 210。
	TradeStatus string
	// LimitUpPrice 涨停板价（元）。码 225。
	LimitUpPrice float64
	// LimitDownPrice 跌停板价（元）。码 226。
	LimitDownPrice float64
	// TotalShares 总股本（股）。码 229。
	TotalShares int64
	// FloatShares 流通股本（股）。码 231。
	FloatShares int64
	// MarketCap 总市值（元）。码 230。
	MarketCap float64
	// FloatMarketCap 流通市值（元）。码 232。
	FloatMarketCap float64
	// DividendYield 股息率（%）。码 335。
	DividendYield float64
	// DividendYieldTTM 股息率 TTM，返回值形如 0.04。码 453。
	DividendYieldTTM float64
	// Dividend 每股股息（元）。码 408。
	Dividend float64
	// ChangePct5D 5 日涨跌幅（%）。码 298。
	ChangePct5D float64
	// ChangePct20D 20 日涨跌幅（%）。码 294。
	ChangePct20D float64
	// ChangePct60D 60 日涨跌幅（%）。码 295。
	ChangePct60D float64
	// ChangePct120D 120 日涨跌幅（%）。码 296。
	ChangePct120D float64
	// ChangePct250D 250 日涨跌幅（%）。码 297。
	ChangePct250D float64
	// ChangePctWeek 本周涨幅（%），实测与 5 日同值。码 342。
	ChangePctWeek float64

	// Values 保存本次请求所有字段的解析结果，key 为英文字段名。
	//
	// 未命名的字段码用 f<code>；五档（码 93–102 与 180–189）汇总在 "depth"
	// 下，类型为 map[string]DepthLevel。
	Values map[string]any
}

// SnapshotResult 是一次快照请求的结果。
type SnapshotResult struct {
	// Rows 成分股快照，已剔除池自身那一行。
	Rows []Row
	// PoolCode 池的内部代码（响应回显或请求值）。
	PoolCode string
	// PoolName 池名称，如「沪深京全A」。
	PoolName string
	// MaxCount 成分总数（不是返回行数）。
	MaxCount int
	// NeedCheck 服务端实际生效的字段码，列顺序以它为准。
	NeedCheck []int
	// HQDate 行情日期 yyyyMMdd。
	HQDate string
	// ServerDateTime 服务器时间。
	ServerDateTime string
	// ErrorNo 响应里的 ERRORNO 原值；正值是正常的，等于返回行数 + 2。
	ErrorNo int
}

// snapshotResponse 是网关响应的原始结构。
type snapshotResponse struct {
	Grid0          []string `json:"GRID0"`
	Action         string   `json:"ACTION"`
	MaxCount       string   `json:"MAXCOUNT"`
	ErrorNo        string   `json:"ERRORNO"`
	ErrorMessage   string   `json:"ERRORMESSAGE"`
	NeedCheck      string   `json:"NEEDCHECK"`
	StockCode      string   `json:"STOCKCODE"`
	StockName      string   `json:"STOCKNAME"`
	HQDate         string   `json:"HQDATE"`
	ServerDateTime string   `json:"SERVERDATETIME"`
	MarketStatus   string   `json:"MARKETSTATUS"`
}

// GetQuoteSnapshotCNAStock 拉取 A 股行情快照。
//
// 一次请求返回整个池的成分快照，不需要分页。返回前会剔除「池自身」那一行
// （其代码是内部码，如请求 000300 回的 1A0300）。
//
// 响应结构异常、ERRORNO 为负值、无 GRID0 数据时返回 *APIError，
// 网络错误与响应截断会自动重试。
func (c *Client) GetQuoteSnapshotCNAStock(ctx context.Context, req SnapshotRequest) (*SnapshotResult, error) {
	poolCode := resolvePool(req.Pool)
	fields := wantedFields(req)

	// 内部固定多取「名称」与「代码」，用于识别并剔除池自身那一行；返回前再过滤掉。
	fetch := append([]int(nil), fields...)
	for _, code := range []int{1, 32} {
		if !containsInt(fetch, code) {
			fetch = append(fetch, code)
		}
	}

	form := url.Values{}
	form.Set("action", ActionQuoteSnapshot)
	form.Set("ReqlinkType", "0")
	form.Set("StockCode", poolCode)
	form.Set("MaxCount", strconv.Itoa(defaultInt(req.MaxCount, DefaultMaxCount)))
	if req.StartPos > 0 {
		form.Set("StartPos", strconv.Itoa(req.StartPos))
	}
	form.Set("NeedCheck", formatNeedCheck(fetch))

	var payload snapshotResponse
	if err := c.postJSON(ctx, form, &payload); err != nil {
		return nil, err
	}
	result, err := decodeSnapshot(&payload, poolCode)
	if err != nil {
		return nil, err
	}
	// 多取的内部字段只在调用方没要时才剔除。
	if len(fetch) != len(fields) {
		wanted := make(map[string]struct{}, len(fields))
		for _, code := range fields {
			wanted[fieldKey(code)] = struct{}{}
			if fieldKind(code) == KindDepth {
				wanted["depth"] = struct{}{}
			}
		}
		for i := range result.Rows {
			for key := range result.Rows[i].Values {
				if _, ok := wanted[key]; !ok {
					delete(result.Rows[i].Values, key)
				}
			}
		}
	}
	return result, nil
}

// decodeSnapshot 校验 ERRORNO 与数据，并解析 GRID0。
func decodeSnapshot(payload *snapshotResponse, poolCode string) (*SnapshotResult, error) {
	errNo, _ := strconv.Atoi(strings.TrimSpace(payload.ErrorNo))
	if errNo < 0 {
		return nil, &APIError{ErrorNo: errNo, Message: payload.ErrorMessage}
	}
	if len(payload.Grid0) == 0 {
		return nil, &APIError{
			Message: fmt.Sprintf("无数据：ERRORNO=%s %s", payload.ErrorNo, payload.ErrorMessage),
		}
	}
	maxCount, _ := strconv.Atoi(strings.TrimSpace(payload.MaxCount))
	result := &SnapshotResult{
		PoolCode:       firstNonEmpty(payload.StockCode, poolCode),
		PoolName:       strings.TrimSpace(payload.StockName),
		MaxCount:       maxCount,
		NeedCheck:      parseNeedCheck(payload.NeedCheck),
		HQDate:         payload.HQDate,
		ServerDateTime: payload.ServerDateTime,
		ErrorNo:        errNo,
	}
	rows, err := parseGrid(payload.Grid0, result.NeedCheck, result.PoolCode, result.PoolName)
	if err != nil {
		return nil, err
	}
	result.Rows = rows
	return result, nil
}

// parseGrid 把 GRID0 解析成结构化行。
//
// 列顺序以 NEEDCHECK 回显为准；码 33–42（卖五…买五）每码占两段「价格|数量」，
// 其余码各占一段。表头与记录都以竖线结尾，切分后需去掉末尾空元素。
func parseGrid(grid []string, order []int, ownCode, ownName string) ([]Row, error) {
	if len(order) == 0 {
		return nil, &APIError{Message: "响应缺少 NEEDCHECK，无法确定列顺序"}
	}
	rows := make([]Row, 0, len(grid)-1)
	for _, record := range grid[1:] {
		cells := trimTrailingEmpty(strings.Split(record, "|"))
		raw := make(map[int]string, len(order))
		index := 0
		for _, code := range order {
			if code >= 33 && code <= 42 {
				price := cellAt(cells, index)
				volume := cellAt(cells, index+1)
				raw[code] = price + "|" + volume
				index += 2
				continue
			}
			raw[code] = cellAt(cells, index)
			index++
		}
		// 池自身那一行：代码等于内部码，或名称与池名一致。
		if raw[32] == ownCode || (ownName != "" && stripSeq(raw[1]) == ownName) {
			continue
		}
		if name, ok := raw[1]; ok {
			raw[1] = stripSeq(name)
		}
		rows = append(rows, buildRow(raw))
	}
	return rows, nil
}

// buildRow 把「字段码 → 原始字符串」转成 Row。
func buildRow(raw map[int]string) Row {
	row := Row{Values: make(map[string]any, len(raw))}
	codes := make([]int, 0, len(raw))
	for code := range raw {
		codes = append(codes, code)
	}
	sort.Ints(codes)

	depth := make(map[string]DepthLevel, 10)
	for _, code := range codes {
		key := fieldKey(code)
		kind := fieldKind(code)
		if kind == KindDepth {
			price, volume, _ := strings.Cut(raw[code], "|")
			level := DepthLevel{}
			if value := convertValue(price, KindPrice); value != nil {
				// 五档价格原始单位是厘，换算成元。
				level.Price = math.Round(value.(float64)/1000*1e4) / 1e4
			}
			if value := convertValue(volume, KindVol); value != nil {
				level.Volume = value.(int64)
			}
			depth[key] = level
			continue
		}
		value := convertValue(raw[code], kind)
		if value == nil && strings.HasPrefix(key, "f") {
			// 未命名字段为空时不输出，避免结果里堆满 nil。
			continue
		}
		row.Values[key] = value
	}
	if len(depth) > 0 {
		row.Values["depth"] = depth
	}
	fillFields(&row, row.Values)
	return row
}

// fillFields 把常用字段填进强类型成员，省去调用方做类型断言。
func fillFields(row *Row, values map[string]any) {
	row.Name = asString(values["name"])
	row.Code = asString(values["code"])
	row.Last = asFloat(values["last"])
	row.ChangePrice = asFloat(values["change_price"])
	row.ChangeRatio = asFloat(values["change_ratio"])
	row.PreClose = asFloat(values["pre_close"])
	row.Open = asFloat(values["open"])
	row.High = asFloat(values["high"])
	row.Low = asFloat(values["low"])
	row.AvgPrice = asFloat(values["avg_price"])
	row.Volume = asInt(values["volume"])
	row.Amount = asFloat(values["amount"])
	row.LastVolume = asInt(values["last_volume"])
	row.TurnoverRate = asFloat(values["turnover_rate"])
	row.BidAskRatio = asFloat(values["bid_ask_ratio"])
	row.BidAskDiff = asFloat(values["bid_ask_diff"])
	row.Amplitude = asFloat(values["amplitude"])
	row.OuterVolume = asInt(values["outer_volume"])
	row.InnerVolume = asInt(values["inner_volume"])
	row.VolumeRatio = asFloat(values["volume_ratio"])
	row.PE = asFloat(values["pe"])
	row.PETTM = asFloat(values["pe_ttm"])
	row.DynamicPE = asFloat(values["dynamic_pe"])
	row.PB = asFloat(values["pb"])
	row.BVPS = asFloat(values["bvps"])
	row.EPS = asFloat(values["eps"])
	row.ROE = asFloat(values["roe"])
	row.NetInflow = asFloat(values["net_inflow"])
	row.SuspendFlag = asInt(values["suspend_flag"])
	row.TradeStatus = asString(values["trade_status"])
	row.LimitUpPrice = asFloat(values["limit_up_price"])
	row.LimitDownPrice = asFloat(values["limit_down_price"])
	row.TotalShares = asInt(values["total_shares"])
	row.FloatShares = asInt(values["float_shares"])
	row.MarketCap = asFloat(values["market_cap"])
	row.FloatMarketCap = asFloat(values["float_market_cap"])
	row.DividendYield = asFloat(values["dividend_yield"])
	row.DividendYieldTTM = asFloat(values["dividend_yield_ttm"])
	row.Dividend = asFloat(values["dividend"])
	row.ChangePct5D = asFloat(values["change_pct_5d"])
	row.ChangePct20D = asFloat(values["change_pct_20d"])
	row.ChangePct60D = asFloat(values["change_pct_60d"])
	row.ChangePct120D = asFloat(values["change_pct_120d"])
	row.ChangePct250D = asFloat(values["change_pct_250d"])
	row.ChangePctWeek = asFloat(values["change_pct_week"])
}

// convertValue 把原始字符串转成 Go 值；`--`、空串与 `-` 转成 nil。
func convertValue(raw string, kind FieldKind) any {
	text := strings.TrimSpace(raw)
	if text == "" || text == "--" || text == "-" {
		return nil
	}
	switch kind {
	case KindStr, KindCode, KindBinary:
		return text
	case KindPct:
		return numberValue(strings.TrimSuffix(text, "%"), false)
	}
	// int / flag / vol / shares 取整，其余按浮点处理。
	asInteger := kind == KindInt || kind == KindFlag || kind == KindVol || kind == KindShares
	if value := numberValue(text, asInteger); value != nil {
		return value
	}
	// 无法解析的保持原文，避免把说明性文本吃掉。
	return text
}

// numberValue 解析「数值 + 可选单位」，按需转成 int64。
func numberValue(text string, asInteger bool) any {
	match := numUnitRe.FindStringSubmatch(text)
	if match == nil {
		return nil
	}
	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return nil
	}
	if unit := match[2]; unit != "" {
		value *= unitMultiplier[unit]
	}
	if asInteger {
		return int64(math.Round(value))
	}
	return value
}

// resolvePool 把池别名换成内部代码；非别名原样当作内部代码。
func resolvePool(pool string) string {
	name := strings.TrimSpace(pool)
	if name == "" {
		return DefaultPoolCode
	}
	if code, ok := Pools[name]; ok {
		return code
	}
	return name
}

// wantedFields 合并 Fields 与 ExtraFields，去掉重复项。
func wantedFields(req SnapshotRequest) []int {
	fields := req.Fields
	if len(fields) == 0 {
		fields = DefaultFields
	}
	out := make([]int, 0, len(fields)+len(req.ExtraFields))
	for _, code := range fields {
		if !containsInt(out, code) {
			out = append(out, code)
		}
	}
	for _, code := range req.ExtraFields {
		if !containsInt(out, code) {
			out = append(out, code)
		}
	}
	return out
}

// formatNeedCheck 把字段码拼成竖线分隔串，末尾同样带竖线（与站点一致）。
func formatNeedCheck(codes []int) string {
	var b strings.Builder
	for _, code := range codes {
		b.WriteString(strconv.Itoa(code))
		b.WriteByte('|')
	}
	return b.String()
}

// parseNeedCheck 解析响应回显的字段码串。
func parseNeedCheck(text string) []int {
	parts := strings.Split(text, "|")
	codes := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if code, err := strconv.Atoi(part); err == nil {
			codes = append(codes, code)
		}
	}
	return codes
}

// stripSeq 去掉名称前的递增序号前缀，如 "1.幸福蓝海" → "幸福蓝海"。
func stripSeq(name string) string {
	return strings.TrimSpace(seqRe.ReplaceAllString(name, ""))
}

// cellAt 取下标处的单元格，越界时给 `--`（个别标的会少段）。
func cellAt(cells []string, index int) string {
	if index >= 0 && index < len(cells) {
		return cells[index]
	}
	return "--"
}

// containsInt 判断切片是否包含指定整数。
func containsInt(items []int, target int) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

// defaultInt 为空或非正时取默认值。
func defaultInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

// firstNonEmpty 返回第一个非空字符串。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// asFloat 把解析结果转成 float64。
func asFloat(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

// asInt 把解析结果转成 int64。
func asInt(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

// asString 把解析结果转成字符串。
func asString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
