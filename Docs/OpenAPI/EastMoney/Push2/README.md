# 东方财富 Push2 行情接口

本项目用到的东财行情端点。它们部署在同集群的两个域名下：列表快照在
`push2.eastmoney.com`，历史 K 线在 `push2his.eastmoney.com`，因此按域名归为一组。

## 文件

| 文件 | 接口 | 用途 |
| --- | --- | --- |
| [GetQuoteList.md](GetQuoteList.md) | `GetQuoteList` | 全市场快照，可按总市值排序翻页 |
| [GetKLine.md](GetKLine.md) | `GetKLine`、`GetTradingCalendar` | 日线（含成交额）与交易日历 |

## 公共约定

- 两组域名各有多个镜像子域（`82.` / `1.` / `7.` / 主域），按顺序轮换，主域偶发 502；
- 必须带 `Referer: https://quote.eastmoney.com/`；
- `data=null` 表示被限频，换镜像重试；本库不主动 sleep，节奏由调用方控制；
- 市值单位是**元**（腾讯是亿元），成交量单位是**手**。

## 代码入口

- `component/quote/eastmoney/push2/client.go`：`Config`、`New`、镜像轮换、`APIError`
- `component/quote/eastmoney/push2/quotelist.go`：`GetQuoteList`、`decodeQuoteRow`
- `component/quote/eastmoney/push2/kline.go`：`GetKLine`、`GetTradingCalendar`
- 常量：`DefaultListHosts`、`DefaultKLineHosts`、`UniverseMainBoard`、`UniverseAllA`
