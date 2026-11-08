# GetFuturesList

按市场与分类获取期货合约列表，支持分页。响应为**位置化字段结构**：字段定义与展示值分列两处。
机器可读规格见
[Meta/OpenAPI/MooMoo/QuoteAPIV2/GetFuturesList.yaml](../../../../Meta/OpenAPI/MooMoo/QuoteAPIV2/GetFuturesList.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://www.moomoo.com` |
| 路径 | `/quote-api/quote-v2/get-futures-list` |
| operationId | `getFuturesList` |
| 代码入口 | `Client.GetFuturesList`（`component/quote/moomoo/quoteapiv2` 包） |
| 备注 | 页码从 0 开始；`marketType` 与 `futuresType` 必填且需成对使用；条目为位置化字段结构。 |

## 请求参数

### 查询参数

参数书写顺序就是签名输入的键序，不能重排。

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `marketType` | 是 | string | 市场类型，原样透传 | `2` |
| `futuresType` | 是 | string | 期货分类，需与 `marketType` 成对使用 | `4` |
| `page` | 否 | string | 页码，从 `0` 开始，缺省补 `0` | `0` |
| `pageSize` | 否 | string | 每页数量，缺省补 `40` | `40` |
| `_` | 否 | string | 毫秒级时间戳，缺省时整键省略 | `1789944039632` |

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `quote-token` | 是 | 请求签名，10 个字符，生成方式见 [Signature.md](Signature.md) |
| `futu-x-csrf-token` | 是 | 该接口观测到不校验其取值，但按既有约定仍需携带 |
| `cookie` | 是 | 需包含 `csrfToken` 与 `locale` 两项 |
| `accept` | 否 | 固定 `application/json` |
| `referer` | 否 | 来源页；默认按 `marketType` 取对应市场的简体中文列表页，未命中回落 `https://www.moomoo.com/hans/`；该值决定返回数据的语言与数值单位口径 |
| `user-agent` | 否 | 浏览器 UA，非浏览器 UA 可能被拒绝 |

### 请求体

`GET` 请求，无请求体，不发送 `content-type`。

## 签名规则

完整算法、校验向量与排查步骤见 [Signature.md](Signature.md)。

```json
{"marketType":"2","futuresType":"4","page":"0","pageSize":"40","_":"1789944039632"}
```

对应 `quote-token = d928a3b3e1`。不带时间戳时对应 `quote-token = f9c3474442`。

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
| `data.pagination` | object | 分页元信息，`page` 从 0 开始 |
| `data.list` | array | 期货条目 |

分页对象字段与股票列表一致：`page`（从 0 起）、`pageCount`、`pageSize`、`total`。

### 条目结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `stockId` | integer | 服务端内部证券唯一标识，`int64` |
| `priceDirect` | string | 价格方向，`up` / `down` |
| `fieldList` | array | 字段定义数组，含原始值与类型 |
| `fields` | object | 字段展示值，键为字段编号的字符串形式 |

两者的关系：

- `fields` 是**按编号索引的展示值**，已换算成可直接展示的文本（如 `"3": "4413.8"`）；
- `fieldList` 是**按位置排列的字段定义**，给出原始值 `value`、文本值 `text`、
  小数位 `precision` 与内容类型 `contentType`；
- `fields` 比 `fieldList` 多一个键 `19`（市场标签），`fieldList` 中没有对应项，
  无法通过位置枚举到它。

一般直接使用 `fields`；需要自己换算数值时才看 `fieldList`。

### fieldList 元素字段

| 字段 | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- |
| `fieldId` | 不出现 | `int`，如 `3` | 字段位置，从 1 开始 |
| `fieldName` | `"价格"` | 不出现 | 字段名，语种可能中英切换，不要按名称硬编码逻辑 |
| `value` | `"4413800000000"` | 不出现 | 数值类字段的原始值（放大后的整数），仅数值类字段返回 |
| `text` | `"GCmain"` | 不出现 | 文本类字段的值，仅文本类字段返回 |
| `precision` | 不出现 | `int`，如 `1` | 小数位：价格类为放大偏移，数量类为展示小数位 |
| `contentType` | 不出现 | `int`，如 `2` | 内容类型，决定数值还原方式 |

### 字段位置对照

`fieldList` 含 18 项，位置固定：

| fieldId | 含义 | fieldId | 含义 |
| --- | --- | --- | --- |
| `1` | 合约代码 | `10` | 成交量 |
| `2` | 名称 | `11` | 持仓量 |
| `3` | 价格 | `12` | 日增仓 |
| `4` | 涨跌幅 | `13` | 开盘 |
| `5` | 涨跌额 | `14` | 最高 |
| `6` | 买入价 | `15` | 最低 |
| `7` | 卖出价 | `16` | 最后交易 |
| `8` | 买量 | `17` | 昨结 |
| `9` | 卖量 | `18` | 昨收 |

`fields` 额外包含 `19`：市场标签（`US` / `HK`）。

### 示例

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "pagination": {"page": 0, "pageCount": 1, "pageSize": 40, "total": 14},
    "list": [
      {
        "stockId": 70000294,
        "priceDirect": "down",
        "fieldList": [
          {"fieldId": 1, "fieldName": "合约代码", "text": "GCmain", "contentType": 4},
          {"fieldId": 3, "fieldName": "价格", "value": "4413800000000", "precision": 1, "contentType": 2},
          {"fieldId": 4, "fieldName": "涨跌幅", "value": "-2508531", "precision": 2, "contentType": 1}
        ],
        "fields": {
          "1": "GCmain", "2": "黄金", "3": "4413.8", "4": "-0.25%",
          "11": "315327.0", "18": "4424.9", "19": "US"
        }
      }
    ]
  }
}
```

### 关于数值还原

原始 `value` 需要按 `contentType` 与 `precision` 还原成展示值，公式属于站点字段字典的内容，
本项目未收录，因此代码里**不做换算**，只提供 `Fields` 中的展示值与 `FieldList` 中的原始
字段。这样不会因猜错口径而给出错误数字。

## 业务码

| code | 含义 | 处理建议 |
| --- | --- | --- |
| `0` | 成功 | 读取 `data.pagination` 与 `data.list` |
| `-403` | 签名被拒 | 检查签名输入的键序与值形式 |
| `500` | 参数错误 | 常见原因是 `marketType` 与 `futuresType` 组合不被接受 |
| 其它非 0 | 业务失败 | 按 `message` 排查，不要按成功处理 |

## 市场与分类组合

| `marketType` | `futuresType` | 含义 | 已观测 `total` |
| --- | --- | --- | --- |
| `2` | `4` | 美股商品期货 | 14 |
| `2` | `1` | 美股股指期货 | 14 |
| `1` | `8` | 港股股指期货 | 102 |

## 相关实现

- `futureslist.go`：必填校验、分页默认值、位置化字段解析
- `client.go`：配置、错误类型与请求发送
- `signature.go`：签名输入拼装与 `quote-token` 计算
- `futureslist_test.go`：键序、默认参数向量、位置化字段解析与错误分类
