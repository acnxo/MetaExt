# GetQuoteMinute

获取指定标的的分钟级分时数据（价格、成交量、成交额与涨跌），并给出当日交易时段列表。
机器可读规格见
[Meta/OpenAPI/MooMoo/QuoteAPIV2/GetQuoteMinute.yaml](../../../../Meta/OpenAPI/MooMoo/QuoteAPIV2/GetQuoteMinute.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://www.moomoo.com` |
| 路径 | `/quote-api/quote-v2/get-quote-minute` |
| operationId | `getQuoteMinute` |
| 代码入口 | `Client.GetQuoteMinute`（`component/quote/moomoo/quoteapiv2` 包） |
| 备注 | 六个定位参数全必填且需与 `stockId` 匹配；`req_section` 为唯一 snake_case 参数，省略即返回全天数据。 |

## 请求参数

### 查询参数

参数书写顺序就是签名输入的键序，不能重排。`req_section` 的位置固定在
`subInstrumentType` 之后、时间戳之前。

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `stockId` | 是 | string | 服务端内部股票 ID | `70000294` |
| `marketType` | 是 | string | 市场类型；本接口 `1` 为 A 股、`2` 为港股 | `2` |
| `type` | 是 | string | 行情类型，原样透传 | `2` |
| `marketCode` | 是 | string | 市场代码，原样透传 | `70` |
| `instrumentType` | 是 | string | 证券类型，原样透传 | `9` |
| `subInstrumentType` | 是 | string | 证券子类型，原样透传 | `9008` |
| `req_section` | 否 | string | 请求时段，观测到 `1` 为盘中；省略时整键省略，返回全天数据 | `1` |
| `_` | 否 | string | 毫秒级时间戳，缺省时整键省略 | `1780507278122` |

注意 `marketType` 的编码与其他接口不同：本接口中 `1` 为 A 股、`2` 为港股，而新闻列表接口的
`market_type` 是另一套含义，跨接口套用会取到错误市场。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `quote-token` | 是 | 请求签名，10 个字符，生成方式见 [Signature.md](Signature.md) |
| `futu-x-csrf-token` | 是 | CSRF 标记，该接口不校验取值但必须存在，且与 Cookie 中的 `csrfToken` 一致 |
| `cookie` | 是 | 需包含 `csrfToken` 与 `locale` 两项 |
| `accept` | 否 | 固定 `application/json` |
| `referer` | 否 | 来源页；默认 `https://www.moomoo.com/hans/`；该值决定返回数据的语言与数值单位口径 |
| `user-agent` | 否 | 浏览器 UA，非浏览器 UA 可能被拒绝 |

### 请求体

`GET` 请求，无请求体，不发送 `content-type`。

## 签名规则

完整算法、校验向量与排查步骤见 [Signature.md](Signature.md)。

不带 `req_section` 时：

```json
{"stockId":"70000294","marketType":"2","type":"2","marketCode":"70","instrumentType":"9","subInstrumentType":"9008","_":"1780507278122"}
```

对应 `quote-token = 45671b44c1`。这组参数与 K 线接口完全相同，因此签名向量也相同，可作为
参数集与键序一致的交叉验证。

带 `req_section` 时（插在 `subInstrumentType` 之后）：

```json
{"stockId":"70000294","marketType":"2","type":"2","marketCode":"70","instrumentType":"9","subInstrumentType":"9008","req_section":"1","_":"1780507278122"}
```

对应 `quote-token = 53de4453c4`。

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
| `data.stockId` | integer | 证券 ID |
| `data.server_time` | integer | 服务端时间戳，毫秒 |
| `data.last_close_price` | number | 昨收价 |
| `data.highest_price` | number | 最高价 |
| `data.lowest_price` | number | 最低价 |
| `data.list` | array | 分钟级分时数据 |
| `data.time_section` | array | 当日交易时段 |

### 分时字段

| 字段 | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- |
| `time` | 不出现 | `int64`，如 `1780480800` | 时间戳，秒级 |
| `price` | 不出现 | `float64`，如 `45.6` | 成交价格 |
| `cc_price` | 不出现 | `float64`，如 `45.6` | 复权价格 |
| `volume` | 不出现 | `float64`，如 `12500` | 成交量 |
| `turnover` | 不出现 | `float64`，如 `570000.0` | 成交额 |
| `ratio` | `"0.22%"` | 不出现 | 涨跌幅，含百分号，按原文保留 |
| `change_price` | 不出现 | `float64`，如 `0.1` | 涨跌额 |

接口以 JSON number 返回价格、量与额，代码用 `float64` 承接；涨跌幅是带百分号的字符串，
不转数值，避免丢失格式。

### 交易时段字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `begin` | integer | 时段开始时间戳，秒级 |
| `end` | integer | 时段结束时间戳，秒级 |

### 示例

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "stockId": 70000294,
    "server_time": 1780507278500,
    "last_close_price": 45.5,
    "highest_price": 46.8,
    "lowest_price": 45.2,
    "list": [
      {
        "time": 1780480800,
        "price": 45.6,
        "cc_price": 45.6,
        "volume": 12500,
        "turnover": 570000.0,
        "ratio": "0.22%",
        "change_price": 0.1
      }
    ],
    "time_section": [
      {"begin": 1780480800, "end": 1780491600}
    ]
  }
}
```

## 业务码

| code | 含义 | 处理建议 |
| --- | --- | --- |
| `0` | 成功 | 读取 `data.list` 与 `data.time_section` |
| `-403` | 签名被拒 | 检查签名输入的键序与值形式，注意 `req_section` 的位置 |
| `500` | 参数错误 | 常见原因是六个定位参数与 `stockId` 不匹配 |
| 其它非 0 | 业务失败 | 按 `message` 排查，不要按成功处理 |

## 相关实现

- `quoteminute.go`：六个必填参数校验、`req_section` 位置、响应解析
- `client.go`：配置、错误类型与请求发送
- `signature.go`：签名输入拼装与 `quote-token` 计算
- `quoteminute_test.go`：键序、`req_section` 插入位置、必填校验与错误分类
