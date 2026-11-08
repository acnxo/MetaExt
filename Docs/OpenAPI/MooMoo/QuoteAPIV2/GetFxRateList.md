# GetFxRateList

按清单分组获取外汇与贵金属汇率列表，返回最新价、买卖盘、开高低与涨跌，支持分页。
机器可读规格见
[Meta/OpenAPI/MooMoo/QuoteAPIV2/GetFxRateList.yaml](../../../../Meta/OpenAPI/MooMoo/QuoteAPIV2/GetFxRateList.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://www.moomoo.com` |
| 路径 | `/quote-api/quote-v2/get-fx-rate-list` |
| operationId | `getFxRateList` |
| 代码入口 | `Client.GetFxRateList`（`component/quote/moomoo/quoteapiv2` 包） |
| 备注 | **页码从 `0` 开始**，`page=0` 即第一页；同一 `listId` 下顺序稳定，需要其它顺序时由调用方排序。 |

## 请求参数

### 查询参数

参数书写顺序就是签名输入的键序，不能重排。本接口**四个参数都有服务端默认值**，
实现中缺省时显式补默认值并参与签名，而不是整键省略。

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `listId` | 否 | string | 清单分组 ID；缺省补 `1`（全部汇率）。分组取值见下节 | `1` |
| `page` | 否 | string | 页码，**从 `0` 开始**；缺省补 `0`。越界页返回空 `list` | `0` |
| `pageSize` | 否 | string | 每页数量；缺省补 `50`。观测 `300` 可用、`500` 返回 `code=500` | `50` |
| `_` | 否 | string | 毫秒级时间戳；缺省时省略该键，不自动补当前时间 | `1793831403000` |

### 清单分组（`listId`）

| `listId` | 前端路由 | 标签 | 观测 `total` |
| :---: | --- | --- | --- |
| `1` | `all-forex` | 全部汇率 | 282 |
| `10` | `Europe&Americas` | 欧美汇率 | 72 |
| `11` | `Asia-Pacific` | 亚太汇率 | 67 |
| `12` | `others` | 其他汇率 | 118 |
| `2` | — | 人民币汇率（全部为 `*CNY`） | 22 |
| `3` | — | 离岸人民币汇率（含 `CNH`） | 12 |
| `4` | — | 指数与美元直盘（含 `USDindex`） | 15 |
| `5` | — | 主要货币对 | 32 |
| `6` | — | 交叉汇率 | 50 |
| `7` | — | 人民币核心汇率 | 5 |
| `8` | — | 美元指数与主要直盘 | 5 |
| `9` | — | 主要货币对（第二组） | 36 |
| `0` | — | 前端「未选择」占位值 | **服务端返回 `code=3`** |

`1` / `10` / `11` / `12` 的标签取自前端产物的静态映射，可视为权威；`2`–`9` 只观测到
「传该值能返回数据」，标签按返回成分归纳，不排除存在其它用途或后续失效。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `quote-token` | 是 | 请求签名，10 个字符，生成方式见 [Signature.md](Signature.md) |
| `futu-x-csrf-token` | 是 | CSRF 标记，观测到不校验取值但需携带，且与 Cookie 中的 `csrfToken` 一致 |
| `cookie` | 是 | 需包含 `csrfToken` 与 `locale` 两项，例如 `csrfToken=0000000000; locale=zh-cn` |
| `referer` | 否 | 来源页；影响返回数据的语言与数值单位口径，默认取简体中文站 `https://www.moomoo.com/hans/` |
| `user-agent` | 否 | 浏览器 UA，非浏览器 UA 可能被拒绝 |
| `accept` | 否 | 固定 `application/json` |

### 请求体

`GET` 请求，无请求体，不发送 `content-type`。

## 签名规则

完整算法、校验向量与排查步骤见 [Signature.md](Signature.md)。签名输入为查询参数的紧凑
JSON（键序即参数顺序，值全部字符串化）。

```json
{"listId":"1","page":"0","pageSize":"50","_":"1793831403000"}
```

对应 `quote-token = 71a7397b8b`。另一组实测向量：

```json
{"listId":"11","page":"2","pageSize":"20","_":"1793859759000"}
```

对应 `quote-token = 116547b41a`。

## 响应

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `code` | integer | 业务码，`0` 表示成功 |
| `message` | string | 业务提示，成功时通常为「成功」 |
| `data.listId` | string | 服务端回显的分组 ID；**页码越界时该字段缺失** |
| `data.list` | array | 当前页条目，元素字段见下节 |
| `data.pagination` | object | 分页元信息，`page` 从 `0` 开始 |

### 分页对象

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `page` | integer | 当前页码，从 `0` 开始 |
| `pageSize` | integer | 每页数量 |
| `pageCount` | integer | 总页数 |
| `total` | integer | 总条数，随服务端货币对增删变化 |

### 条目字段

价格按各货币对精度格式化（`USDJPY` 3 位、`USDCNY` 4 位、`EURUSD` 5 位），涨跌幅与振幅
是服务端已格式化的百分比文本，**统一按字符串保留**，需要数值计算时自行转换。

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `stockId` | int64 | 证券内部 ID，外汇标的集中在 `72000000` 段 | `72000027` |
| `stockCode` | string | 货币对代码；指数类为 `USDindex` / `EURindex` | `EURUSD` |
| `marketLabel` | string | 市场标签，外汇固定 `FX` | `FX` |
| `instrumentType` | int | 证券类型，外汇固定 `10` | `10` |
| `name` | string | 展示名称 | `EUR/USD` |
| `priceDirect` | string | 最新价方向：`up` / `down` / `flat` | `flat` |
| `priceNominal` | string | 最新价 | `1.12286` |
| `changeAmount` | string | 涨跌额，带符号 | `-0.00001` |
| `changePercent` | string | 涨跌幅，含 `%` | `0.00%` |
| `priceBid` | string | 买盘价 | `1.12269` |
| `priceBidDirect` | string | 买盘价方向 | `down` |
| `priceAsk` | string | 卖盘价 | `1.12303` |
| `priceAskDirect` | string | 卖盘价方向 | `up` |
| `priceOpen` | string | 今开 | `1.12287` |
| `priceLastClose` | string | 昨收 | `1.12287` |
| `priceHighest` | string | 今日最高 | `1.12291` |
| `priceLowest` | string | 今日最低 | `1.12266` |
| `amplitudePrice` | string | 振幅，含 `%` | `0.02%` |

### 示例

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "listId": "1",
    "list": [
      {
        "stockId": 72000027,
        "stockCode": "EURUSD",
        "marketLabel": "FX",
        "instrumentType": 10,
        "name": "EUR/USD",
        "priceDirect": "flat",
        "priceNominal": "1.12286",
        "changeAmount": "-0.00001",
        "changePercent": "0.00%",
        "priceBid": "1.12269",
        "priceBidDirect": "down",
        "priceAsk": "1.12303",
        "priceAskDirect": "up",
        "priceOpen": "1.12287",
        "priceLastClose": "1.12287",
        "priceHighest": "1.12291",
        "priceLowest": "1.12266",
        "amplitudePrice": "0.02%"
      }
    ],
    "pagination": {"page": 0, "pageSize": 50, "pageCount": 6, "total": 282}
  }
}
```

页码越界（`page=99`，`pageCount=6`）时 `code` 仍为 `0`，`list` 为空且 `data.listId` 缺失：

```json
{"code":0,"message":"成功","data":{"list":[],"pagination":{"page":99,"pageSize":50,"pageCount":6,"total":282}}}
```

## 业务码

| code | 含义 | 处理建议 |
| --- | --- | --- |
| `0` | 成功 | 读取 `data.listId`、`data.pagination` 与 `data.list` |
| `-403` | 签名被拒 | 检查键序是否为 `listId` → `page` → `pageSize` → `_` |
| `3` | 清单分组不可用（如 `listId=0`） | 换用 `1` 或 `10`~`12`；消息为 `[srpc异常]` |
| `500` | 参数错误（如 `pageSize=500`） | 降低 `pageSize`，观测 `300` 可用 |
| 其它非 0 | 业务失败 | 按 `message` 排查，不要按成功处理 |

## 相关实现

- `fxratelist.go`：请求参数默认值与校验、参数构造、响应解析
- `client.go`：配置、错误类型与请求发送
- `signature.go`：签名输入拼装与 `quote-token` 计算
- `fxratelist_test.go`：请求装配、默认值、两组签名向量、越界页与错误分类
