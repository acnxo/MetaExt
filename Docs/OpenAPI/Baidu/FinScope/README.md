# 百度股市通接口文档（FinScope）

百度股市通（`finance.pae.baidu.com`）行情接口的规格与说明，供接入、排障或替换数据源时查阅。
百度未公开官方 OpenAPI 描述，规格按实测请求与响应整理；未取得真实响应的接口标注「未验证」。

## 文件

| 文件 | 位置 | 说明 |
| --- | --- | --- |
| `GetConstituents.yaml` | `Meta/OpenAPI/Baidu/FinScope/` | `GET /sapi/v1/constituents` 的 OpenAPI 规格 |
| `GetQuoteKLine.yaml` | `Meta/OpenAPI/Baidu/FinScope/` | `GET /vapi/v1/getquotation`（`quotation_kline_ab`）的规格 |
| `GetQuoteFiveDay.yaml` | `Meta/OpenAPI/Baidu/FinScope/` | `GET /vapi/v1/getquotation`（`quotation_fiveday_ab`）的规格 |
| `GetQuoteForexMinute.yaml` | `Meta/OpenAPI/Baidu/FinScope/` | `GET /vapi/v1/getquotation`（`huilv_minute`）的规格，占位 |
| `GetQuoteFuturesMinute.yaml` | `Meta/OpenAPI/Baidu/FinScope/` | `GET /vapi/v1/getquotation`（`quotation_futures_minute`）的规格，占位 |
| `GetQuoteFuturesFiveDay.yaml` | `Meta/OpenAPI/Baidu/FinScope/` | `GET /selfselect/getstockquotation` 的规格，占位 |
| `GetConstituents.md` | 本目录 | 成分股列表的可读说明 |
| `GetQuoteKLine.md` | 本目录 | K 线的可读说明 |
| `GetQuoteFiveDay.md` | 本目录 | 五日分时的可读说明 |
| `GetQuoteForexMinute.md` | 本目录 | 外汇分钟的可读说明 |
| `GetQuoteFuturesMinute.md` | 本目录 | 期货分钟的可读说明 |
| `GetQuoteFuturesFiveDay.md` | 本目录 | 期货五日分时的可读说明 |
| `Market.md` | 本目录 | 市场编码字典（跨接口说明） |

两份文件同名成对：`Meta/` 放机器可读规格，`Docs/` 放人读说明。外汇、期货四个接口因风控
未取得真实响应，其规格标 `x-status: placeholder`，只用 `x-todo` 列出待确认项，不编造字段。

## 接口一览

| 文档 | 方法 | 路径 | group | 用途 |
| --- | --- | --- | --- | --- |
| [GetConstituents.md](GetConstituents.md) | `GET` | `/sapi/v1/constituents` | — | 指数 / 概念板块成分股列表（含行情快照） |
| [GetQuoteKLine.md](GetQuoteKLine.md) | `GET` | `/vapi/v1/getquotation` | `quotation_kline_ab` | 分钟 / 日级 K 线 |
| [GetQuoteFiveDay.md](GetQuoteFiveDay.md) | `GET` | `/vapi/v1/getquotation` | `quotation_fiveday_ab` | 近五个交易日分钟分时 |
| [GetQuoteForexMinute.md](GetQuoteForexMinute.md) | `GET` | `/vapi/v1/getquotation` | `huilv_minute` | 外汇 / 数字货币对分钟行情 |
| [GetQuoteFuturesMinute.md](GetQuoteFuturesMinute.md) | `GET` | `/vapi/v1/getquotation` | `quotation_futures_minute` | 期货分钟行情 |
| [GetQuoteFuturesFiveDay.md](GetQuoteFuturesFiveDay.md) | `GET` | `/selfselect/getstockquotation` | `quotation_futures_fiveday` | 期货五日分时 |

`/vapi/v1/getquotation` 下多个 `group` 共用路径，靠 `group` 区分返回结构：股票 K 线
`quotation_kline_ab`、股票五日 `quotation_fiveday_ab`、外汇 `huilv_minute`、期货分钟
`quotation_futures_minute`，参数与响应不能互串。期货五日走另一条路径
`/selfselect/getstockquotation`。

字段取值字典（市场编码等）统一维护在 [Market.md](Market.md)，接口文档只引用不重复维护。

指数批量行情不在本目录：改走腾讯 `qt.gtimg.cn` 接口，见根目录 README「指数采集」。

## 公共约定

### 站点与传输

