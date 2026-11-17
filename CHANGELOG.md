# 变更日志

本文件记录本库的所有重要变更。版本号遵循语义化版本，格式参考 Keep a Changelog。
发版规范见 [Docs/DevSpec/ReleaseSpec.md](Docs/DevSpec/ReleaseSpec.md)。

分类：`Added` 新增、`Changed` 变更、`Deprecated` 弃用、`Removed` 移除、`Fixed` 修复、
`Security` 安全。不兼容变更标注 `BREAKING`。

## [Unreleased]

### Added

- `FlexInt64`：兼容「数字」与「字符串数字」的整数类型，`null` / 空串解析为 0，
  非数字文本与小数返回错误；`MarshalJSON` 仍输出数字
- `xetapp.Client.GetQuoteSnapshotCNAETF`：沪深全量 ETF 行情快照（`action=20192` +
  板块中文名 `沪深ETF基金`），一次请求拿全量，字段集与 A 股接口同一套
- `xetapp.Client.GetQuoteSnapshotCNAIndex`：沪深京全量指数，串行拉取 4 个板块池
  （`20191` / `20192` / `20198`）并按证券代码去重，同时返回各池的抓取统计
- `xetapp.Row` 的基金 / ETF 强类型字段：`Date`、`UnitNetValue`、`AccNetValue`、
  `FundNav`、`FundScale`、`DiscountRate`、`ETFClass1`、`ETFClass2`、
  `TrackIndexCode`、`TrackIndexName`
- `component/quote/internal/fetch`：行情接口共用的 GET 请求层（退避重试、响应体上限、
  GBK/UTF-8 解码），供下列新包复用
- `eastmoney/push2`：`GetQuoteList`（全市场快照，镜像域名轮换、`data` 为空时换镜像）、
  `GetKLine`（日线，含真实成交额）、`GetTradingCalendar`（上证指数日线取交易日）
- `tencent/gtimg`：`GetQuoteSnapshot`（GBK 批量快照，900 只分批、414 时二分降级）、
  `GetKLine`（日线，支持前复权）
- `sina/finance/kline`：`GetKLine`（长历史日线，无成交额）
- `xueqiu/stock`：`GetKLine`（全历史日线，含成交量与成交额；cookie 预热与失效重建）
- `sse/query`：`GetStockList`（沪市 A 股列表与上市日期）
- `szse/webapi`：`GetStockList`（深市 A 股列表与上市日期，并发翻页）
- `cninfo/dataapi`：`GetStockList`（深市代码清单，一次请求取全）
- `csindex/indexapi`：`GetIndexPerf`（中证指数日行情，按日期去重）
- `wind/indexapi`：`GetIndexKLine`（万得官网指数日线，只跳过 TLS 主机名校验）

### Changed

- 引入首个第三方依赖 `golang.org/x/text`（腾讯快照需 GBK 解码），新增 `go.sum`；
  CI 与 Release 工作流的 `cache-dependency-path` 改为 `go.mod` + `go.sum`
- 11 份新接口说明统一补充「限制与配额」一节：证券范围、分页/批量上限、历史区间、
  最早数据日期与限流经验；站点未公布配额的项一律标注「未实测」，不做推测
- `xetapp`：把 A 股快照的请求组装与解析抽成共用实现（`fetchBoardSnapshot`），
  `GetQuoteSnapshotCNAStock` 的行为与签名不变；新增动作号常量
  `ActionQuoteSnapshotETF` / `ActionQuoteSnapshotByMarket` / `ActionQuoteSnapshotMarketIndex`

### Fixed

- `GetFuturesList` 的 `stockId` 兼容字符串编码（线上实际返回 `"70000294"`）。此前按
  `int64` 声明且无自定义解码，`json.Unmarshal` 在首个不匹配字段即失败、整页数据丢失，
  接口对任何调用方不可用；字段类型与调用方代码均不变（非破坏）
- `GetFuturesList` 的 `fieldId` / `precision` / `contentType` 与 `fields` 的值同样兼容
  字符串与数字两种编码，避免同一接口后续字段变换编码形态时再次整页解码失败
- `tencent/gtimg`、`sina/finance/kline`、`xueqiu/stock` 的代码前缀推断：
  `4` / `8` 开头的北交所代码此前被误拼成 `sz` / `SZ`，现改为 `bj` / `BJ`；
  东财 `secid` 对北交所使用 `0.` 前缀，不受影响
- `xetapp` 的字段码 `22` 由 `nav_date`（字符串）纠正为 `growth_rate`（百分比），
  与上游脚本一致；`Row.Values` 中该 key 随之改名 **BREAKING**
- `xetapp` 补上字段码 `326`（`fund_scale`，基金规模），此前按 `f326` 输出且未识别类型

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
