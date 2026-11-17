# 深交所 WebAPI 接口

本项目用到的深交所端点：A 股列表与上市日期（分页约 146 页）。站点为
`www.szse.cn/api/report/...`，因此按 `/api/` 路径归为 `WebAPI` 组。

## 文件

| 文件 | 接口 | 用途 |
| --- | --- | --- |
| [GetStockList.md](GetStockList.md) | `GetStockList` | 深市 A 股代码、简称与上市日期 |

## 公共约定

- 站点只支持 **HTTP**（`http://www.szse.cn`）；
- 每页 20 条，需先取第 1 页的 `metadata.pagecount` 再并发翻其余页（默认 8 并发）；
- 简称字段带 `<u>` / `<a>` 高亮标签，需两步清理；
- 任一分页失败即整体报错，不返回残缺结果。

## 代码入口

- `component/quote/szse/webapi/client.go`：配置、`APIError`
- `component/quote/szse/webapi/stocklist.go`：`GetStockList`、并发翻页、名称清理
