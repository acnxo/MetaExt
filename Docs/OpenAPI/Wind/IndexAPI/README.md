# 万得官网 IndexAPI 接口

本项目用到的万得官网端点：指数日线（含成交量与成交额），不需要 Wind 终端。

## 文件

| 文件 | 接口 | 用途 |
| --- | --- | --- |
| [GetIndexKLine.md](GetIndexKLine.md) | `GetIndexKLine` | 万得微盘股指数的日线 |

## 公共约定

- 公开网页 API，**无需 Wind 终端、无需登录**，可直接在 CI 里跑；
- `indexId` 是官网内部编码（Base64 形态），与 `windCode` 不是一套，需映射表；
- 只返回最近约 60 个交易日，必须与本地数据按日期合并去重；
- SSL 证书是 `*.wind.com.cn`，需关闭主机名校验但保留证书链校验；
- 最高价字段名拼写为 `hight`，不是 `high`。

## 代码入口

- `component/quote/wind/indexapi/client.go`：配置、TLS 配置、`APIError`
- `component/quote/wind/indexapi/indexkline.go`：`GetIndexKLine`、日期换算与去重
- 映射表：`indexapi.DefaultIndexIDs`
