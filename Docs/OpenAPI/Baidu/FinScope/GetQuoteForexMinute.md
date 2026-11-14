# GetQuoteForexMinute

获取外汇 / 数字货币对（`USDCNY`、`BTCUSD` 等）的分钟行情。与股票 K 线同为
`/vapi/v1/getquotation`，靠 `group=huilv_minute` 区分；请求**不带** `market_type`，
货币对直接写在 `code` 里。
机器可读规格见
[Meta/OpenAPI/Baidu/FinScope/GetQuoteForexMinute.yaml](../../../../Meta/OpenAPI/Baidu/FinScope/GetQuoteForexMinute.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://finance.pae.baidu.com` |
| 路径 | `/vapi/v1/getquotation` |
| operationId | `getQuoteForexMinute`（`group=huilv_minute`） |
| 备注 | 外汇没有 `market` / `market_type` 概念；URL 中也未出现 `srcid`、`pointType`、`newFormat`。 |

## 请求参数

### 查询参数

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `group` | 是 | string | 接口分组，外汇分钟固定 `huilv_minute` | `huilv_minute` |
| `need_reverse_real` | 是 | integer | 固定 `1`。字面为「需要反向真实值」，具体口径未验证，按原样传递 | `1` |
| `code` | 是 | string | 货币对，6 位字母，`<基础货币><报价货币>` | `USDCNY` |
| `finClientType` | 是 | string | 客户端类型，固定 `pc` | `pc` |

与股票接口的关键差异：

| 差异 | 股票 K 线 | 外汇分钟 |
| --- | --- | --- |
| 分组 | `quotation_kline_ab` | `huilv_minute` |
| 市场参数 | `market_type=ab` | 无 |
| 代码 | 6 位数字证券代码 | 6 位字母货币对 |
| 其它固定参数 | `srcid`、`pointType`、`newFormat` | 仅 `need_reverse_real=1` |

### 货币对示例

| `code` | 含义 |
| --- | --- |
| `USDCNY` | 美元兑人民币 |
| `BTCUSD` | 比特币兑美元 |

代码大小写敏感，按 `USDCNY` 形式全大写传入。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `User-Agent` | 是 | 浏览器 UA，建议 Chrome 131 桌面版 |
| `Referer` | 是 | `https://gushitong.baidu.com/` |

### 请求体

`GET` 请求，无请求体。

### 示例

```bash
# 美元兑人民币
curl -sS \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36' \
  -H 'Referer: https://gushitong.baidu.com/' \
  'https://finance.pae.baidu.com/vapi/v1/getquotation?group=huilv_minute&need_reverse_real=1&code=USDCNY&finClientType=pc'

# 比特币兑美元
curl -sS \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36' \
  -H 'Referer: https://gushitong.baidu.com/' \
  'https://finance.pae.baidu.com/vapi/v1/getquotation?group=huilv_minute&need_reverse_real=1&code=BTCUSD&finClientType=pc'
```

## 响应

### 外层结构

与其它百度接口同一层信封，见 [README.md](README.md)「外层信封」：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `QueryID` | string | 请求追踪 ID |
| `ResultCode` | integer | 网关层结果，`0` 表示请求被正常处理（**不代表业务成功**） |
| `Result` | object | 业务数据；风控时为错误对象 |

### 业务数据结构

**未验证**。本机请求该接口返回风控结构：

```json
{"QueryID": "7852813130838806383", "ResultCode": 0, "Result": {"code": 403, "isCaptchaEnabled": true, "msg": "hit risk"}}
```

外汇分钟的 `Result` 内部结构尚未取得样本，**不能照搬股票 K 线的 `newMarketData`**
（`headers` / `keys` / `marketData`）字段；拿到真实响应后应补本节，并注明字段顺序来源。

## 业务码与异常

| 情形 | 表现 | 处理建议 |
| --- | --- | --- |
| 风控 | 外层 `ResultCode=0`，`Result` 为 `{"code":403,"isCaptchaEnabled":true,"msg":"hit risk"}` | 降频或更换出口网络，不要按成功解析 |
| 正常 | 待实测 | 拿到样本后按 `Result` 实际字段解析 |
| 货币对非法 | 待实测 | 核对 `code` 是否为 6 位大写字母组合 |

