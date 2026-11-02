# GetEtfList

按板块获取 ETF 列表，默认按资产规模（AUM）降序，支持分页；返回行情快照与多周期涨跌幅。
机器可读规格见
[Meta/OpenAPI/MooMoo/QuoteAPIV2/GetEtfList.yaml](../../../../Meta/OpenAPI/MooMoo/QuoteAPIV2/GetEtfList.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://www.moomoo.com` |
| 路径 | `/quote-api/quote-v2/get-etf-list` |
| operationId | `getEtfList` |
| 代码入口 | `Client.GetEtfList`（`component/quote/moomoo/quoteapiv2` 包） |
| 备注 | 页码从 **0** 开始；`leverage` 为空也必须以空串参与签名，省略会返回 `-403`；条目结构与股票列表一致。 |

## 请求参数

### 查询参数

参数书写顺序就是签名输入的键序，不能重排。与股票列表不同，本接口的可选参数**都有默认值**，
缺省时补默认值而不是省略该键。

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `routeType` | 否 | string | 站点路由类型，缺省补 `1` | `1` |
| `plateId` | 否 | string | 板块 ID，缺省补 `10002799`（美股全部 ETF）；港股观测为 `900075` | `10002799` |
| `rankType` | 否 | string | 排行类型，缺省补 `6`（按 AUM 降序） | `6` |
| `leverage` | 否 | string | 杠杆筛选；**为空也要以空串发送并参与签名** | 空串 |
| `page` | 否 | string | 页码，从 `0` 开始（`0` 即第 1 页），缺省补 `0` | `0` |
| `pageSize` | 否 | string | 每页数量，缺省补 `50` | `50` |
| `_` | 否 | string | 毫秒级时间戳，缺省时整键省略 | `1789943966621` |

`leverage` 是本接口的易错点：省略该键（而不是传空串）会让签名输入少一个字段，
服务端返回 `-403`。实现中始终保留该参数。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `quote-token` | 是 | 请求签名，10 个字符，生成方式见 [Signature.md](Signature.md) |
| `futu-x-csrf-token` | 是 | 该接口观测到不校验其取值，但按既有约定仍需携带 |
| `cookie` | 是 | 需包含 `csrfToken` 与 `locale` 两项 |
| `accept` | 否 | 固定 `application/json` |
| `referer` | 否 | 来源页，建议带上站点页面地址 |
| `user-agent` | 否 | 浏览器 UA，非浏览器 UA 可能被拒绝 |

### 请求体

`GET` 请求，无请求体，不发送 `content-type`。

## 签名规则

完整算法、校验向量与排查步骤见 [Signature.md](Signature.md)。

全参数示例（注意 `leverage` 是空串但键仍在）：

```json
{"routeType":"1","plateId":"10002799","rankType":"6","leverage":"","page":"0","pageSize":"50","_":"1789943966621"}
```

对应 `quote-token = 217d908009`。不带时间戳的默认参数组合对应 `quote-token = 88cf5ca696`。

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
| `data.pagination` | object | 分页元信息，`page` 从 0 开始 |
| `data.list` | array | 当前页 ETF 条目 |

### 分页对象

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `page` | integer | 当前页码，从 0 开始 |
| `pageSize` | integer | 每页数量 |
| `pageCount` | integer | 总页数 |
| `total` | integer | 总条数，随行情变化 |

### 条目字段

条目结构与 [GetStockList.md](GetStockList.md#条目字段) 完全一致，字段定义不在此重复。
本接口的差异只有三点：

| 字段 | 本接口取值 | 说明 |
| --- | --- | --- |
| `instrumentType` | 数值 `4` | 标的类型为 ETF |
| `leverage` | 字符串，如 `"1"` | 股票列表中普通股为 `null`，此接口为字符串 |
| `trustAum` | 字符串，如 `"2.34万亿"` | 带单位，是本接口的默认排序依据 |

行情类字段（涨跌幅、价格、成交量额、市值、股本）均为带单位或百分号的字符串，
按原文保留；`stockId`、`instrumentType` 为数值。

### 示例

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "pagination": {"page": 0, "pageSize": 50, "pageCount": 122, "total": 6097},
    "list": [
      {
        "stockId": 202861,
        "name": "整体股市指数ETF-Vanguard",
        "stockCode": "VTI",
        "marketLabel": "US",
        "instrumentType": 4,
        "changeRatio": "+0.02%",
        "priceNominal": "375.430",
        "trustAum": "2.34万亿",
        "leverage": "1"
      }
    ]
  }
}
```

## 业务码

| code | 含义 | 处理建议 |
| --- | --- | --- |
| `0` | 成功 | 读取 `data.pagination` 与 `data.list` |
| `-403` | 签名被拒 | 先检查 `leverage` 是否为空的键被省略 |
| `500` | 参数错误 | 检查各参数取值，消息通常为 `Params Error` |
| 其它非 0 | 业务失败 | 按 `message` 排查，不要按成功处理 |

板块 ID 未知时观测到返回空 `list`，具体返回码未逐项验证，调用方应按空列表处理。

## 相关实现

- `etflist.go`：参数默认值、`leverage` 空串保留、分页校验与响应解析
- `stocklist.go`：条目结构与分页对象复用自股票列表
- `client.go`：配置、错误类型与请求发送
- `signature.go`：签名输入拼装与 `quote-token` 计算
- `etflist_test.go`：空 `leverage`、默认参数签名向量、分页校验与错误分类
