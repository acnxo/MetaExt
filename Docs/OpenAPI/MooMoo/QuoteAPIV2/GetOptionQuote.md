# GetOptionQuote

批量获取期权合约的实时报价，一次可传多个合约，响应以合约 ID 为键。
机器可读规格见
[Meta/OpenAPI/MooMoo/QuoteAPIV2/GetOptionQuote.yaml](../../../../Meta/OpenAPI/MooMoo/QuoteAPIV2/GetOptionQuote.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `POST` |
| 站点 | `https://www.moomoo.com` |
| 路径 | `/quote-api/quote-v2/get-option-quote` |
| operationId | `getOptionQuote` |
| 代码入口 | `Client.GetOptionQuote`（`component/quote/moomoo/quoteapiv2` 包） |
| 备注 | 无查询参数，签名输入是请求体原始字节；已过期或已摘牌的合约返回空对象而非错误。 |

## 请求参数

### 查询参数

无。该接口的 URL 不携带查询串。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `quote-token` | 是 | 请求签名，10 个字符；签名输入为请求体原始字节，见 [Signature.md](Signature.md) |
| `futu-x-csrf-token` | 是 | CSRF 标记，该接口不校验取值但必须存在，且与 Cookie 中的 `csrfToken` 一致 |
| `cookie` | 是 | 需包含 `csrfToken` 与 `locale` 两项，例如 `csrfToken=0000000000; locale=zh-cn` |
| `content-type` | 是 | 固定 `application/json`，POST 请求附加 |
| `origin` | 是 | 固定 `https://www.moomoo.com`，POST 请求附加 |
| `accept` | 否 | 固定 `application/json` |
| `referer` | 否 | 来源页，建议带上站点页面地址 |
| `user-agent` | 否 | 浏览器 UA，非浏览器 UA 可能被拒绝 |

### 请求体

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `ids` | array | 是 | 期权合约数组，至少 1 个元素 |
| `ids[].id` | string | 是 | 期权合约 ID，**字符串**形式 |
| `ids[].spreadTableCode` | int | 是 | 价差表代码，**整数**形式 |

请求体示例（紧凑格式，即实际发送与参与签名的那份字节）：

```json
{"ids":[{"id":"505895381","spreadTableCode":81},{"id":"505928056","spreadTableCode":81}]}
```

约束：

- 必须紧凑，无多余空格与换行；
- 每个元素的键序固定为 `id` 在前、`spreadTableCode` 在后；
- `id` 是字符串、`spreadTableCode` 是整数，类型不得改动：写成 `"81"` 会使签名输入
  与服务端不一致，返回 `-403`；
- 签名字节一经确定，不得再格式化或重新序列化。

## 签名规则

完整算法、校验向量与排查步骤见 [Signature.md](Signature.md)。

本接口与其他接口的区别在于**签名输入是请求体的原始字节**，不拼接查询串：

```text
input = '{"ids":[{"id":"505895381","spreadTableCode":81},{"id":"505928056","spreadTableCode":81}]}'
token = d28d1397f7
```

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
| `data` | object | 以合约 ID 为键的报价对象，可能是空对象，失败时可能为 `null` |

`data` 是**对象而非数组**：键是请求中的合约 ID，值是该合约的报价。调用方需要按请求的
ID 逐个查表，缺键表示该合约没有有效报价。

### 报价字段

| 字段 | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- |
| `priceBid` | `"8.600"` | 不出现 | 买盘价格，保留三位小数 |
| `volumeBid` | `"29"` | 不出现 | 买盘数量，整数形式的字符串 |
| `priceAsk` | `"9.900"` | 不出现 | 卖盘价格，保留三位小数 |
| `volumeAsk` | `"13"` | 不出现 | 卖盘数量，整数形式的字符串 |
| `price` | `"6.500"` | 不出现 | 最新成交价，保留三位小数 |
| `changeRatio` | `"-65.68%"` | 不出现 | 涨跌幅，带百分号、保留两位小数 |
| `change` | `"-12.440"` | 不出现 | 涨跌额，保留三位小数 |
| `volume` | `"15"` | 不出现 | 成交量 |
| `turnover` | `"9705.00"` | 不出现 | 成交额，保留两位小数 |
| `priceDirect` | `"down"` | 不出现 | 价格方向，`up` / `down` |

全部字段为字符串，包括价格与数量；需要数值时由调用方自行转换。代码中统一用 `string`
承载，避免在解析阶段引入浮点误差。

### 示例

成功：

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "505895381": {
      "priceBid": "8.600",
      "volumeBid": "29",
      "priceAsk": "9.900",
      "volumeAsk": "13",
      "price": "6.500",
      "changeRatio": "-65.68%",
      "change": "-12.440",
      "volume": "15",
      "turnover": "9705.00",
      "priceDirect": "down"
    }
  }
}
```

合约已过期或已摘牌（正常返回，不是错误）：

```json
{"code": 0, "message": "成功", "data": {}}
```

## 业务码

| code | 含义 | 处理建议 |
| --- | --- | --- |
| `0` | 成功 | 读取 `data`，按请求的 ID 逐个查表 |
| `-403` | 签名被拒 | 检查请求体是否为紧凑 JSON、键序与字段类型是否一致 |
| `500` | 参数错误 | 检查价差表代码与市场是否匹配 |
| 其它非 0 | 业务失败 | 按 `message` 排查，不要按成功处理 |

## 相关实现

- `optionquote.go`：请求体构造、参数校验与响应解析
- `client.go`：配置、错误类型，以及 GET/POST 共用的请求发送
- `signature.go`：签名输入拼装与 `quote-token` 计算
- `optionquote_test.go`：请求体字节、签名向量、空对象、错误分类
