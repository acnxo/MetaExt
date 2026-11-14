package xetapp

import "strconv"

// FieldKind 是字段值的转换方式，决定原始字符串怎么变成 Go 值。
type FieldKind string

// 字段值类型。
const (
	KindAmount FieldKind = "amount"
	KindAuto   FieldKind = "auto"
	KindBinary FieldKind = "binary"
	KindCode   FieldKind = "code"
	KindDepth  FieldKind = "depth"
	KindFlag   FieldKind = "flag"
	KindInt    FieldKind = "int"
	KindPct    FieldKind = "pct"
	KindPrice  FieldKind = "price"
	KindShares FieldKind = "shares"
	KindStr    FieldKind = "str"
	KindVol    FieldKind = "vol"
)

// fieldSpec 描述一个字段码：英文 key、值类型与中文列名。
type fieldSpec struct {
	// Key 输出用的英文字段名。
	Key string
	// Kind 值转换方式。
	Kind FieldKind
	// Label 中文列名，仅用于文档与排障。
	Label string
}

// fieldSpecs 是内嵌的字段规格表，覆盖 207 个已命名编码；
// 未列出的码按 f<code> + auto 处理，值为空时不出现在结果里。
var fieldSpecs = map[int]fieldSpec{
	1:   {Key: "name", Kind: KindStr, Label: "名称"},
	2:   {Key: "last", Kind: KindPrice, Label: "最新"},
	3:   {Key: "change_price", Kind: KindPrice, Label: "涨跌"},
	4:   {Key: "change_ratio", Kind: KindPct, Label: "幅度"},
	5:   {Key: "pre_close", Kind: KindPrice, Label: "昨收"},
	6:   {Key: "open", Kind: KindPrice, Label: "开盘"},
	7:   {Key: "high", Kind: KindPrice, Label: "最高"},
	8:   {Key: "low", Kind: KindPrice, Label: "最低"},
	9:   {Key: "avg_price", Kind: KindPrice, Label: "均价"},
	10:  {Key: "volume", Kind: KindVol, Label: "总手"},
	11:  {Key: "amount", Kind: KindAmount, Label: "总额"},
	12:  {Key: "last_volume", Kind: KindVol, Label: "现手"},
	13:  {Key: "turnover_rate", Kind: KindPct, Label: "换手"},
	14:  {Key: "bid_ask_ratio", Kind: KindPct, Label: "委比"},
	15:  {Key: "bid_ask_diff", Kind: KindAuto, Label: "委差"},
	16:  {Key: "amplitude", Kind: KindPct, Label: "振幅"},
	17:  {Key: "net_value", Kind: KindPrice, Label: "净值"},
	18:  {Key: "pre_settle", Kind: KindPrice, Label: "前结"},
	19:  {Key: "unit_net_value", Kind: KindPrice, Label: "单位净值"},
	20:  {Key: "acc_net_value", Kind: KindPrice, Label: "累计净值"},
	21:  {Key: "growth_value", Kind: KindPrice, Label: "增长值"},
	22:  {Key: "nav_date", Kind: KindStr, Label: "增长率"},
	23:  {Key: "date", Kind: KindStr, Label: "日期"},
	24:  {Key: "growth_date", Kind: KindStr, Label: "增长值日期"},
	25:  {Key: "prev_unit_net_value", Kind: KindPrice, Label: "前日单位净值"},
	26:  {Key: "prev_acc_net_value", Kind: KindPrice, Label: "前日累计净值"},
	27:  {Key: "fund_company", Kind: KindStr, Label: "公司名称"},
	28:  {Key: "fund_type", Kind: KindStr, Label: "类型"},
	29:  {Key: "subscribe_status", Kind: KindStr, Label: "申购状态"},
	30:  {Key: "redeem_status", Kind: KindStr, Label: "赎回状态"},
	31:  {Key: "closed_period", Kind: KindStr, Label: "是否封闭期"},
	32:  {Key: "code", Kind: KindCode, Label: "代码"},
	33:  {Key: "ask5", Kind: KindDepth, Label: "卖五"},
	34:  {Key: "ask4", Kind: KindDepth, Label: "卖四"},
	35:  {Key: "ask3", Kind: KindDepth, Label: "卖三"},
	36:  {Key: "ask2", Kind: KindDepth, Label: "卖二"},
	37:  {Key: "ask1", Kind: KindDepth, Label: "卖一"},
	38:  {Key: "bid1", Kind: KindDepth, Label: "买一"},
	39:  {Key: "bid2", Kind: KindDepth, Label: "买二"},
	40:  {Key: "bid3", Kind: KindDepth, Label: "买三"},
	41:  {Key: "bid4", Kind: KindDepth, Label: "买四"},
	42:  {Key: "bid5", Kind: KindDepth, Label: "买五"},
	43:  {Key: "up_count", Kind: KindInt, Label: "上涨"},
	44:  {Key: "down_count", Kind: KindInt, Label: "下跌"},
	45:  {Key: "flat_count", Kind: KindInt, Label: "平盘"},
	46:  {Key: "bid_volume", Kind: KindInt, Label: "委买"},
	47:  {Key: "ask_volume", Kind: KindInt, Label: "委卖"},
	48:  {Key: "outer_volume", Kind: KindVol, Label: "外盘"},
	49:  {Key: "inner_volume", Kind: KindVol, Label: "内盘"},
	50:  {Key: "volume_ratio", Kind: KindPrice, Label: "量比"},
	51:  {Key: "iep", Kind: KindAuto, Label: "IEP"},
	52:  {Key: "iev", Kind: KindAuto, Label: "IEV"},
	53:  {Key: "time", Kind: KindStr, Label: "时间"},
	54:  {Key: "deal_price", Kind: KindPrice, Label: "成交"},
	55:  {Key: "decision_price", Kind: KindPrice, Label: "决策价"},
	56:  {Key: "ref_price", Kind: KindPrice, Label: "参考价"},
	57:  {Key: "retail", Kind: KindAuto, Label: "散户"},
	58:  {Key: "mid_holder", Kind: KindAuto, Label: "中户"},
	59:  {Key: "big_holder", Kind: KindAuto, Label: "大户"},
	60:  {Key: "main_force", Kind: KindAuto, Label: "主力"},
	61:  {Key: "close", Kind: KindPrice, Label: "收盘"},
	62:  {Key: "open_interest", Kind: KindAuto, Label: "持仓"},
	63:  {Key: "pre_settle_price", Kind: KindPrice, Label: "昨结算"},
	64:  {Key: "pre_close_price", Kind: KindPrice, Label: "昨收价"},
	65:  {Key: "fund_manager", Kind: KindStr, Label: "基金经理"},
	66:  {Key: "invest_type", Kind: KindStr, Label: "投资类型"},
	67:  {Key: "invest_style", Kind: KindStr, Label: "投资风格"},
	84:  {Key: "fund_nav", Kind: KindPrice, Label: "基金净值"},
	91:  {Key: "lot_size", Kind: KindInt, Label: "手数"},
	92:  {Key: "pe", Kind: KindPrice, Label: "市盈率"},
	93:  {Key: "ask5_price", Kind: KindPrice, Label: "卖五价"},
	94:  {Key: "ask4_price", Kind: KindPrice, Label: "卖四价"},
	95:  {Key: "ask3_price", Kind: KindPrice, Label: "卖三价"},
	96:  {Key: "ask2_price", Kind: KindPrice, Label: "卖二价"},
	97:  {Key: "ask1_price", Kind: KindPrice, Label: "卖一价"},
	98:  {Key: "bid1_price", Kind: KindPrice, Label: "买一价"},
	99:  {Key: "bid2_price", Kind: KindPrice, Label: "买二价"},
	100: {Key: "bid3_price", Kind: KindPrice, Label: "买三价"},
	101: {Key: "bid4_price", Kind: KindPrice, Label: "买四价"},
	102: {Key: "bid5_price", Kind: KindPrice, Label: "买五价"},
	138: {Key: "change_pct_alt", Kind: KindPct, Label: "涨跌"},
	139: {Key: "yield_per_10k", Kind: KindAuto, Label: "万份收益"},
	140: {Key: "yield_7d", Kind: KindPct, Label: "七日年化收益"},
	162: {Key: "net_inflow", Kind: KindAuto, Label: "净流入"},
	163: {Key: "net_inflow_3d", Kind: KindAuto, Label: "3日净流入"},
	164: {Key: "net_inflow_10d", Kind: KindAuto, Label: "10日净流入"},
	165: {Key: "position_ratio", Kind: KindPct, Label: "增仓占比"},
	166: {Key: "position_ratio_3d", Kind: KindPct, Label: "3日增仓占比"},
	167: {Key: "position_ratio_10d", Kind: KindPct, Label: "10日增仓占比"},
	168: {Key: "inflow_rank", Kind: KindInt, Label: "排名"},
	169: {Key: "inflow_rank_3d", Kind: KindInt, Label: "3日排名"},
	170: {Key: "inflow_rank_10d", Kind: KindInt, Label: "10日排名"},
	171: {Key: "change_pct_3d", Kind: KindPct, Label: "3日涨幅"},
	172: {Key: "change_pct_10d", Kind: KindPct, Label: "10日涨幅"},
	174: {Key: "total_volume_shares", Kind: KindAuto, Label: "总手"},
	175: {Key: "last_volume_shares", Kind: KindAuto, Label: "现量"},
	176: {Key: "leader_code", Kind: KindCode, Label: "领涨代码"},
	177: {Key: "leader_name", Kind: KindStr, Label: "领涨名称"},
	178: {Key: "leader_last", Kind: KindPrice, Label: "领涨最新"},
	179: {Key: "leader_change_pct", Kind: KindPct, Label: "领涨幅度"},
	180: {Key: "ask5_volume", Kind: KindVol, Label: "卖五量"},
	181: {Key: "ask4_volume", Kind: KindVol, Label: "卖四量"},
	182: {Key: "ask3_volume", Kind: KindVol, Label: "卖三量"},
	183: {Key: "ask2_volume", Kind: KindVol, Label: "卖二量"},
	184: {Key: "ask1_volume", Kind: KindVol, Label: "卖一量"},
	185: {Key: "bid1_volume", Kind: KindVol, Label: "买一量"},
	186: {Key: "bid2_volume", Kind: KindVol, Label: "买二量"},
	187: {Key: "bid3_volume", Kind: KindVol, Label: "买三量"},
	188: {Key: "bid4_volume", Kind: KindVol, Label: "买四量"},
	189: {Key: "bid5_volume", Kind: KindVol, Label: "买五量"},
	206: {Key: "position_diff", Kind: KindAuto, Label: "仓差"},
	208: {Key: "pb", Kind: KindPrice, Label: "市净率"},
	209: {Key: "suspend_flag", Kind: KindFlag, Label: "停牌标志"},
	210: {Key: "trade_status", Kind: KindStr, Label: "停牌状态"},
	211: {Key: "retail_inflow", Kind: KindAuto, Label: "散户流入"},
	212: {Key: "retail_outflow", Kind: KindAuto, Label: "散户流出"},
	213: {Key: "retail_net_inflow", Kind: KindAuto, Label: "散户净流入"},
	214: {Key: "large_inflow", Kind: KindAuto, Label: "大户流入"},
	215: {Key: "large_outflow", Kind: KindAuto, Label: "大户流出"},
	216: {Key: "large_net_inflow", Kind: KindAuto, Label: "大户净流入"},
	217: {Key: "institution_inflow", Kind: KindAuto, Label: "机构流入"},
	218: {Key: "institution_outflow", Kind: KindAuto, Label: "机构流出"},
	219: {Key: "institution_net_inflow", Kind: KindAuto, Label: "机构净流入"},
	220: {Key: "xl_inflow", Kind: KindAuto, Label: "特大单流入"},
	221: {Key: "xl_outflow", Kind: KindAuto, Label: "特大单流出"},
	222: {Key: "xl_net_inflow", Kind: KindAuto, Label: "特大单净流入"},
	223: {Key: "market_no", Kind: KindStr, Label: "市场号"},
	224: {Key: "leader_market_no", Kind: KindStr, Label: "领涨股市场号"},
	225: {Key: "limit_up_price", Kind: KindPrice, Label: "涨停板价"},
	226: {Key: "limit_down_price", Kind: KindPrice, Label: "跌停板价"},
	227: {Key: "dynamic_pe", Kind: KindPrice, Label: "动态市盈率"},
	228: {Key: "bvps", Kind: KindPrice, Label: "每股净资产"},
	229: {Key: "total_shares", Kind: KindShares, Label: "总股本"},
	230: {Key: "market_cap", Kind: KindAmount, Label: "总市值"},
	231: {Key: "float_shares", Kind: KindShares, Label: "流通盘"},
	232: {Key: "float_market_cap", Kind: KindAmount, Label: "流通市值"},
	233: {Key: "eps", Kind: KindPrice, Label: "每股收益"},
	234: {Key: "report_period", Kind: KindStr, Label: "季报年报"},
	240: {Key: "industry_code", Kind: KindCode, Label: "行业代码"},
	241: {Key: "industry_name", Kind: KindStr, Label: "行业名称"},
	242: {Key: "market_status_code", Kind: KindStr, Label: "开市状态"},
	243: {Key: "rise_days", Kind: KindInt, Label: "连涨天数"},
	244: {Key: "fall_days", Kind: KindInt, Label: "连跌天数"},
	256: {Key: "roe", Kind: KindPct, Label: "净资产收益率"},
	257: {Key: "debt_ratio", Kind: KindPct, Label: "资产负债率"},
	258: {Key: "net_profit_yoy", Kind: KindPct, Label: "净利润同比增长率"},
	259: {Key: "revenue_yoy", Kind: KindPct, Label: "营业收入同比增长率"},
	260: {Key: "op_profit_yoy", Kind: KindPct, Label: "营业利润同比增长率"},
	261: {Key: "total_asset_yoy", Kind: KindPct, Label: "总资产同比增长率"},
	265: {Key: "fast_change_pct", Kind: KindPct, Label: "快速涨跌幅"},
	277: {Key: "lot", Kind: KindInt, Label: "每手"},
	288: {Key: "adj_last", Kind: KindPrice, Label: "后复权最新价"},
	292: {Key: "activity", Kind: KindAuto, Label: "活跃度"},
	293: {Key: "speculation", Kind: KindPrice, Label: "投机度"},
	294: {Key: "change_pct_20d", Kind: KindPct, Label: "20日涨跌幅"},
	295: {Key: "change_pct_60d", Kind: KindPct, Label: "60日涨跌幅"},
	296: {Key: "change_pct_120d", Kind: KindPct, Label: "120日涨跌幅"},
	297: {Key: "change_pct_250d", Kind: KindPct, Label: "250日涨跌幅"},
	298: {Key: "change_pct_5d", Kind: KindPct, Label: "5日涨跌幅"},
	299: {Key: "change_pct_ytd", Kind: KindPct, Label: "今年涨幅"},
	306: {Key: "pe_ttm", Kind: KindPrice, Label: "市盈率TTM"},
	309: {Key: "trend_binary", Kind: KindBinary, Label: "走势"},
	331: {Key: "main_net_inflow", Kind: KindAuto, Label: "主力净买入"},
	333: {Key: "margin_ratio", Kind: KindPct, Label: "融资抵押率"},
	335: {Key: "dividend_yield", Kind: KindPct, Label: "股息率"},
	341: {Key: "change_pct_month", Kind: KindPct, Label: "本月涨幅"},
	342: {Key: "change_pct_week", Kind: KindPct, Label: "本周涨幅"},
	343: {Key: "premarket_flag", Kind: KindFlag, Label: "盘前盘后标识"},
	344: {Key: "premarket_change", Kind: KindPrice, Label: "盘前盘后涨跌"},
	345: {Key: "premarket_last", Kind: KindPrice, Label: "盘前盘后最新价"},
	346: {Key: "premarket_change_pct", Kind: KindPct, Label: "盘前/盘后涨跌幅"},
	347: {Key: "discount_rate", Kind: KindPct, Label: "折价率"},
	348: {Key: "change_pct_6m", Kind: KindPct, Label: "近6个月涨幅"},
	349: {Key: "change_pct_1y", Kind: KindPct, Label: "近1年涨幅"},
	356: {Key: "lot_amount", Kind: KindAmount, Label: "一手金额"},
	357: {Key: "spread_cost", Kind: KindAmount, Label: "买卖成本差价"},
	359: {Key: "lot_shares", Kind: KindInt, Label: "一手数量"},
	362: {Key: "limit_up_pct", Kind: KindPct, Label: "涨停限制幅度"},
	363: {Key: "limit_down_pct", Kind: KindPct, Label: "跌停限制幅度"},
	364: {Key: "etf_class1", Kind: KindStr, Label: "ETF分类1"},
	365: {Key: "etf_class2", Kind: KindStr, Label: "ETF分类2"},
	372: {Key: "net_inflow_5d", Kind: KindAuto, Label: "5日净流入"},
	380: {Key: "prev_day_change_pct", Kind: KindPct, Label: "昨日涨幅"},
	381: {Key: "change_pct_3y", Kind: KindPct, Label: "近3年涨幅"},
	382: {Key: "change_pct_5y", Kind: KindPct, Label: "近5年涨幅"},
	387: {Key: "track_index_code", Kind: KindCode, Label: "跟踪指数代码"},
	388: {Key: "track_index_name", Kind: KindStr, Label: "跟踪指数名称"},
	390: {Key: "limit_up_count", Kind: KindInt, Label: "涨停家数"},
	391: {Key: "limit_down_count", Kind: KindInt, Label: "跌停家数"},
	401: {Key: "remaining_scale", Kind: KindAuto, Label: "剩余规模"},
	404: {Key: "name_pinyin_initial", Kind: KindStr, Label: "证券名称首字母"},
	408: {Key: "dividend", Kind: KindPrice, Label: "股息"},
	418: {Key: "name_full", Kind: KindStr, Label: "扩位简称"},
	419: {Key: "underlying_market_no", Kind: KindStr, Label: "正股市场号"},
	420: {Key: "is_unprofitable", Kind: KindFlag, Label: "是否尚未盈利"},
	421: {Key: "has_voting_diff", Kind: KindFlag, Label: "是否存在投票权差异"},
	424: {Key: "speed", Kind: KindPct, Label: "涨速"},
	425: {Key: "net_profit", Kind: KindAmount, Label: "净利润"},
	427: {Key: "mid_price", Kind: KindPrice, Label: "中间价"},
	428: {Key: "auction_amount", Kind: KindAmount, Label: "集合竞价金额"},
	429: {Key: "auction_change_pct", Kind: KindPct, Label: "竞价涨幅"},
	430: {Key: "auction_volume", Kind: KindVol, Label: "竞价量"},
	452: {Key: "latest_price", Kind: KindPrice, Label: "行情最新价"},
	453: {Key: "dividend_yield_ttm", Kind: KindPrice, Label: "股息率TTM"},
	454: {Key: "change_pct_1m", Kind: KindPct, Label: "近1月涨幅"},
	455: {Key: "change_pct_3m", Kind: KindPct, Label: "近3月涨幅"},
	456: {Key: "main_inflow_minute_binary", Kind: KindBinary, Label: "主力净流入分时"},
}

// keyToCode 是字段 key 到字段码的反查表，供按名称追加字段使用。
var keyToCode = func() map[string]int {
	m := make(map[string]int, len(fieldSpecs))
	for code, spec := range fieldSpecs {
		m[spec.Key] = code
	}
	return m
}()

// DefaultFields 是默认的 45 个字段码，与上游脚本的 DEFAULT_FIELDS 一致。
var DefaultFields = []int{
	1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 32, 48, 49, 50, 92, 162, 208,
	209, 210, 225, 226, 227, 228, 229, 230, 231, 232, 233, 256, 294, 295, 296, 297, 298,
	306, 335, 342, 408, 453,
}

// fieldKey 返回字段码对应的输出 key；未命名的码用 f<code>。
func fieldKey(code int) string {
	if spec, ok := fieldSpecs[code]; ok {
		return spec.Key
	}
	return "f" + strconv.Itoa(code)
}

// fieldKind 返回字段码的转换方式；未命名的码按 auto 处理。
func fieldKind(code int) FieldKind {
	if spec, ok := fieldSpecs[code]; ok {
		return spec.Kind
	}
	return KindAuto
}
