# 变更日志

本文件记录本库的所有重要变更。版本号遵循语义化版本，格式参考 Keep a Changelog。
发版规范见 [Docs/DevSpec/ReleaseSpec.md](Docs/DevSpec/ReleaseSpec.md)。

分类：`Added` 新增、`Changed` 变更、`Deprecated` 弃用、`Removed` 移除、`Fixed` 修复、
`Security` 安全。不兼容变更标注 `BREAKING`。

## [Unreleased]

## [0.1.2] - 2026-11-08

`quoteapiv2` 支持自定义请求头与中文站来源页，并新增两个外汇接口。

### Added

- `Config.Header`：额外请求头，在内置默认头之后逐项覆盖；`New` 时浅拷贝，避免共享 map 引发竞态
- `Config.Referer` / `Config.Referers`：全局与按 `marketType` 覆盖请求来源页
- `DefaultReferers`：港股 / 美股 / 沪深分别指向对应市场的简体中文列表页
- `GetFxRateList`：按清单分组获取汇率列表，支持 0-based 分页与 `listId` 分组
- `GetFxOverview`：一次获取 9 币种两两组合的 72 个货币对最新价
- 上述两个接口的 OpenAPI 规格与同名说明文档

### Changed

- `DefaultReferer` 由站点根路径改为简体中文站 `https://www.moomoo.com/hans/`，
  返回的证券名称与数值单位随之为中文口径（`亿/万` 而非 `B/M/K`）；需要英文口径时
  显式设置 `Config.Referer` 或 `Config.Header["referer"]` **BREAKING**
- 列表类接口（股票 / ETF / 期货）的 `referer` 改为按 `marketType` 选择，
  其它接口保持全局默认
- 包注释的接口清单同步为当前十个接口

### Fixed

- `GetStockList` 的 `page` 下限由 1 改为 0，与接口的 0-based 语义对齐；
  此前恒定跳过第一页（港股按市值采集缺前 20 名，全量少 `pageSize` 条），
  且错误由客户端本地产生、易被误判为服务端要求 1-based **BREAKING**（缺省请求现在包含第一页）
- `GetFxOverview` 省略参数时按空对象 `{}` 计算签名，避免落入空串降级分支导致 `-403`

## [0.1.1] - 2026-11-03

补充新浪交易日历与版本号机制；本段为 2026-11-08 回溯补录。

### Added

- `GetCNATradeCalendar`：在 `GetTradeCalendar` 结果上补齐数据源缺口，返回 8797 天
- 新浪交易日历客户端与规格文档，规格目录调整为 `Sina/Finance/CnaTradeCalendar`
- `VERSION` 文件与 `base/version` 包：版本号唯一来源，构建时由 CI 注入
- 发版、提交信息与 PR 的规范文档
- 新浪交易日历解码的故障排查记录

### Changed

- CI 工作流输出交易日历周表；`AutoTag` 按 `VERSION` 打标签并创建 Release
- 交易日历规格与代码实现对账，导出 `ShanghaiOpenDay` / `ShenzhenOpenDay`

### Fixed

- 沪市日历缺少 1992-05-04，`GetCNATradeCalendar` 补齐该缺口

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
