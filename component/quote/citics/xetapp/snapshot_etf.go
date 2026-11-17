package xetapp

import (
	"context"
	"strings"
)

// DefaultETFBoard 是全量沪深 ETF 的板块中文名。
//
// 实测「沪市ETF基金」「深市ETF基金」都返回 -206021（查无相应的排名信息），
// 只有本板块可用；其他板块名（如「科创板」）能返回数据，但内容是股票而非 ETF。
const DefaultETFBoard = "沪深ETF基金"

// ETFSnapshotRequest 是 ETF 行情快照的请求参数。
type ETFSnapshotRequest struct {
	// Board 板块中文名，为空取 DefaultETFBoard（沪深ETF基金）。
	Board string
	// Fields 完全自定义的字段码；为空使用 DefaultFields（45 个）。
	// 传入后输出只包含这些字段。
	Fields []int
	// ExtraFields 在 Fields（或默认集）基础上追加的字段码，重复的自动忽略。
	// ETF 专属指标用它追加，如 84（基金净值）、326（基金规模）、347（折价率）。
	ExtraFields []int
	// MaxCount 返回行数上限，0 取 DefaultMaxCount。
	// 该值必须显式传：站点在不传时只返回 1 行。
	MaxCount int
	// StartPos 分页起点，从 1 开始，是**成分序号**而非行号；0 表示不传。
	StartPos int
}

// GetQuoteSnapshotCNAETF 拉取沪深两市全量 ETF 行情快照。
//
// 动作号 20192，按板块中文名取成分，一次请求返回全量（实测 1727 只），无需分页。
// 与 A 股接口的差别：响应里没有「池自身」那一行，因此不会剔除首条；`STOCKCODE` /
// `STOCKNAME` 也不返回。名称仍带 `1.` 这类序号前缀，解析时已剥离。
//
// 响应结构异常、ERRORNO 为负值、无 GRID0 数据时返回 *APIError，
// 网络错误与响应截断会自动重试。
func (c *Client) GetQuoteSnapshotCNAETF(ctx context.Context, req ETFSnapshotRequest) (*SnapshotResult, error) {
	board := strings.TrimSpace(req.Board)
	if board == "" {
		board = DefaultETFBoard
	}
	return c.fetchBoardSnapshot(ctx, boardFetch{
		Action:   ActionQuoteSnapshotETF,
		Board:    board,
		Fields:   req.Fields,
		Extra:    req.ExtraFields,
		MaxCount: req.MaxCount,
		StartPos: req.StartPos,
	})
}
