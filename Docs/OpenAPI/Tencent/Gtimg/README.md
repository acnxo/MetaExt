# 腾讯 Gtimg 接口

本项目用到的腾讯行情端点：批量快照（主源）与日线。两者都在
`gtimg.cn` 域下（`qt.gtimg.cn` 与 `web.ifzq.gtimg.cn`），因此归为 `Gtimg` 组。

## 文件

| 文件 | 接口 | 用途 |
| --- | --- | --- |
| [GetQuoteSnapshot.md](GetQuoteSnapshot.md) | `GetQuoteSnapshot` | 批量快照，总市值/流通市值锚点 |
| [GetKLine.md](GetKLine.md) | `GetKLine` | 日线（支持前复权），本项目暂无默认调用点 |

## 公共约定

- 快照响应是 **GBK 文本**，不是 JSON，需按 GBK 解码后按 `~` 拆分；
- 单批最多约 900 只（URL 约 8.1 KB），超过触发 HTTP 414，本库自动二分降级；
- 市值单位是**亿元**（×1e8 得元），成交额单位是**万元**（×1e4 得元），成交量单位是**手**；
- 快照不返回上市日期与股本，需由交易所清单与「市值 ÷ 价格」补齐。

## 代码入口

- `component/quote/tencent/gtimg/client.go`：配置、`APIError`、`symbolOf`
- `component/quote/tencent/gtimg/snapshot.go`：`GetQuoteSnapshot`、分批与二分降级
- `component/quote/tencent/gtimg/kline.go`：`GetKLine`
