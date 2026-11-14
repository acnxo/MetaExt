// Command quote 是百度股市通行情接口（vapi/v1/getquotation，group=quotation_kline_ab）
// 的命令行工具，用于获取证券或指数的 K 线。
//
// 该接口风控严格：本机对指数、个股、各种 group 的请求都会返回
// {"code":403,"isCaptchaEnabled":true,"msg":"hit risk"}，换 UA、Referer 与出口均无效。
// 工具会把这种情况明确报告为风控错误，而不是打印空结果。
//
// 行情接口被风控时，可改用搜索接口（/vapi/v1/sug）按名称定位代码并读取其行情快照：
//
//	go run ./_cmd/quote -wd 北证50 -type index
//
// 用法示例：
//
//	go run ./_cmd/quote -code 899050 -ktype day -count 10
//	go run ./_cmd/quote -code 920045 -ktype min1 -count 241
//	go run ./_cmd/quote -code 899050 -ktype day -json
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
			fmt.Fprintln(os.Stderr, "提示：站点风控命中，不是参数问题。需要降低频率、复用连接或更换出口网络。")
		}
		os.Exit(1)
	}
}

func run() error {
	var (
		code     = flag.String("code", "", "证券或指数代码，如 899050（北证50）、920045")
		wd       = flag.String("wd", "", "按名称搜索并输出行情快照，如 北证50；与 -code 二选一")
		itemType = flag.String("type", "", "搜索时按类型过滤：index / stock / etf / block / fund")
		market   = flag.String("marketType", finscope.MarketA, "市场：ab A 股、hk 港股、us 美股")
		ktype    = flag.String("ktype", finscope.KTypeDay, "周期：min1 分钟、min5 五分钟、day 日线")
		count    = flag.Int("count", 20, "返回条数")
		name     = flag.String("name", "", "证券简称，可空；传了更贴近浏览器真实请求")
		isKC     = flag.Int("isKC", 0, "注册制板块标记，科创板/创业板/北交所为 1，其它为 0")
		referer  = flag.String("referer", "", "自定义 Referer，默认使用站点首页")
		baseURL  = flag.String("baseURL", "", "自定义接口站点，可用于代理调试")
		asJSON   = flag.Bool("json", false, "以 JSON 输出完整字段")
		timeout  = flag.Duration("timeout", 30*time.Second, "整体超时时间")
	)
	flag.Parse()
	if *code == "" && *wd == "" {
		return errors.New("必须指定 -code 或 -wd，例如 -code 899050 / -wd 北证50")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	client := finscope.New(finscope.Config{BaseURL: *baseURL, Referer: *referer})
	if *wd != "" {
		return runSuggest(ctx, client, *wd, *itemType)
	}
	result, err := client.GetQuoteKLine(ctx, finscope.GetQuoteKLineRequest{
		Code:       *code,
		MarketType: *market,
		Name:       *name,
		IsKC:       *isKC,
		KType:      *ktype,
		Count:      *count,
	})
	if err != nil {
		return err
	}
	if len(result.Bars) == 0 {
		fmt.Println("接口返回正常，但没有任何 K 线数据（可能停牌或代码不存在）")
		return nil
	}
	if *asJSON {
		return printJSON(result)
	}
	printTable(result)
	return nil
}

// runSuggest 用搜索接口按名称查询，输出候选及其行情快照。
//
// 该接口不参与行情接口的风控，适合在 getquotation 被拒时确认代码与最新点位。
func runSuggest(ctx context.Context, client *finscope.Client, wd, itemType string) error {
	items, err := client.FindSuggest(ctx, wd, itemType)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Printf("没有匹配 %q 的条目\n", wd)
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "代码\t名称\t类型\t交易所\t最新价/点位\t涨跌幅")
	for _, item := range items {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Code, item.Name, item.Type, item.Exchange, item.Price, item.Ratio)
	}
	_ = w.Flush()
	fmt.Printf("\n共 %d 条\n", len(items))
	return nil
}

// printJSON 输出解析后的 K 线与原始列名。
func printJSON(result *finscope.KLineResult) error {
	payload := struct {
		Keys   []string            `json:"keys"`
		Bars   []finscope.KLineBar `json:"bars"`
		Count  int                 `json:"count"`
		Source string              `json:"source"`
	}{
		Keys:  result.Keys,
		Bars:  result.Bars,
		Count: len(result.Bars),
		Source: "baidu finance.pae.baidu.com /vapi/v1/getquotation" +
			" group=quotation_kline_ab",
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化结果失败：%w", err)
	}
	fmt.Println(string(data))
	return nil
}

// printTable 输出时间、开高低收、涨跌幅与成交量。
func printTable(result *finscope.KLineResult) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "时间\t开盘\t最高\t最低\t收盘\t涨跌幅\t成交量")
	for _, bar := range result.Bars {
		turnover := "-"
		if bar.TurnoverRatio != nil {
			turnover = fmt.Sprintf("%.2f%%", *bar.TurnoverRatio)
		}
		_ = turnover
		fmt.Fprintf(w, "%s\t%.2f\t%.2f\t%.2f\t%.2f\t%+.2f%%\t%.0f\n",
			bar.Time, bar.Open, bar.High, bar.Low, bar.Close, bar.ChangeRatio, bar.Volume)
	}
	_ = w.Flush()
	fmt.Printf("\n共 %d 条（跳过残缺记录 %d 条）\n", len(result.Bars), result.Skipped)
}
