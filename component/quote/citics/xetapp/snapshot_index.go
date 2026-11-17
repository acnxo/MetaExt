package xetapp

import (
	"context"
	"fmt"
)

// IndexBoard 是「全量指数」合并过程中的一个板块池。
type IndexBoard struct {
	// Name 池名称，用于统计与错误定位，如「中证指数」。
	Name string
	// Action 站点动作号：20191 按市场号、20192 按板块中文名、20198 大盘指数表。
	Action string
	// Board 请求的 StockCode：20191 传市场号（如 1100），20192 传板块中文名，
	// 20198 不使用该参数。
	Board string
}

// IndexBoards 是默认合并的 4 个板块池，按顺序请求、按代码去重。
//
// 站点没有「全部指数」的单一入口：中证指数 832 只、国证指数 20 只、股转指数 10 只、
// 大盘指数 42 只。四个池有重叠（实测 904 行去重后 879 只），先出现的池优先。
var IndexBoards = []IndexBoard{
	{Name: "中证指数", Action: ActionQuoteSnapshotByMarket, Board: "1100"},
	{Name: "国证指数", Action: ActionQuoteSnapshotETF, Board: "国证指数"},
	{Name: "股转指数", Action: ActionQuoteSnapshotETF, Board: "股转指数"},
	{Name: "大盘指数", Action: ActionQuoteSnapshotMarketIndex},
}

// IndexSnapshotRequest 是全量指数快照的请求参数。
type IndexSnapshotRequest struct {
	// Boards 自定义的板块池；为空使用 IndexBoards（4 个池合并成沪深京全量指数）。
	Boards []IndexBoard
	// Fields 完全自定义的字段码；为空使用 DefaultFields（45 个）。
	Fields []int
	// ExtraFields 在 Fields（或默认集）基础上追加的字段码，重复的自动忽略。
	ExtraFields []int
	// MaxCount 每个池的返回行数上限，0 取 DefaultMaxCount。
	MaxCount int
}

// IndexPool 是某个板块池在一次合并中的统计。
type IndexPool struct {
	// Name 池名称。
	Name string
	// Action 站点动作号。
	Action string
	// Board 请求的 StockCode；大盘指数池为空。
	Board string
	// MaxCount 该池的成分总数（响应 MAXCOUNT）。
	MaxCount int
	// Fetched 该池返回的行数（去重前）。
	Fetched int
	// Added 去重后新增的行数。
	Added int
	// ErrorNo 响应 ERRORNO 原值；正值是正常的，等于返回行数 + 2。
	ErrorNo int
}

// IndexSnapshotResult 是合并多个板块池后的指数快照。
type IndexSnapshotResult struct {
	// Rows 去重后的指数行情，顺序为各池首次出现的顺序。
	Rows []Row
	// Pools 各池的请求与统计，顺序与请求顺序一致。
	Pools []IndexPool
	// NeedCheck 服务端实际生效的字段码，取首个池的回显（四个池传同一组码，结果一致）。
	NeedCheck []int
	// HQDate 行情日期 yyyyMMdd。
	HQDate string
	// ServerDateTime 服务器时间。
	ServerDateTime string
}

// GetQuoteSnapshotCNAIndex 拉取沪深京全量指数行情快照。
//
// 站点没有「全部指数」的单一入口，因此按 IndexBoards 串行请求 4 个池再合并：
// 中证指数（20191 + 市场号 1100）、国证指数与股转指数（20192 + 板块中文名）、
// 大盘指数（20198）。去重键是证券代码，先出现的池优先。
//
// 四个请求共用一个字段集，传了 NeedCheck 后各池表头一致，因此可共用解析逻辑
// （不传时各池的默认列数不同：20191 为 9 列，20192 / 20198 为 20 列）。
//
// 任一池失败即返回 *APIError（错误信息带池名），不会返回缺数据的结果。
func (c *Client) GetQuoteSnapshotCNAIndex(ctx context.Context, req IndexSnapshotRequest) (*IndexSnapshotResult, error) {
	boards := req.Boards
	if len(boards) == 0 {
		boards = IndexBoards
	}
	result := &IndexSnapshotResult{
		Rows:  make([]Row, 0, len(boards)*256),
		Pools: make([]IndexPool, 0, len(boards)),
	}
	seen := make(map[string]struct{}, len(boards)*256)
	for _, board := range boards {
		fetched, err := c.fetchBoardSnapshot(ctx, boardFetch{
			Action:   board.Action,
			Board:    board.Board,
			Fields:   req.Fields,
			Extra:    req.ExtraFields,
			MaxCount: req.MaxCount,
		})
		if err != nil {
			return nil, fmt.Errorf("指数池 %s（action=%s）：%w", board.Name, board.Action, err)
		}
		pool := IndexPool{
			Name:     board.Name,
			Action:   board.Action,
			Board:    board.Board,
			MaxCount: fetched.MaxCount,
			Fetched:  len(fetched.Rows),
			ErrorNo:  fetched.ErrorNo,
		}
		if len(result.NeedCheck) == 0 {
			result.NeedCheck = fetched.NeedCheck
			result.HQDate = fetched.HQDate
			result.ServerDateTime = fetched.ServerDateTime
		}
		for _, row := range fetched.Rows {
			if row.Code == "" {
				continue
			}
			if _, ok := seen[row.Code]; ok {
				continue
			}
			seen[row.Code] = struct{}{}
			pool.Added++
			result.Rows = append(result.Rows, row)
		}
		result.Pools = append(result.Pools, pool)
	}
	return result, nil
}
