// Package xetapp 封装上游行情网关。
//
// 当前实现三个行情快照接口，字段码同一套、解析逻辑共用：
//
//   - GetQuoteSnapshotCNAStock：A 股全市场（`action=20199` + 板块内部码 `994650`），
//     一次请求拿到五千余只证券；
//   - GetQuoteSnapshotCNAETF：沪深全量 ETF（`action=20192` + 板块中文名 `沪深ETF基金`），
//     一次请求拿到一千七百余只；
//   - GetQuoteSnapshotCNAIndex：沪深京全量指数（`action=20191` / `20192` / `20198`
//     四个池合并去重），站点没有「全部指数」单一入口，由本包串起 4 个请求。
//
// 三个接口都无需分页、无需登录态、无签名。
//
// 该网关的响应不是 JSON 数组，而是「表头 + 竖线分隔记录」的字符串表：
//
//	GRID0[0] = "名称|最新|幅度|...|代码|"            ← 表头
//	GRID0[1] = "沪深京全A|1488.03|-2.08%|...|994650|"  ← 池自身，需剔除
//	GRID0[2] = "1.幸福蓝海|17.99|20.01%|...|300528|"   ← 成分，名称带序号前缀
//
// 使用前需要知道四件事：
//
//   - **列顺序以响应的 `NEEDCHECK` 为准**，不按传参顺序；顶层 `*INDEX` 是默认 20 列的
//     索引，传自定义 `NeedCheck` 后不会更新，因此不能用它定位列；
//   - `MaxCount` 必须显式传，不传只返回池自身与第一条成分；
//   - `ERRORNO` 正值是正常的（等于返回行数 + 2），**只有负值才是错误**，且 HTTP 状态恒为 200；
//   - 全市场请求偶发「HTTP 200 但响应体被截断」，本包内置退避重试。
//
// 约定：
//
//   - 行情类字段在原始响应里带单位（`亿`/`万`/`%`），本包统一还原成数值：
//     百分数去掉 `%`，`万`/`亿`/`万亿` 换算成元，「--」转成零值或不在结果中出现；
//   - 需要额外指标时用 ExtraFields 追加字段码，不必改代码。
package xetapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// 协议默认值。
const (
	// DefaultGatewayURL 网关地址，POST 表单即可。
	DefaultGatewayURL = "https://xetapp.citics.com/reqxml"
	// DefaultUserAgent 默认 UA；站点不校验，带浏览器 UA 更稳妥。
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/120.0 Safari/537.36"
	// ActionQuoteSnapshot 按池内部码取成分行情的动作号（A 股全市场）。
	ActionQuoteSnapshot = "20199"
	// ActionQuoteSnapshotETF 按板块中文名取成分行情的动作号（ETF 与部分指数池）。
	ActionQuoteSnapshotETF = "20192"
	// ActionQuoteSnapshotByMarket 按市场号取成分行情的动作号（中证指数池）。
	ActionQuoteSnapshotByMarket = "20191"
	// ActionQuoteSnapshotMarketIndex 大盘指数表的动作号，不需要 StockCode。
	ActionQuoteSnapshotMarketIndex = "20198"
	// DefaultPoolCode 沪深京全A 的板块内部代码（全市场，含北交所与 ST）。
	DefaultPoolCode = "994650"
	// DefaultMaxCount 返回行数上限；实测 5579~5580 只，留足余量。
	DefaultMaxCount = 60000
	// DefaultTimeout 单次请求超时；全市场响应约 1.4MB，需要留足时间。
	DefaultTimeout = 60 * time.Second
	// DefaultRetries 请求与解析失败的重试次数。
	DefaultRetries = 4

	// retryBackoff 退避基数，第 n 次重试等待 n 倍。
	retryBackoff = 1500 * time.Millisecond
	// maxBodyBytes 限制响应体大小。
	maxBodyBytes = 64 << 20
	// errorBodyLimit 错误信息中保留的响应体长度。
	errorBodyLimit = 300
)

// Pools 是池别名到板块内部代码的映射，与上游脚本的 POOLS 一致。
//
// 别名之外也可以直接传内部代码（如 `994650`、`000300`、`993665`）。
var Pools = map[string]string{
	"all":       DefaultPoolCode, // 沪深京全A（全市场）
	"sh_sz":     "994649",        // 沪深A股（不含北交所）
	"all_share": "1B0985",        // 中证全指（指数样本）
	"hs300":     "000300",        // 沪深300
	"zz500":     "1B0905",        // 中证500
	"zz2000":    "932000",        // 中证2000
	"star50":    "1B0688",        // 科创板 50
	"gem":       "399006",        // 创业板指
	"army":      "993665",        // 军工（行业板块）
}

