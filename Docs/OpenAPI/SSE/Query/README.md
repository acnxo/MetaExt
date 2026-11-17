# 上交所 Query 接口

本项目用到的上交所端点：A 股列表与上市日期。站点为 `query.sse.com.cn`，
因此按子域名归为 `Query` 组。

## 文件

| 文件 | 接口 | 用途 |
| --- | --- | --- |
| [GetStockList.md](GetStockList.md) | `GetStockList` | 沪市 A 股代码、简称与上市日期 |

## 公共约定

- 站点只支持 **HTTP**（`http://query.sse.com.cn`）；
- 必须带 `Referer: http://www.sse.com.cn/`；
- `sqlId` 固定为 `COMMON_SSE_CP_GPJCTPZ_GPLB_GP_L`，`pageSize` 设 3000 即可一次取全；
- `LIST_DATE` 为 `yyyyMMdd`，不是八位数字的行直接丢弃。

## 代码入口

- `component/quote/sse/query/client.go`：配置、`APIError`
- `component/quote/sse/query/stocklist.go`：`GetStockList`、日期校验
