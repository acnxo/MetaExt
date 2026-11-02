# QuoteAPIV2 接口规格

本目录记录 moomoo（富途）网页端行情接口 `/quote-api/quote-v2/*` 的 HTTP 规格。

每个接口由两份文件描述：机器可读的 OpenAPI 3.0.3 规格（`.yaml`）保存在仓库的
`Meta/OpenAPI/MooMoo/QuoteAPIV2/` 下，面向人阅读的说明（`.md`）保存在本目录，两份
文件同名。

## 文件

| 文件 | 位置 | 说明 |
| --- | --- | --- |
| `GetKLine.yaml` | `Meta/OpenAPI/MooMoo/QuoteAPIV2/` | `GET /quote-api/quote-v2/get-kline` 的 OpenAPI 规格 |
| `GetKLine.md` | 本目录 | 同一接口的可读说明 |
| `GetStockList.yaml` | `Meta/OpenAPI/MooMoo/QuoteAPIV2/` | `GET /quote-api/quote-v2/get-stock-list` 的 OpenAPI 规格 |
| `GetStockList.md` | 本目录 | 同一接口的可读说明 |
| `GetOptionQuote.yaml` | `Meta/OpenAPI/MooMoo/QuoteAPIV2/` | `POST /quote-api/quote-v2/get-option-quote` 的 OpenAPI 规格 |
| `GetOptionQuote.md` | 本目录 | 同一接口的可读说明 |
| `GetNewsList.yaml` | `Meta/OpenAPI/MooMoo/QuoteAPIV2/` | `GET /quote-api/quote-v2/get-news-list` 的 OpenAPI 规格 |
| `GetNewsList.md` | 本目录 | 同一接口的可读说明 |
| `GetEtfList.yaml` | `Meta/OpenAPI/MooMoo/QuoteAPIV2/` | `GET /quote-api/quote-v2/get-etf-list` 的 OpenAPI 规格 |
| `GetEtfList.md` | 本目录 | 同一接口的可读说明 |
| `GetFuturesList.yaml` | `Meta/OpenAPI/MooMoo/QuoteAPIV2/` | `GET /quote-api/quote-v2/get-futures-list` 的 OpenAPI 规格 |
| `GetFuturesList.md` | 本目录 | 同一接口的可读说明 |
| `GetHistoryCashTrend.yaml` | `Meta/OpenAPI/MooMoo/QuoteAPIV2/` | `GET /quote-api/quote-v2/get-history-cash-trend` 的 OpenAPI 规格 |
| `GetHistoryCashTrend.md` | 本目录 | 同一接口的可读说明 |
| `GetQuoteMinute.yaml` | `Meta/OpenAPI/MooMoo/QuoteAPIV2/` | `GET /quote-api/quote-v2/get-quote-minute` 的 OpenAPI 规格 |
| `GetQuoteMinute.md` | 本目录 | 同一接口的可读说明 |
| `Signature.md` | 本目录 | 请求签名 `quote-token` 的算法与校验向量，适用于本组全部接口 |

## 命名约定

- 文件名取接口路径末段的驼峰形式，与 Go 客户端方法名保持一致：
  `get-kline` → `GetKLine`、`get-stock-list` → `GetStockList`、
  `get-option-quote` → `GetOptionQuote`、`get-news-list` → `GetNewsList`、
  `get-etf-list` → `GetEtfList`、`get-futures-list` → `GetFuturesList`、
  `get-history-cash-trend` → `GetHistoryCashTrend`、`get-quote-minute` → `GetQuoteMinute`，
  分别对应同名客户端方法；
- 新增接口时按同一规则分别添加 `.yaml` 与同名 `.md`：`.yaml` 放 `Meta/OpenAPI/` 下
  对应厂商与接口组目录，`.md` 放本目录。

## 与代码的对应关系

- 规范中的 `operationId` 与 Go 客户端方法名一致：`getKLine` 对应
  `component/quote/moomoo/quoteapiv2` 包中的 `Client.GetKLine`；
- 查询参数在规范中的书写顺序就是签名输入的键序，与代码里的参数构造顺序一致；
- 响应结构 `KLineResponse` 对应代码中的 `klinePayload`、`KLineItem` 对应 `klineItem`；
- 股票列表的响应结构 `StockListResponse` 对应代码中的 `stockListPayload`，`StockListItem`
  与 `Pagination` 与代码同名；
- 期权报价的 `OptionQuoteResponse` 对应代码中的 `optionQuotePayload`，`OptionQuote`
  与代码同名，`data` 解为以合约 ID 为键的 map；
- 新闻列表的 `NewsListResponse` 对应代码中的 `newsListPayload`，`NewsListItem` 与
  `NewsListPage` 与代码同名，`data` 中的 snake_case 字段对应 `NewsListPage` 字段；
- ETF 列表的 `EtfListResponse` 对应代码中的 `etfListPayload`，条目复用 `StockListItem`，
  结果结构为 `EtfListPage`。
- 期货列表的 `FuturesListResponse` 对应代码中的 `futuresListPayload`，`FuturesListItem`
  与 `FuturesListPage` 与代码同名，位置化字段对应 `FuturesField`。
- 资金流水的 `CashTrendResponse` 对应代码中的 `cashTrendPayload`，`CashTrendItem` 与
  `CashTrendPage` 与代码同名。
- 分时行情的 `QuoteMinuteResponse` 对应代码中的 `quoteMinutePayload`，`QuoteMinute`、
  `QuoteMinuteTick`、`QuoteMinuteSection` 与代码同名。
- 签名算法与校验向量见 `Signature.md`，不对应具体接口，同一目录下的接口共用。

新增接口时，先补齐规格文档，再按同一命名规则实现客户端方法。

## 查看与校验

`.yaml` 是标准 OpenAPI 3.0 文档，可用任意兼容 OpenAPI 3.0 的工具预览、生成文档或
做 lint，例如编辑器插件、Swagger UI、Redoc 一类的查看器。

改动后用 YAML 解析即可确认语法，并检查 `$ref` 指向的组件是否存在。

## 维护约定

- 规范依据网页端实际发出的请求与返回的响应整理，不是厂商公开发布的接口文档；
- 站点行为发生变化时，先实测确认，再同步更新规格与说明，最后修改实现，避免文档与
  代码互相矛盾；
- 字段口径以响应原始形态为准：数值可能是字符串、也可能是数字，缺失时可能为 `null`，
  规范中用 `FlexNumber` 统一描述。
