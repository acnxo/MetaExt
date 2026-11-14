# GetQuoteFuturesMinute

获取期货合约的分钟行情。与股票 K 线同为 `/vapi/v1/getquotation`，靠
`group=quotation_futures_minute` + `financeType=futures` 区分；市场由 `market_type` 指定
（`ab` 国内商品期货、`us` 美国市场期货）。
机器可读规格见
[Meta/OpenAPI/Baidu/FinScope/GetQuoteFuturesMinute.yaml](../../../../Meta/OpenAPI/Baidu/FinScope/GetQuoteFuturesMinute.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://finance.pae.baidu.com` |
| 路径 | `/vapi/v1/getquotation` |
| operationId | `getQuoteFuturesMinute`（`group=quotation_futures_minute`） |
| 备注 | 与股票 K 线同路径不同 `group`；多了 `all`、`eprop`、`financeType` 三个期货专用参数。 |

## 请求参数

### 查询参数

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `srcid` | 是 | string | 来源标识，固定 `5353` | `5353` |
| `all` | 是 | integer | 返回全部字段，固定 `1` | `1` |
| `code` | 是 | string | 期货合约代码 | `AU888` |
| `query` | 是 | string | 搜索词，取与 `code` 相同的值 | `AU888` |
| `eprop` | 是 | string | 周期属性，分钟线固定 `min` | `min` |
| `financeType` | 是 | string | 品种类型，期货固定 `futures` | `futures` |
| `group` | 是 | string | 接口分组，固定 `quotation_futures_minute` | `quotation_futures_minute` |
| `market_type` | 是 | string | 市场：`ab` 国内商品期货、`us` 美国市场期货 | `ab` |
| `finClientType` | 是 | string | 客户端类型，固定 `pc` | `pc` |

与股票 K 线的差异：`financeType=futures`、`eprop=min` 为期货特有；股票侧用
`market_type=ab` 时对应的是 A 股，期货侧同一个 `ab` 表示国内商品期货。

### 合约代码格式

| `code` | `market_type` | 含义 |
| --- | --- | --- |
| `AU888` | `ab` | 沪金主力连续合约（`888` 为连续合约占位） |
| `AU2612` | `ab` | 沪金 2026 年 12 月合约 |
| `GC26Z` | `us` | COMEX 黄金 2026 年 12 月合约（`Z` 为 12 月月份码） |

国内品种用「品种字母 + 合约数字」（`888` 表示主力连续），外盘用「品种代码 + 两位年份 + 月份码」。
具体品种前缀与交易所归属未在本仓库逐一整理。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `User-Agent` | 是 | 浏览器 UA，统一 Chrome 131 桌面版 |
| `Referer` | 是 | `https://gushitong.baidu.com/` |

### 请求体

`GET` 请求，无请求体。

### 示例

```bash
# 国内商品期货：沪金主力连续
curl -sS \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36' \
  -H 'Referer: https://gushitong.baidu.com/' \
  'https://finance.pae.baidu.com/vapi/v1/getquotation?srcid=5353&all=1&code=AU888&query=AU888&eprop=min&financeType=futures&group=quotation_futures_minute&market_type=ab&finClientType=pc'
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

**未验证**。本机请求 `getquotation` 组持续返回风控：

```json
{"ResultCode": 0, "Result": {"code": 403, "isCaptchaEnabled": true, "msg": "hit risk"}}
```

期货分钟的 `Result` 内部结构尚未取得样本，**不要照搬股票 K 线的 `newMarketData` 字段表**；
拿到真实响应后补本节，并核对 `keys` 是否包含期货特有字段（持仓量、结算价等）。

## 业务码与异常

| 情形 | 表现 | 处理建议 |
| --- | --- | --- |
| 风控 | 外层 `ResultCode=0`，`Result` 为 `{"code":403,...}` | 降频或更换出口网络 |
| 正常 | 待实测 | 按 `Result` 实际字段解析 |
| 合约非法 / 已交割 | 待实测 | 核对 `code` 与 `market_type` 的搭配 |