| 项 | 内容 |
| --- | --- |
| 站点 | `https://finance.pae.baidu.com` |
| 协议 | HTTPS，`GET`，无请求体，不发送 `content-type` |
| 返回格式 | `application/json` |
| 认证 | 无 token、无签名；靠 `Referer` 与浏览器 UA 表明来源，靠访问频率触发风控 |

### 市场编码（`market` / `market_type`）

`constituents` 用参数名 `market`，`getquotation` 用参数名 `market_type`，两者取值是同一套：
`ab` A 股（沪 / 深 / 北）、`hk` 港股、`us` 美股。代码格式、代码示例、个股页地址、验证状态
与维护约定统一维护在字典文档 [Market.md](Market.md)，本目录其余文档只引用不重复。

期货接口复用 `ab` / `us`，含义变为国内商品期货 / 美国市场期货（如 `AU888`、`GC26Z`）；
外汇接口（`huilv_minute`）**不用** `market` / `market_type`，货币对直接写在 `code`
（`USDCNY`、`BTCUSD`）。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `User-Agent` | 是 | 建议使用浏览器 UA（如 Chrome 131 桌面版）；空 UA 或明显工具 UA 更易被风控 |
| `Referer` | 是 | `https://gushitong.baidu.com/`；个股页地址亦可 |
| `Accept` | 否 | `application/json, text/plain, */*` |

所有参数都走查询字符串，参数顺序不影响结果（接口不签名）。

### 外层信封

`constituents`、`getquotation` 与 `selfselect/getstockquotation` 均返回同一层信封：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `QueryID` | string | 请求追踪 ID，仅用于排障 |
| `ResultCode` | integer | 网关层结果，`0` 表示请求被正常处理（**不代表业务成功**） |
| `ResultNum` | integer | 部分接口返回，K 线接口实测恒为 `0` |
| `Result` | object \| array | 业务数据；业务成败一律看这一层 |

### 风控与降频

#### 触发范围（实测）

| 接口 | 路径 | group | 风控实测 |
| --- | --- | --- | --- |
| 成分股 | `/sapi/v1/constituents` | — | 未命中，可连续分页 |
| 股票 K 线 | `/vapi/v1/getquotation` | `quotation_kline_ab` | 命中 |
| 股票五日 | `/vapi/v1/getquotation` | `quotation_fiveday_ab` | 命中 |
| 外汇分钟 | `/vapi/v1/getquotation` | `huilv_minute` | 命中 |
| 期货分钟 | `/vapi/v1/getquotation` | `quotation_futures_minute` | 命中 |
| 期货五日 | `/selfselect/getstockquotation` | `quotation_futures_fiveday` | 命中，`QueryID` 可能为 `"0"` |

风控覆盖的不只是股票：外汇、期货的参数取自真实请求 URL，本机请求仍全部返回 403，
**参数正确不等于能取到数据**。

#### 响应特征

```json
{"QueryID": "7852813130838806383", "ResultCode": 0, "Result": {"code": 403, "isCaptchaEnabled": true, "msg": "hit risk"}}
```

- 外层 `ResultCode=0`，业务层 `Result.code=403`：只看外层或 HTTP 状态码会误判为成功。
- `isCaptchaEnabled=true` 表示上游要求验证码，普通 HTTP 请求无法通过。
- `selfselect/getstockquotation`（期货五日）风控时可能同时返回 `QueryID: "0"`。

#### 已尝试且无效的手段

- `User-Agent` + `Referer` 按浏览器请求补齐：仍 403。
- 先访问 `finance.baidu.com` 拿 `BAIDUID_BFESS` 等 cookie 再请求：仍被拒（403 或 TLS 重置）。
- 换不同证券 / 货币对 / 合约代码：仍 403，与标的无关。
- 部分请求直接 TLS 重置（`curl: (35) ... unexpected eof while reading`），与 403 交替出现。

#### 应对

1. 降频：请求之间留间隔，避免短时间连续试错，持续 403 后先停一段时间。
2. 复用 HTTP 连接（keep-alive），减少并发。
3. 持续 403 时更换出口网络 / IP（代理或换网络）。
4. 严格判断业务层 `Result.code`，不要只看 `ResultCode` 或 HTTP 200。

#### 失败处理与文档约定

- 风控应视为失败：不要返回半成品，也不要把风控响应当成空数据继续处理。
- 需要「空页继续」语义时，也必须先校验业务层字段存在且类型正确。
- 因风控未取得真实响应的接口（如外汇、期货），文档的响应结构一律标注「未验证」，
  不照搬其它 `group` 的字段表。

## 示例数据

各接口文档中的示例均标注来源，数值为当时快照，不代表实时行情。百度未提供官方样例文件，
示例一律内联在接口文档里，不单独存放 `json`。
