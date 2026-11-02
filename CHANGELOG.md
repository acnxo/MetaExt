# 变更日志

本文件记录本库的所有重要变更。版本号遵循语义化版本，格式参考 Keep a Changelog。
发版规范见 [Docs/DevSpec/ReleaseSpec.md](Docs/DevSpec/ReleaseSpec.md)。

分类：`Added` 新增、`Changed` 变更、`Deprecated` 弃用、`Removed` 移除、`Fixed` 修复、
`Security` 安全。不兼容变更标注 `BREAKING`。

## [Unreleased]

## [0.1.0] - 2026-11-02

首个可用版本，提供 moomoo（富途）网页行情接口 `/quote-api/quote-v2/*` 的客户端封装。

### Added

- 客户端骨架：`Config` 零值可用，可注入 `HTTPClient` 与时钟；GET/POST 两种请求与签名方式
- 签名：`Sign`、`SignInput`、`SignParams`、`SignedQuery`，支持查询串与请求体两种签名输入
- `GetKLine`：K 线全量序列
- `GetStockList`：板块股票列表，支持排序与分页
- `GetOptionQuote`：期权合约批量报价（POST，签名基于请求体原始字节）
- `GetNewsList`：个股新闻列表
- `GetEtfList`：ETF 列表
- `GetFuturesList`：期货合约列表，位置化字段结构
- `GetHistoryCashTrend`：历史资金流水
- `GetQuoteMinute`：分钟级分时行情
- 错误处理：`APIError`、`IsTokenRejected`、`IsParamsError`、`IsParamsMissing`
- 各接口的 OpenAPI 3.0.3 规格与同名说明文档，以及签名算法与校验向量
- 单元测试 64 个用例，覆盖率约 97%
- CI/CD 工作流：`CICD`（格式检查、vet、构建、测试、覆盖率）与 `Release`（tag 触发发布）

### Security

- 不含任何凭据；签名密钥取自网页前端公开产物，非用户凭据
