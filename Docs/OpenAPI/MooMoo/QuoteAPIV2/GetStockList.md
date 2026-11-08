# GetStockList

按市场与板块获取股票列表，支持排行榜排序与分页，返回行情快照与多周期涨跌幅。
机器可读规格见
[Meta/OpenAPI/MooMoo/QuoteAPIV2/GetStockList.yaml](../../../../Meta/OpenAPI/MooMoo/QuoteAPIV2/GetStockList.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://www.moomoo.com` |
| 路径 | `/quote-api/quote-v2/get-stock-list` |
| operationId | `getStockList` |
| 代码入口 | `Client.GetStockList`（`component/quote/moomoo/quoteapiv2` 包） |
| 备注 | 列表顺序由服务端决定；**页码从 0 开始**，`page=0` 才是第一页；需要其它顺序时由调用方排序。 |

## 请求参数

### 查询参数

参数书写顺序就是签名输入的键序，不能重排。**可选参数缺省时整键省略**，不要传空串。

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `marketType` | 是 | string | 市场类型，原样透传 | `1` |
| `plateType` | 否 | string | 板块类型；缺省时按默认值 `1` 补上 | `1` |
| `rankType` | 否 | string | 排行类型；缺省时省略该键，由服务端决定默认排序 | `3` |
| `page` | 否 | string | 页码，**从 `0` 开始**（`0` 即第一页）；缺省时省略，按第 `0` 页处理；`-1` 等负数在本地被拒绝 | `0` |
| `pageSize` | 否 | string | 每页数量；缺省时省略，服务端默认观测为 `50` | `50` |
| `_` | 否 | string | 毫秒级时间戳；缺省时省略该键，不自动补当前时间 | `1781611867445` |

`plateType` 与其它可选项不同：服务端对缺少该键的请求返回 `code=500`（`Params Error`），
因此实现中缺省时会显式补默认值 `1`，而不是省略该键。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `quote-token` | 是 | 请求签名，10 个字符，生成方式见 [Signature.md](Signature.md) |
| `futu-x-csrf-token` | 是 | CSRF 标记，该接口不校验取值但必须存在，且与 Cookie 中的 `csrfToken` 一致 |
| `cookie` | 是 | 需包含 `csrfToken` 与 `locale` 两项，例如 `csrfToken=0000000000; locale=zh-cn` |
| `referer` | 否 | 来源页。**决定返回数据的语言与数值单位口径**：简体中文站（`/hans/`）返回中文名与 `万亿/亿/万`，站点根路径返回英文名与 `T/B/M/K`。默认取对应市场的简体中文列表页（见下） |
| `user-agent` | 否 | 浏览器 UA，非浏览器 UA 可能被拒绝 |
| `accept` | 否 | 固定 `application/json` |

### 请求体

`GET` 请求，无请求体，不发送 `content-type`。

### 返回语言与自定义请求头

服务端**只根据 `referer` 决定返回语言与数值单位**；`cookie` 里的 `locale` 与
`accept-language` 实测都无效。因此本包默认把 `referer` 指向简体中文站的业务页面：

| marketType | 默认 `referer` |
| --- | --- |
| `1`（港股） | `https://www.moomoo.com/hans/quote/hk/stock-list/all-hk-stocks/top-market-cap` |
| `2`（美股） | `https://www.moomoo.com/hans/quote/us/stock-list/all-us-stocks/top-market-cap` |
| `4`（沪深） | `https://www.moomoo.com/hans/quote/cn/stock-list/all-cn-stocks/top-market-cap` |
| 其它 / 未命中 | `https://www.moomoo.com/hans/`（`DefaultReferer`） |

覆盖顺序（越靠后优先级越高）：`DefaultReferer` → `DefaultReferers[marketType]` →
`Config.Referer` → `Config.Referers[marketType]` → `Config.Header["referer"]`。需要补其它请求头（如 `accept-language`）时用 `Config.Header`：

```go
client := quoteapiv2.New(quoteapiv2.Config{
	Header: map[string]string{"accept-language": "zh-CN,zh;q=0.9,en;q=0.8"},
})
// 只想改某些市场的来源页：
client = quoteapiv2.New(quoteapiv2.Config{
	Referers: map[string]string{"2": "https://www.moomoo.com/hans/quote/us/stock-list/all-us-stocks/top-market-cap"},
})
```

`Config.Header` 在内置默认头（含 `referer`）之后逐项 `Set`，同名即覆盖；未设置时请求头
与之前版本逐项一致。

## 签名规则

完整算法、校验向量与排查步骤见 [Signature.md](Signature.md)。

本接口的签名输入为查询参数的紧凑 JSON（键序即参数顺序，值全部字符串化）。全参数示例：

```json
{"marketType":"1","plateType":"1","rankType":"3","page":"0","pageSize":"50","_":"1781611867445"}
```

对应 `quote-token = 34193eae90`。只传必填参数时为
`{"marketType":"1","plateType":"1"}`，对应 `quote-token = 0fdcd3ce32`。

> `page` 是该接口**唯一 0-based** 的分页语义差异点：传 `1` 会跳过第一页的 `pageSize` 条，
> 按市值倒序采集时会恒定缺失前 `pageSize` 名（港股即腾讯、汇丰、阿里等）。

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
| `data` | object | 业务数据，失败时可能为 `null` |
| `data.pagination` | object | 分页元信息，见下节 |
| `data.list` | array | 当前页条目，元素字段见下节 |

### 分页对象

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `page` | integer | 当前页码，从 `0` 开始 |
| `pageSize` | integer | 每页数量 |
| `pageCount` | integer | 总页数 |
| `total` | integer | 总条数，随行情变化 |

### 条目字段

行情类字段带单位（如「亿」「万」）或百分号，统一按字符串保留，不在解析阶段转数值。

| 字段 | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- |
| `stockId` | 不出现 | `int64`，如 `58531814313881` | 标的编号 |
| `name` | `"洛阳钼业"` | 不出现 | 名称 |
| `stockCode` | `"03993"` | 不出现 | 代码，可能以 0 开头，必须按字符串处理 |
| `marketLabel` | `"HK"` | 不出现 | 市场标识 |
| `instrumentType` | 不出现 | `int`，如 `3` | 标的类型编号，普通股通常为 3 |
| `changeRatio` | `"-4.30%"` | 不出现 | 涨跌幅，带百分号、保留两位小数 |
| `priceDirect` | `"down"` | 不出现 | 价格方向，`up` / `down` |
| `change` | `"-0.860"` | 不出现 | 涨跌额，保留三位小数 |
| `priceNominal` | `"19.120"` | 不出现 | 最新价，保留三位小数 |
| `tradeTrunover` | `"14.33亿"` | 不出现 | 成交额，带单位 |
| `tradeVolumn` | `"7408.21万"` | 不出现 | 成交量，带单位 |
| `marketVal` | `"4090.59亿"` | 不出现 | 总市值，带单位 |
| `circulationMarketValue` | `"752.08亿"` | 不出现 | 流通市值，带单位 |
| `totalShares` | `"213.94亿"` | 不出现 | 总股本，带单位 |
| `circulationTotalShares` | `"39.33亿"` | 不出现 | 流通股本，带单位 |
| `c_5Days` | `"+15.18%"` | 不出现 | 5 日涨跌幅，带符号与百分号 |
| `c_5Days_priceDirect` | `"up"` | 不出现 | 5 日价格方向 |
| `c_10Days` | `"-2.00%"` | 不出现 | 10 日涨跌幅 |
| `c_10Days_priceDirect` | `"down"` | 不出现 | 10 日价格方向 |
| `c_20Days` | `"+6.40%"` | 不出现 | 20 日涨跌幅 |
| `c_20Days_priceDirect` | `"up"` | 不出现 | 20 日价格方向 |
| `c_60Days` | `"+5.03%"` | 不出现 | 60 日涨跌幅 |
| `c_60Days_priceDirect` | `"up"` | 不出现 | 60 日价格方向 |
| `c_120Days` | `"+6.56%"` | 不出现 | 120 日涨跌幅 |
| `c_120Days_priceDirect` | `"up"` | 不出现 | 120 日价格方向 |
| `c_250Days` | `"+206.62%"` | 不出现 | 250 日涨跌幅 |
| `c_250Days_priceDirect` | `"up"` | 不出现 | 250 日价格方向 |
| `c_YearDays` | `"+1.09%"` | 不出现 | 一年涨跌幅 |
| `c_YearDays_priceDirect` | `"up"` | 不出现 | 一年价格方向 |
| `trustAum` | `"0.00"` | 不出现 | 信托资产规模，未适用时为 `"0.00"` |
| `leverage` | `"2.5"` | `float64`，如 `2.5` | 杠杆率，普通股为 `null` |
| `tradeChangeraio` | `"1.88%"` | 不出现 | 换手率，接口字段名原样为 `tradeChangeraio` |
| `priceAmplitude` | `"5.26%"` | 不出现 | 振幅，带百分号 |
| `volumnRatio` | `"1.09"` | 不出现 | 量比，接口字段名原样为 `volumnRatio` |
| `buysellRatio` | `"0.00%"` | 不出现 | 买卖盘比例，带百分号 |

三个字段名与常规拼写不一致，规格与代码均保持原样：`tradeTrunover`（成交额）、
`tradeVolumn`（成交量）、`tradeChangeraio`（换手率）。代码中 Go 字段用正确拼写，
JSON tag 用接口原名。

### 示例

成功的完整响应见规格文件的 `examples.success`，片段如下：

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "pagination": {"page": 0, "pageSize": 50, "pageCount": 65, "total": 3228},
    "list": [
      {
        "stockId": 58531814313881,
        "name": "洛阳钼业",
        "stockCode": "03993",
        "marketLabel": "HK",
        "instrumentType": 3,
        "changeRatio": "-4.30%",
        "priceDirect": "down",
        "change": "-0.860",
        "priceNominal": "19.120",
        "leverage": null
      }
    ]
  }
}
```

参数错误的响应同样是 HTTP 200：

```json
{"code": 500, "message": "Params Error", "data": null}
```

## 业务码

| code | 含义 | 处理建议 |
| --- | --- | --- |
| `0` | 成功 | 读取 `data.pagination` 与 `data.list` |
| `-403` | 签名被拒 | 检查签名输入的键序、取值与字符串化方式 |
| `500` | 参数错误 | 检查 `marketType`、`plateType` 等必填项；消息通常为 `Params Error` |
| 其它非 0 | 业务失败 | 按 `message` 排查，不要按成功处理 |

## 相关实现

- `stocklist.go`：请求参数校验、参数构造与响应解析
- `client.go`：配置、错误类型与请求发送
- `signature.go`：签名输入拼装与 `quote-token` 计算
- `stocklist_test.go`：请求装配、键序、签名向量、响应解析与错误分类
