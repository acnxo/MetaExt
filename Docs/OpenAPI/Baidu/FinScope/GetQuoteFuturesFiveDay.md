# GetQuoteFuturesFiveDay

获取期货合约近五个交易日的分钟分时。接口路径与股票不同：走
`/selfselect/getstockquotation`（不是 `/vapi/v1/getquotation`），靠
`group=quotation_futures_fiveday` + `isFutures=true` 区分。
机器可读规格见
[Meta/OpenAPI/Baidu/FinScope/GetQuoteFuturesFiveDay.yaml](../../../../Meta/OpenAPI/Baidu/FinScope/GetQuoteFuturesFiveDay.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://finance.pae.baidu.com` |
| 路径 | `/selfselect/getstockquotation` |
| operationId | `getQuoteFuturesFiveDay`（`group=quotation_futures_fiveday`） |
| 备注 | 用 `isFutures=true` 等 5 个品种开关代替股票侧的 `financeType`；市场仍由 `market_type` 指定。 |

## 请求参数

### 查询参数

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `all` | 是 | integer | 返回全部字段，固定 `1` | `1` |
| `code` | 是 | string | 期货合约代码 | `AU888` |
| `isIndex` | 是 | boolean | 指数开关，期货固定 `false` | `false` |
| `isBk` | 是 | boolean | 板块开关，期货固定 `false` | `false` |
| `isBlock` | 是 | boolean | Block 开关，期货固定 `false` | `false` |
| `isFutures` | 是 | boolean | **期货开关，固定 `true`** | `true` |
| `isStock` | 是 | boolean | 股票开关，期货固定 `false` | `false` |
| `newFormat` | 是 | integer | 新返回格式，固定 `1` | `1` |
| `market_type` | 是 | string | 市场：`ab` 国内商品期货、`us` 美国市场期货 | `ab` |
| `group` | 是 | string | 接口分组，固定 `quotation_futures_fiveday` | `quotation_futures_fiveday` |
| `finClientType` | 是 | string | 客户端类型，固定 `pc` | `pc` |

5 个 `is*` 是品种类型开关，**必须且只能 `isFutures=true`**，其余四个为 `false`；
与股票五日的差异还在于没有 `srcid`、`pointType`、`query`、`name`、`is_kc`。

### 合约代码与市场

| `code` | `market_type` | 含义 |
| --- | --- | --- |
| `AU888` | `ab` | 沪金主力连续合约 |
| `AU2612` | `ab` | 沪金 2026 年 12 月合约 |
| `GC26Z` | `us` | COMEX 黄金 2026 年 12 月合约（`Z` 为 12 月月份码） |

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
  'https://finance.pae.baidu.com/selfselect/getstockquotation?all=1&code=AU888&isIndex=false&isBk=false&isBlock=false&isFutures=true&isStock=false&newFormat=1&market_type=ab&group=quotation_futures_fiveday&finClientType=pc'

# 外盘期货：COMEX 黄金 2026-12
curl -sS \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36' \
  -H 'Referer: https://gushitong.baidu.com/' \
  'https://finance.pae.baidu.com/selfselect/getstockquotation?all=1&code=GC26Z&isIndex=false&isBk=false&isBlock=false&isFutures=true&isStock=false&newFormat=1&market_type=us&group=quotation_futures_fiveday&finClientType=pc'
```

## 响应

### 外层结构

与其它百度接口同一层信封（`QueryID` / `ResultCode` / `Result`），见
[README.md](README.md)「外层信封」。注意本接口路径是 `selfselect/...`，风控时可能直接
返回 `{"code":403,...}` 或 `QueryID` 为 `"0"`。

### 业务数据结构

**未验证**。本机请求返回风控：

```json
{"QueryID": "0", "ResultCode": 0, "Result": {"code": 403, "isCaptchaEnabled": true, "msg": "hit risk"}}
```

期货五日的 `Result` 内部结构尚未取得样本。它与股票五日（`quotation_fiveday_ab`）是否同为
`newMarketData.headers/keys/marketData` 结构、是否包含夜盘时段，都需拿到真实响应后确认，
不要直接套用 [GetQuoteFiveDay.md](GetQuoteFiveDay.md) 的字段表。

## 业务码与异常

| 情形 | 表现 | 处理建议 |
| --- | --- | --- |
| 风控 | `Result` 为 `{"code":403,...}`，或 `QueryID` 为 `"0"` | 降频或更换出口网络 |
| 正常 | 待实测 | 按 `Result` 实际字段解析 |
| 合约非法 / 已交割 | 待实测 | 核对 `code` 与 `market_type` 的搭配 |

