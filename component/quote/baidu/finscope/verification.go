package finscope

// VerificationStatus 表示接口在本仓库的验证状态。
type VerificationStatus string

const (
	// StatusVerified 已用真实响应验证过字段与解析，可以直接使用。
	StatusVerified VerificationStatus = "verified"
	// StatusBlocked 请求参数已按文档构造并核对无误，但站点风控（Result.code=403）挡住了
	// 真实响应，字段结构未取得样本，解析逻辑未经真机验证。
	//
	// 这类接口的 Go 方法仍然可用：风控命中会返回 *APIError 并由 IsRiskControl 识别，
	// 不会当成空数据。等出口不被风控时，用 `FINSCOPE_LIVE=1 go test -run TestLive` 复测，
	// 通过后把状态改成 StatusVerified 并补齐结构化解析。
	StatusBlocked VerificationStatus = "blocked"
)

// Endpoint 描述一个接口的验证状态，用于在代码里留下可查询的进度清单。
type Endpoint struct {
	// Method 导出的客户端方法名。
	Method string
	// Path 接口路径。
	Path string
	// Group getquotation 系列的分组名；独立路径的接口留空。
	Group string
	// Status 验证状态。
	Status VerificationStatus
	// Note 备注：已验证的写数据来源，被阻塞的写复测条件。
	Note string
}

// Endpoints 返回各接口的验证状态清单。
//
// 截止 2026-11-15，只有 constituents 与 sug 两个接口能用真实响应验证；
// getquotation 的四个分组与 selfselect 的期货五日在本机一律返回
// `{"code":403,"isCaptchaEnabled":true,"msg":"hit risk"}`，换 UA、Referer 与出口均无效。
func Endpoints() []Endpoint {
	return []Endpoint{
		{
			Method: "GetConstituents",
			Path:   DefaultConstituentsPath,
			Status: StatusVerified,
			Note:   "真实响应验证：355 家北交所企业、北证50 成分股、分页一致性均已核对",
		},
		{
			Method: "Suggest",
			Path:   DefaultSuggestPath,
			Status: StatusVerified,
			Note:   "真实响应验证：按名称查到北证50、专精特新（含指数与板块）及其实时点位",
		},
		{
			Method: "GetQuoteKLine",
			Path:   DefaultQuotePath,
			Group:  GroupKLine,
			Status: StatusBlocked,
			Note:   "站点风控 403；解析逻辑用文档实测样本的单测覆盖，待真机复测",
		},
		{
			Method: "GetQuoteFiveDay",
			Path:   DefaultQuotePath,
			Group:  GroupFiveDay,
			Status: StatusBlocked,
			Note:   "间歇性风控：曾取得 200 响应，实测 keys 为 10 列（比文档多 avgPrice/totalVolume/totalAmount），但当时 marketData 为空；有数据时需复测解析",
		},
		{
			Method: "GetQuoteForexMinute",
			Path:   DefaultQuotePath,
			Group:  GroupForexMinute,
			Status: StatusBlocked,
			Note:   "站点风控 403；返回 RawQuoteResult，待取得真实 Result 结构",
		},
		{
			Method: "GetQuoteFuturesMinute",
			Path:   DefaultQuotePath,
			Group:  GroupFuturesMinute,
			Status: StatusBlocked,
			Note:   "站点风控 403；待确认 keys 是否含持仓量、结算价等期货专有列",
		},
		{
			Method: "GetQuoteFuturesFiveDay",
			Path:   DefaultFuturesFiveDayPath,
			Group:  GroupFuturesFiveDay,
			Status: StatusBlocked,
			Note:   "站点风控 403；待确认是否与股票五日同构、是否含夜盘时段",
		},
	}
}

// BlockedEndpoints 返回所有被站点风控挡住、尚待真机复测的接口。
func BlockedEndpoints() []Endpoint {
	blocked := make([]Endpoint, 0, len(Endpoints()))
	for _, endpoint := range Endpoints() {
		if endpoint.Status == StatusBlocked {
			blocked = append(blocked, endpoint)
		}
	}
	return blocked
}
