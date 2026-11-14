// Command snapshot 是行情快照（action=20199）的命令行工具。
//
// 一次请求拿到整个池的成分快照，默认输出代码、名称、最新价、涨跌幅、成交额与总市值。
//
// 用法示例：
//
//	go run ./_cmd/snapshot -limit 20                    # 全市场前 20 只
//	go run ./_cmd/snapshot -pool hs300 -limit 10        # 沪深300 前 10 只
//	go run ./_cmd/snapshot -pool 993665 -fields 1,2,4,92 -limit 5
//	go run ./_cmd/snapshot -pool all -extra 341,372 -json > snapshot.json
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/acnxo/MetaExt/component/quote/hqgw/snapshot"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "错误："+err.Error())
		if snapshot.IsBusinessError(err) {
			fmt.Fprintln(os.Stderr, "提示：这是服务端的业务错误码（ERRORNO 为负），检查池代码或改用支持的标的。")
		}
		os.Exit(1)
	}
}

func run() error {
	var (
		pool    = flag.String("pool", "all", "池别名（all/sh_sz/hs300/zz500/zz2000/star50/gem/army）或内部代码")
		fields  = flag.String("fields", "", "自定义字段码，逗号分隔，如 1,2,4,92；为空用默认 45 字段")
		extra   = flag.String("extra", "", "在默认字段上追加的字段码，逗号分隔，如 341,372")
		limit   = flag.Int("limit", 20, "最多输出条数，0 表示全部")
		maxRows = flag.Int("maxCount", snapshot.DefaultMaxCount, "请求的返回行数上限")
		asJSON  = flag.Bool("json", false, "以 JSON 输出完整字段")
		timeout = flag.Duration("timeout", 120*time.Second, "整体超时时间")
	)
	flag.Parse()

	fieldCodes, err := parseCodes(*fields)
	if err != nil {
		return err
	}
	extraCodes, err := parseCodes(*extra)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	client := snapshot.New(snapshot.Config{})
	started := time.Now()
	result, err := client.GetQuoteSnapshotCNAStock(ctx, snapshot.SnapshotRequest{
		Pool:        *pool,
		Fields:      fieldCodes,
		ExtraFields: extraCodes,
		MaxCount:    *maxRows,
	})
	if err != nil {
		return err
	}
	rows := result.Rows
	if *limit > 0 && len(rows) > *limit {
		rows = rows[:*limit]
	}
	if *asJSON {
		return printJSON(result, rows)
	}
	printTable(result, rows, time.Since(started))
	return nil
}

// parseCodes 解析逗号分隔的字段码。
func parseCodes(text string) ([]int, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	parts := strings.Split(text, ",")
	codes := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		code, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("字段码 %q 不是整数", part)
		}
		if code < 1 || code > 462 {
			return nil, fmt.Errorf("字段码 %d 超出范围 1–462", code)
		}
		codes = append(codes, code)
	}
	return codes, nil
}

// printJSON 输出元信息与结构化行。
func printJSON(result *snapshot.SnapshotResult, rows []snapshot.Row) error {
	payload := struct {
		Pool      string         `json:"pool"`
		PoolName  string         `json:"pool_name"`
		Total     int            `json:"total"`
		Returned  int            `json:"returned"`
		HQDate    string         `json:"hq_date"`
		ServerAt  string         `json:"server_datetime"`
		NeedCheck []int          `json:"need_check"`
		Rows      []snapshot.Row `json:"rows"`
	}{
		Pool:      result.PoolCode,
		PoolName:  result.PoolName,
		Total:     result.MaxCount,
		Returned:  len(result.Rows),
		HQDate:    result.HQDate,
		ServerAt:  result.ServerDateTime,
		NeedCheck: result.NeedCheck,
		Rows:      rows,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化结果失败：%w", err)
	}
	fmt.Println(string(data))
	return nil
}

// printTable 输出常用列；字段缺失时显示为 0。
func printTable(result *snapshot.SnapshotResult, rows []snapshot.Row, elapsed time.Duration) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "代码\t名称\t最新\t涨跌幅\t成交额(亿)\t总市值(亿)\t市盈率\t市净率")
	for _, row := range rows {
		fmt.Fprintf(w, "%s\t%s\t%.2f\t%+.2f%%\t%.2f\t%.2f\t%.2f\t%.2f\n",
			row.Code, row.Name, row.Last, row.ChangeRatio,
			row.Amount/1e8, row.MarketCap/1e8, row.PE, row.PB)
	}
	_ = w.Flush()
	fmt.Printf("\n池 %s（%s）共 %d 只，本次输出 %d 只，耗时 %s\n",
		result.PoolName, result.PoolCode, result.MaxCount, len(rows), elapsed.Round(time.Millisecond))
	if result.HQDate != "" {
		fmt.Printf("行情日期 %s\n", result.HQDate)
	}
}
