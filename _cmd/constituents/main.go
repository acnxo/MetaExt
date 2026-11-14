// Command constituents 是百度股市通成分股接口（sapi/v1/constituents）的命令行工具。
//
// 用于按指数或概念板块拉取成分股与行情快照，默认按总市值降序、分页翻完为止。
// 该接口无签名，靠 Referer、浏览器 UA 与访问频率做风控，被拒时工具会提示降频，
// 而不是把风控响应当成空数据。
//
// 用法示例：
//
//	go run ./_cmd/constituents -code 000300 -limit 20
//	go run ./_cmd/constituents -financeType block -code GN2309 -sortKey amount -market ab
//	go run ./_cmd/constituents -code 000300 -json
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/acnxo/MetaExt/component/quote/baidu/finscope"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "错误："+err.Error())
		if finscope.IsRiskControl(err) {
			fmt.Fprintln(os.Stderr, "提示：这是站点风控，不是参数问题。请降低请求频率、复用连接或更换出口网络。")
		}
		os.Exit(1)
	}
}

func run() error {
	var (
		financeType = flag.String("financeType", finscope.FinanceTypeIndex, "成分类型：index 指数、block 概念/行业板块")
		market      = flag.String("market", finscope.MarketA, "市场编码：ab A 股、hk 港股、us 美股")
		code        = flag.String("code", "000300", "来源标识码：指数 6 位代码，板块 GNxxxx")
		sortKey     = flag.String("sortKey", finscope.SortByMarketValue, "排序字段：marketValue 总市值、amount 成交额")
		sortType    = flag.String("sortType", finscope.SortDesc, "排序方向：desc 降序、asc 升序")
		pageSize    = flag.Int("rn", 100, "单页条数")
		limit       = flag.Int("limit", 0, "最多输出条数，0 表示全部")
		referer     = flag.String("referer", "", "自定义 Referer，默认使用站点首页；个股页地址更宽松")
		baseURL     = flag.String("baseURL", "", "自定义接口站点，默认 https://finance.pae.baidu.com，可用于代理调试")
		asJSON      = flag.Bool("json", false, "以 JSON 输出完整字段，而不是表格")
		timeout     = flag.Duration("timeout", 60*time.Second, "整体超时时间")
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	client := finscope.New(finscope.Config{BaseURL: *baseURL, Referer: *referer})
	req := finscope.GetConstituentsRequest{
		FinanceType: *financeType,
		Market:      *market,
		Code:        *code,
		SortKey:     *sortKey,
		SortType:    *sortType,
		Limit:       *pageSize,
	}
	result, err := client.GetAllConstituents(ctx, req)
	if err != nil {
		// 超时或被中断时，已经取到的条目仍然打印，便于确认进度。
		if len(result.Items) == 0 {
			return err
		}
		fmt.Fprintf(os.Stderr, "警告：翻页中断，仅输出已取到的 %d 条：%v\n", len(result.Items), err)
	}
	items := result.Items
	if *limit > 0 && len(items) > *limit {
		items = items[:*limit]
	}
	if len(items) == 0 {
		return errors.New("没有取到成分股：请确认 code 与 financeType 的组合，或该来源已无成分股")
	}
	if *asJSON {
		return printJSON(items)
	}
	printTable(items, result.Pages)
	return nil
}

// printJSON 输出完整条目，包含 rawData 与板块信息。
func printJSON(items []finscope.Constituent) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化结果失败：%w", err)
	}
	fmt.Println(string(data))
	return nil
}

// printTable 按列输出「代码 / 简称 / 交易所 / 最新价 / 涨跌幅 / 成交额」。
//
// 表格取原始数值渲染，避免直接拼接带单位的中文字符串导致列错位。
func printTable(items []finscope.Constituent, pages int) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "代码\t简称\t交易所\t最新价\t涨跌幅\t成交额(万)")
	for _, item := range items {
		fmt.Fprintf(w, "%s\t%s\t%s\t%.2f\t%.2f%%\t%.0f\n",
			item.Code, item.Name, item.Exchange,
			item.RawData.LastPrice, item.RawData.ChangeRatio, item.RawData.Amount/1e4)
	}
	_ = w.Flush()
	fmt.Printf("\n共 %d 条，%d 页\n", len(items), pages)
}
