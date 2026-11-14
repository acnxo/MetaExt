# Meta

本目录存放**机器可读**的接口规格与元数据，面向工具消费：OpenAPI 预览、代码生成、lint、
接口对比等。叙述性说明不放在这里，见 `Docs/`。

## 内容

| 路径 | 说明 |
| --- | --- |
| `OpenAPI/<厂商>/<接口组>/<接口名>.yaml` | 外部 HTTP 接口的 OpenAPI 3.0.3 规格，一个接口一个文件 |
| `FT/Dict/<字典名>.md` | 富途 / moomoo 字段取值字典，一个字段一个文件 |

当前已收录：

```
Meta/OpenAPI/
├── Baidu/FinScope/GetConstituents.yaml     # 成分股列表（含行情快照）
├── Baidu/FinScope/GetQuoteKLine.yaml       # 分钟 / 日级 K 线
├── Baidu/FinScope/GetQuoteFiveDay.yaml     # 近五个交易日分钟分时
├── Baidu/FinScope/GetQuoteForexMinute.yaml # 外汇分钟行情（占位）
├── Baidu/FinScope/GetQuoteFuturesMinute.yaml  # 期货分钟行情（占位）
├── Baidu/FinScope/GetQuoteFuturesFiveDay.yaml # 期货五日分时（占位）
├── MooMoo/QuoteAPIV2/GetKLine.yaml         # 行情 K 线
├── MooMoo/QuoteAPIV2/GetStockList.yaml     # 板块股票列表
├── MooMoo/QuoteAPIV2/GetOptionQuote.yaml   # 期权批量报价
├── MooMoo/QuoteAPIV2/GetNewsList.yaml      # 个股新闻列表
├── MooMoo/QuoteAPIV2/GetEtfList.yaml       # ETF 列表
├── MooMoo/QuoteAPIV2/GetFuturesList.yaml   # 期货合约列表
├── MooMoo/QuoteAPIV2/GetHistoryCashTrend.yaml  # 历史资金流水
├── MooMoo/QuoteAPIV2/GetQuoteMinute.yaml   # 分钟级分时行情
└── Sina/Finance/CnaTradeCalendar/GetTradeCalendar.yaml  # A 股交易日历

Meta/FT/
└── Dict/
    ├── CsMarket.md                         # 富途 cs_market 字段取值字典
    ├── CurrencyCode.md                     # 富途 currency_code 字段取值字典
    ├── Exchange.md                         # 富途 exchange 字段取值字典
    ├── InstrumentType.md                   # 富途 instrument_type 字段取值字典（已过时，见 V2）
    ├── InstrumentTypeV2.md                 # 富途 instrument_type_v2 字段取值字典
    ├── ListedExchange.md                   # 富途 listed_exchange 字段取值字典
    ├── MarketCode.md                       # 富途 market_code 字段取值字典
    ├── MarketLabel.md                      # moomoo marketLabel 响应字段取值字典
    ├── MarketType.md                       # moomoo market_type / marketType 请求参数取值字典
    ├── Region.md                           # 富途 region 字段取值字典
    ├── SubInstrumentType.md                # 富途 sub_instrument_type 字段取值字典（已过时，见 V2）
    └── SubInstrumentTypeV2.md              # 富途 sub_instrument_type_v2 字段取值字典
```

## 与 Docs 的分工

| 目录 | 定位 | 内容 |
| --- | --- | --- |
| `Meta/` | 给机器读 | OpenAPI 规格、字段取值字典；只放结构化的规格与稳定口径 |
| `Docs/` | 给人读 | 与规格同名的说明、签名规则、目录索引、开发规范 |

两侧目录层级与文件名保持一致，例如：

```
Meta/OpenAPI/MooMoo/QuoteAPIV2/GetKLine.yaml
Docs/OpenAPI/MooMoo/QuoteAPIV2/GetKLine.md
```

完整约定见 [`Docs/DevSpec/HTTPAPIDocSpec.md`](../Docs/DevSpec/HTTPAPIDocSpec.md) 第九节。

## 命名约定

- `<厂商>`：来源的稳定名称，如 `MooMoo`、`Sina`；
- `<接口组>`：接口路径中的版本或系列名，如 `QuoteAPIV2`、`TradeCalendar`；
- `<接口名>`：与 Go 客户端方法名、`operationId` 三者一致，如 `GetKLine`。

不再向下拆更多层级。

## 新增一个接口规格

1. 按命名约定建目录，写 `<接口名>.yaml`，使用 OpenAPI 3.0.3；
2. 在 `Docs/OpenAPI/` 对应目录写同名 `.md`，并更新该目录的 `README.md` 索引；
3. 校验 YAML 可解析、`$ref` 指向的组件都存在；
4. 与实现代码、说明文档一起提交，不要只提交其中一部分。

## 新增一个字段字典

1. 在 `Meta/<厂商>/Dict/` 下建 `<字段名>.md`，文件名用大驼峰；
2. 顶部引用块写清所属、用途、数据来源与关联字典，正文给取值表；
3. 取值来自上游时注明统计时点，「数量」一类统计值只作参考，不当成枚举约束；
4. 厂商目录有多个字典时补 `README.md` 索引，并更新本文件的收录列表。

## 编写要求

- 响应用 `code` / `message` / `data` 描述，至少给出一份成功示例；
- 可复用结构放 `components.schemas`，`$ref` 不得悬空；
- `$ref` 旁不要放会被忽略的兄弟字段（`description`、`nullable` 等），需要补充说明时用
  `allOf` 包裹；
- 字段类型混用时用 `oneOf` 或具名 schema（如 `FlexNumber`）描述，并说明取值形态；
- 查询参数的书写顺序与代码中的构造顺序一致，涉及签名的接口尤其不能随意调整。

## 占位规格

尚未实测确认的接口可以先提交占位骨架，但必须：

- 在文件顶部写明用途与状态，标注 `x-status: placeholder`；
- 只记录已确认的事实，不编造请求头、响应字段与示例；
- 用 `x-todo` 列出待确认项；
- 确认完成后去掉 `x-status`，补齐内容。

参考 [`OpenAPI/Sina/Finance/CnaTradeCalendar/GetTradeCalendar.yaml`](OpenAPI/Sina/Finance/CnaTradeCalendar/GetTradeCalendar.yaml)。
