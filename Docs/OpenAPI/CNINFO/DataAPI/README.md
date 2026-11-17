# 巨潮资讯 DataAPI 接口

本项目用到的巨潮端点：深市代码清单（一次请求拿全）。接口路径为
`/new/data/szse_stock.json`，按数据接口路径归为 `DataAPI` 组。

## 文件

| 文件 | 接口 | 用途 |
| --- | --- | --- |
| [GetStockList.md](GetStockList.md) | `GetStockList` | 深市全部代码与简称，**不含上市日期** |

## 公共约定

- 站点只支持 **HTTP**（`http://www.cninfo.com.cn`）；
- 一次返回全部深市证券，无分页，是「代码清单」的首选路径；
- 响应可能包在 `stockList` 下，也可能直接是数组，两种都要兼容；
- 缺上市日期，需由东财 `f26` 或本地上次落盘的清单补齐。

## 代码入口

- `component/quote/cninfo/dataapi/client.go`：配置、`APIError`
- `component/quote/cninfo/dataapi/stocklist.go`：`GetStockList`、双形态解析
