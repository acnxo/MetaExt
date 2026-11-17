# 雪球 Stock 接口

本项目用到的雪球端点：全历史日线，是历史行情采集的**主源**。接口路径为
`/v5/stock/chart/kline.json`，因此归为 `Stock` 组。

## 文件

| 文件 | 接口 | 用途 |
| --- | --- | --- |
| [GetKLine.md](GetKLine.md) | `GetKLine` | 全历史日线（最多 8000 根），含成交量与成交额 |

## 公共约定

- **需要 cookie**：先访问 `https://xueqiu.com/` 预热会话，再用同一 cookie jar 请求；
- 响应用列式结构（`column` + `item`），必须按列名查下标；
- `volume` 单位是**股**、`amount` 单位是**元**，都无需换算；
- cookie 失效时 `item` 为空，本库重建会话并重试一次。

## 代码入口

- `component/quote/xueqiu/stock/client.go`：配置、cookie jar、预热与失效重建
- `component/quote/xueqiu/stock/kline.go`：`GetKLine`、列式解析、日期换算
