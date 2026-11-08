# GetHistoryCashTrend

获取个股历史资金流水：各单型净流入、主力净流入与占比，支持日 / 周 / 月三档粒度，
以响应记录的 `time` 作为游标向前翻页。
机器可读规格见
[Meta/OpenAPI/MooMoo/QuoteAPIV2/GetHistoryCashTrend.yaml](../../../../Meta/OpenAPI/MooMoo/QuoteAPIV2/GetHistoryCashTrend.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://www.moomoo.com` |
| 路径 | `/quote-api/quote-v2/get-history-cash-trend` |
| operationId | `getHistoryCashTrend` |
| 代码入口 | `Client.GetHistoryCashTrend`（`component/quote/moomoo/quoteapiv2` 包） |
| 备注 | `type` 省略会被服务端静默忽略并返回空列表；比率字段为百分数 ×100；空列表不等于参数错误。 |

## 请求参数

### 查询参数

参数书写顺序就是签名输入的键序，不能重排。

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `stockId` | 是 | string | 服务端内部股票 ID | `203290` |
| `type` | 是 | string | 资金流粒度：`1` 日、`2` 周、`3` 月 | `1` |
| `baseTime` | 是 | string | 分页游标（秒）；`0` 表示最新一段，为空时实现补 `0` | `0` |
| `_` | 否 | string | 毫秒级时间戳，缺省时整键省略 | `1789943637356` |

`type` 是**非校验型参数**：服务端对省略或未知取值既不报错也不返回数据，只给空列表。
`0`、`4`、`5`、`6` 与完全省略的表现一致。因此实现会在本地拒绝空的 `type`，
避免调用方把「忘传参数」误判为「没有数据」。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `quote-token` | 是 | 请求签名，10 个字符，生成方式见 [Signature.md](Signature.md) |
| `futu-x-csrf-token` | 是 | 该接口观测到不校验其取值，但按既有约定仍需携带 |
| `cookie` | 是 | 需包含 `csrfToken` 与 `locale` 两项 |
| `accept` | 否 | 固定 `application/json` |
| `referer` | 否 | 来源页；默认 `https://www.moomoo.com/hans/`；该值决定返回数据的语言与数值单位口径 |
| `user-agent` | 否 | 浏览器 UA，非浏览器 UA 可能被拒绝 |

### 请求体

`GET` 请求，无请求体，不发送 `content-type`。

## 签名规则

完整算法、校验向量与排查步骤见 [Signature.md](Signature.md)。

```json
{"stockId":"203290","type":"1","baseTime":"0","_":"1789943637356"}
```

对应 `quote-token = 44346db4fa`。不带时间戳的默认组合对应 `quote-token = cca66938e3`。

## 响应

### 响应头

| 响应头 | 说明 |
| --- | --- |
| `content-type` | `application/json` |

响应头不属于接口契约，业务成败一律以响应体的 `code` 为准。

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `code` | integer | 业务码，`0` 表示成功 |
| `message` | string | 业务提示，成功时通常为「成功」 |
| `data.hasMore` | boolean | 是否还有更早一段数据 |
| `data.list` | array | 资金流水记录，可能为空数组 |

### 记录字段

全部为字符串，包括时间、金额与比率。

| 字段 | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- |
| `time` | `"1786075200"` | 不出现 | 秒级时间戳字符串，翻页时作为下次 `baseTime` |
| `totalNetIn` | `"-700324120157"` | 不出现 | 总净流入，最小货币单位的整数字符串 |
| `superNetIn` | `"9752295125"` | 不出现 | 特大单净流入 |
| `bigNetIn` | `"-293476434557"` | 不出现 | 大单净流入 |
| `midNetIn` | `"-248713964950"` | 不出现 | 中单净流入 |
| `smlNetIn` | `"-167886015775"` | 不出现 | 小单净流入，接口字段名为 `smlNetIn` |
| `mainNetIn` | `"-283724139432"` | 不出现 | 主力净流入，即特大单加大单 |
| `priceChangeRatio` | `"117"` | 不出现 | 涨跌幅，百分数 ×100，`117` 表示 +1.17% |
| `closePrice` | `"723.03"` | 不出现 | 该期收盘价，保留两位小数 |
| `mainDealRatio` | `"2897"` | 不出现 | 主力成交占比，百分数 ×100，`2897` 表示 28.97% |
| `accMainNetIn` | `"-283724139432"` | 不出现 | 累计主力净流入 |

**换算口径**：比率字段除以 `100` 得到百分数；金额字段是最小货币单位的整数，不含小数位信息。
这套口径与期货列表接口的数值还原方式不同，不能混用。

### 示例

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "hasMore": true,
    "list": [
      {
        "time": "1786075200",
        "totalNetIn": "-700324120157",
        "superNetIn": "9752295125",
        "bigNetIn": "-293476434557",
        "midNetIn": "-248713964950",
        "smlNetIn": "-167886015775",
        "mainNetIn": "-283724139432",
        "priceChangeRatio": "117",
        "closePrice": "723.03",
        "mainDealRatio": "2897",
        "accMainNetIn": "-283724139432"
      }
    ]
  }
}
```

## 空列表的含义

空列表是正常响应，`code` 仍为 `0`，不能据此判断参数有误。以下情形返回同一形态：

| 情形 | 响应 |
| --- | --- |
| 省略 `type` | `code=0`，`list=[]` |
| `type` 取无数据的值（`0`、`4`、`5`、`6`） | `code=0`，`list=[]` |
| `stockId` 无效 | `code=0`，`list=[]` |
| 标的在当前时段没有资金流数据 | `code=0`，`list=[]` |

要确认某个标的究竟有没有资金流，需要先用 `type=1` 取一次非空列表作对照。

## 翻页与最新数据

| 目标 | 做法 |
| --- | --- |
| 最新一段 | `type=1` 且 `baseTime=0` |
| 当日累计 | 取返回记录中 `time` 最大的一条 |
| 更早的历史 | 把已取到记录的 `time` 作为下次的 `baseTime` 继续翻页 |

`hasMore` 为 `true` 表示还能继续往前取。列表顺序与服务端实现相关，翻页时建议按 `time`
比较取值，不要依赖元素位置。本接口没有独立的实时模式，也不提供分钟级刷新。

## 业务码

| code | 含义 | 处理建议 |
| --- | --- | --- |
| `0` | 成功 | 读取 `data.list`，按 `hasMore` 决定是否继续翻页 |
| `-403` | 签名被拒 | 检查签名输入的键序与值形式 |
| `500` | 参数错误 | 检查 `stockId`、`type`、`baseTime` 的取值 |
| 其它非 0 | 业务失败 | 按 `message` 排查，不要按成功处理 |

## 相关实现

- `cashtrend.go`：必填校验（含 `type` 静默失败的本地拦截）、游标参数与响应解析
- `client.go`：配置、错误类型与请求发送
- `signature.go`：签名输入拼装与 `quote-token` 计算
- `cashtrend_test.go`：键序、`type` 必填、默认游标、空列表语义与错误分类
