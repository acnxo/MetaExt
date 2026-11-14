# 市场编码字典（market / market_type）

> 所属：百度股市通接口文档 · 存放：`Docs/OpenAPI/Baidu/FinScope/Market.md`\
> 用途：`constituents` 请求参数 `market` 与 `getquotation` 请求参数 `market_type` 的取值字典——**市场编码为键，市场中文名为值**，\
> 同一编码也出现在响应字段 `market` 与个股页地址 `stock/<market>-<code>` 中。

## 一、字典表

| 编码 | 市场 | 代码格式 | 代码示例 | 个股页地址 | 验证状态 |
|------|------|------|------|------|:---:|
| `ab` | A 股（沪 / 深 / 北）/ 国内商品期货 | 6 位数字 / 品种+合约 | `601939` / `AU888` | `https://finance.baidu.com/stock/ab-601939` | 证券实测 |
| `hk` | 港股 | 5 位数字，含前导零 | `01810` | `https://finance.baidu.com/stock/hk-01810` | 页面 URL 推断 |
| `us` | 美股 / 美国市场期货 | 大写字母代码 / 品种+年月码 | `NVDA` / `GC26Z` | `https://finance.baidu.com/stock/us-NVDA` | 证券：页面 URL 推断 |

## 二、说明

- 编码：`market` / `market_type` 的**原始取值**，小写两字母，同时也是响应 `market` 字段值与个股页 URL 的路径片段。
- 参数名差异：`constituents` 用 `market`，`getquotation`（K 线、五日分时）用 `market_type`，两者取值同一套，不另建映射。
- A 股只有一个编码 `ab`，不区分沪 / 深 / 北；交易所由响应字段 `exchange`（`SH` / `SZ` / `BJ`）区分。
- 代码格式随编码变化：A 股 6 位数字（含指数 `000300`）；港股 5 位数字且**保留前导零**
  （`01810`，不能写成 `1810`）；美股为交易所代码，保留大写字母（`NVDA`）。
- 个股页地址固定为 `https://finance.baidu.com/stock/<market>-<code>`，可据此反推编码与代码格式。
- 排序：按编码字母序 `ab` < `hk` < `us`。
- 验证状态：`ab` 为本机实测；`hk` / `us` 由个股页 URL 前缀得到，`getquotation` 在本机持续风控
  （`{"code":403,"isCaptchaEnabled":true,"msg":"hit risk"}`），两项未实测。
- `group` 参数名自带市场后缀（`quotation_kline_ab`、`quotation_fiveday_ab`），切换市场时需同时确认
  `group` 是否仍可用，不能只改 `market_type`。
- 当前只验证了 A 股；`hk` / `us` 仅作为接口能力记录。
- 外汇不在本字典范围：`huilv_minute` 接口不用 `market` / `market_type`，货币对直接写在
  `code`（`USDCNY`、`BTCUSD`），见 [GetQuoteForexMinute.md](GetQuoteForexMinute.md)。
- 期货语境：同一套编码下，`ab` 指国内商品期货（`AU888` 主力连续、`AU2612` 具体合约），
  `us` 指美国市场期货（`GC26Z`，COMEX 黄金 2026-12）；靠接口路径与 `financeType` /
  `group` 区分，不能单看编码判断品种。期货入口见
  [GetQuoteFuturesMinute.md](GetQuoteFuturesMinute.md)、
  [GetQuoteFuturesFiveDay.md](GetQuoteFuturesFiveDay.md)；外汇见
  [GetQuoteForexMinute.md](GetQuoteForexMinute.md)。
- 期货两项的编码来自已确认的请求 URL，响应结构未实测（本机风控），故不写「实测」。

## 三、维护约定

- 本文件描述的是**百度股市通侧的原始取值**，取值集合由上游决定；上游变更后同步本文件，
  不在此处自行裁剪或新增条目。
- 新增编码按字母序插入，并同时补齐代码格式、代码示例、个股页地址与验证状态；
  未实测的写 `页面 URL 推断`。
- 各接口文档只引用本文件，不再重复维护取值表；改动编码说明时先改这里，再检查
  `README.md`、`GetConstituents.md`、`GetQuoteKLine.md`、`GetQuoteFiveDay.md` 中的引用是否仍准确。
