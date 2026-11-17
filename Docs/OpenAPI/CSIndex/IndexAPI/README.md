# 中证指数 IndexAPI 接口

本项目用到的中证指数官网端点：指数日行情，用于基准对比。

## 文件

| 文件 | 接口 | 用途 |
| --- | --- | --- |
| [GetIndexPerf.md](GetIndexPerf.md) | `GetIndexPerf` | 中证 2000（`932000`）、中证全指（`000985`）日行情 |

## 公共约定

- 必须带 `Referer: https://www.csindex.com.cn/`；
- 指数有发布日，早于发布日的区间无数据（中证 2000 为 2023-08-11）；
- `tradeDate` 可能是 `yyyyMMdd` 或 `yyyy-MM-dd`，本库统一归一化；
- 官网偶有重复行，按日期去重保留最后一条；
- 只给 OHLC 与涨跌，不含成交量与成交额。

## 代码入口

- `component/quote/csindex/indexapi/client.go`：配置、`APIError`
- `component/quote/csindex/indexapi/indexperf.go`：`GetIndexPerf`、日期归一化与去重