// Config 是客户端配置；零值代表使用协议默认值。
type Config struct {
	// GatewayURL 网关地址，默认 DefaultGatewayURL。
	GatewayURL string
	// UserAgent 请求 UA，默认 DefaultUserAgent。
	UserAgent string
	// Timeout 单次请求超时，默认 DefaultTimeout。
	Timeout time.Duration
	// Retries 重试次数，默认 DefaultRetries；小于 1 时按 1 处理。
	Retries int
	// HTTPClient 可注入自定义客户端，主要用于测试与代理场景。
	HTTPClient *http.Client
	// Sleep 可注入等待函数，测试时用来跳过退避等待。
	Sleep func(time.Duration)
}

func (c Config) withDefaults() Config {
	if strings.TrimSpace(c.GatewayURL) == "" {
		c.GatewayURL = DefaultGatewayURL
	}
	if strings.TrimSpace(c.UserAgent) == "" {
		c.UserAgent = DefaultUserAgent
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	if c.Retries <= 0 {
		c.Retries = DefaultRetries
	}
	if c.Sleep == nil {
		c.Sleep = time.Sleep
	}
	return c
}

// Client 是 行情网关客户端。
//
// Client 并发安全：创建后仅读取配置，内部不保存可变状态。
type Client struct {
	cfg  Config
	http *http.Client
}

// New 创建客户端，零值配置按协议默认值补齐。
func New(cfg Config) *Client {
	cfg = cfg.withDefaults()
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &Client{cfg: cfg, http: client}
}

// Config 返回生效配置（已补齐默认值）。
func (c *Client) Config() Config { return c.cfg }

// APIError 表示接口没有返回可用数据。
//
// ErrorNo 为响应里的 ERRORNO 负值；请求与解析失败时为 0。
type APIError struct {
	// ErrorNo 业务错误码，只有负值才是错误。
	ErrorNo int
	// Message 服务端的中文提示或本地错误说明。
	Message string
	// Attempts 实际发起请求的次数。
	Attempts int
	// Err 被包装的底层错误，可能为 nil。
	Err error
}

func (e *APIError) Error() string {
	switch {
	case e.ErrorNo != 0:
		return fmt.Sprintf("行情网关调用失败：ERRORNO=%d %s", e.ErrorNo, e.Message)
	case e.Attempts > 0:
		return fmt.Sprintf("行情网关请求失败（%d 次尝试）：%s", e.Attempts, e.Message)
	default:
		return "行情网关请求失败：" + e.Message
	}
}

// Unwrap 返回被包装的底层错误。
func (e *APIError) Unwrap() error { return e.Err }

// postJSON 以表单提交并对失败做退避重试。
//
// 站点会偶发返回 HTTP 200 但响应体被截断，表现为 JSON 解析失败；这类错误会重试，
// 业务错误（ERRORNO 为负）不重试，交给调用方判断。
func (c *Client) postJSON(ctx context.Context, form url.Values, out any) error {
	body := form.Encode()
	var lastErr error
	for attempt := 1; attempt <= c.cfg.Retries; attempt++ {
		if err := ctx.Err(); err != nil {
			return &APIError{Message: err.Error(), Attempts: attempt - 1, Err: err}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.GatewayURL, strings.NewReader(body))
		if err != nil {
			return &APIError{Message: "构造请求失败: " + err.Error(), Attempts: attempt, Err: err}
		}
		req.Header.Set("content-type", "application/x-www-form-urlencoded")
		req.Header.Set("user-agent", c.cfg.UserAgent)

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
		} else {
			payload, readErr := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
			_ = resp.Body.Close()
			if readErr != nil {
				lastErr = readErr
			} else if resp.StatusCode != http.StatusOK {
				lastErr = fmt.Errorf("HTTP %d %s", resp.StatusCode, truncate(string(payload), errorBodyLimit))
			} else if err := json.Unmarshal(payload, out); err != nil {
				// 响应被截断时 JSON 不完整，重试通常第二次即成功。
				lastErr = fmt.Errorf("响应解析失败（可能被截断）: %w", err)
			} else {
				return nil
			}
		}
		if attempt < c.cfg.Retries {
			c.cfg.Sleep(time.Duration(attempt) * retryBackoff)
		}
	}
	return &APIError{
		Message:  lastErr.Error(),
		Attempts: c.cfg.Retries,
		Err:      lastErr,
	}
}

// IsBusinessError 判断错误是否为服务端返回负值 ERRORNO。
func IsBusinessError(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.ErrorNo < 0
}

// truncate 截断过长的文本，避免错误信息携带整个响应体。
func truncate(s string, limit int) string {
	s = strings.TrimSpace(s)
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}

// trimTrailingEmpty 去掉切分后末尾的空元素（记录与表头都以竖线结尾）。
func trimTrailingEmpty(cells []string) []string {
	if n := len(cells); n > 0 && cells[n-1] == "" {
		return cells[:n-1]
	}
	return cells
}
