# GetConstituents

获取指定指数或概念板块的成分股列表，按 `sortKey` 排序后分页返回；每条记录同时带上该证券的
行情快照，因此一次分页即可完成「成分股 + 行情」采集。
机器可读规格见
[Meta/OpenAPI/Baidu/FinScope/GetConstituents.yaml](../../../../Meta/OpenAPI/Baidu/FinScope/GetConstituents.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://finance.pae.baidu.com` |
| 路径 | `/sapi/v1/constituents` |
| operationId | `getConstituents` |
| 备注 | `pn` 是**偏移量**而非页码，必须 `pn += rn` 递进；无翻页游标，以返回条数小于 `rn` 判定末页。 |

## 请求参数

### 查询参数

参数顺序不影响结果（接口无签名）。

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `financeType` | 是 | string | 成分类型：`index` 指数、`block` 概念 / 行业板块 | `index` |
| `market` | 是 | string | 市场：`ab` A 股 / `hk` 港股 / `us` 美股；目前只验证了 A 股 | `ab` |
| `code` | 是 | string | 标识码，格式随 `market` 变化：A 股指数为 6 位代码、板块为 `GNxxxx`；港股 5 位数字、美股字母代码 | `000300` |
| `sortKey` | 是 | string | 排序字段：`marketValue` 总市值、`amount` 成交额 | `marketValue` |
| `sortType` | 是 | string | 排序方向：`desc` / `asc` | `desc` |
| `style` | 是 | string | 返回样式，固定 `tablelist`（表格结构） | `tablelist` |
| `pn` | 是 | integer | **偏移量**，首屏为 `0`，下一页 `pn += rn` | `0` |
| `rn` | 是 | integer | 单页条数，常用 `500`。实测 `rn=5000` 时一次返回该来源全部 2236 条，没有 500 硬上限 | `500` |
| `finClientType` | 是 | string | 客户端类型，固定 `pc` | `pc` |

> **翻页陷阱**：把 `pn` 当页码按 `pn += 1` 翻页，每页只会新增 1 条数据，最终严重漏采。

### `market` 取值

取值字典见 [Market.md](Market.md)：`ab` A 股、`hk` 港股、`us` 美股，含代码格式、
个股页地址与验证状态。目前只验证了 A 股；成分股来源（指数 / 板块）均为 A 股，改写
`market` 时需同时确认 `code` 与 `financeType` 是否成立。

### 来源码说明

| `financeType` | `code` 示例 | 含义 |
| --- | --- | --- |
| `index` | `000300` | 沪深300 |
| `index` | `000001` | 上证综指（覆盖沪市全量） |
| `index` | `399106` | 深证综指（覆盖深市全量） |
| `index` | `399380` | 国证ETF（全市场 ETF 池） |
| `block` | `GN2309` | 北交所上市企业 |
| `block` | `GN2282` | 微盘股 概念 |

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `User-Agent` | 是 | 浏览器 UA，建议 Chrome 131 桌面版 |
| `Referer` | 是 | `https://gushitong.baidu.com/` |

### 请求体

`GET` 请求，无请求体。

### 示例

```bash
curl -sS \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36' \
  -H 'Referer: https://gushitong.baidu.com/' \
  'https://finance.pae.baidu.com/sapi/v1/constituents?financeType=index&market=ab&code=000300&sortKey=marketValue&sortType=desc&style=tablelist&pn=0&rn=500&finClientType=pc'
```

## 响应

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `QueryID` | string | 请求追踪 ID |
| `ResultCode` | integer | 网关层结果，`0` 表示正常处理 |
| `Result` | object | 业务数据 |
| `Result.list` | object | 表格数据容器 |
| `Result.list.headers` | array | 列定义：`key`（字段名）、`name`（中文列名）、`canSort`（是否可排序） |
| `Result.list.body` | array | 成分股数组，元素结构见下节 |
| `Result.pcUrl` | string | 该指数的 PC 页面地址，采集可忽略 |
| `Result.url_xcx_params` / `Result.webStatusUrl` | object / string | 小程序跳转参数，采集可忽略 |

实测该接口不返回 `ResultNum`。

### 成分股字段

`Result.list.body[]` 的元素。**同一指标有两份**：顶层的带单位展示字符串（给人看）与
`rawData` 中的原始数值（给程序用），采集一律取 `rawData`。

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `code` | string | 证券代码，去重主键 | `601939` |
| `name` | string | 证券简称 | `建设银行` |
| `exchange` | string | 交易所：`SH` / `SZ` / `BJ` | `SH` |
| `market` | string | 市场标识，取值同请求参数：`ab` / `hk` / `us` | `ab` |
| `financeType` | string | 品种类型；本接口对股票与 ETF 都返回 `stock` | `stock` |
| `hasQuote` | integer | 是否有行情：`1` / `0` | `1` |
| `lastPx` | string | 最新价（元），带符号 | `12.11` |
| `pxChange` | string | 涨跌额（元），带符号 | `+0.10` |
| `pxChangeRate` | string | 涨跌幅（%），带 `%` | `+0.83%` |
| `volume` | string | 成交量，带中文单位 | `128万手` |
| `amount` | string | 成交额，带中文单位 | `15.42亿` |
| `turnoverRatio` | string | 换手率（%），带 `%` | `1.33%` |
| `marketValue` | string | 总市值，带中文单位 | `3.17万亿` |
| `rawData` | object | 上述指标的原始数值，见下表 | — |
| `block` | object | 所属行业板块：`code` / `name` / `market` / `financeType` | `480200` |
| `logo` | object | `type` / `logo`（Logo 图地址） | — |
| `pcUrl` | string | 个股 PC 页面地址，形如 `https://finance.baidu.com/stock/<market>-<code>` | `https://finance.baidu.com/stock/ab-601939` |
| `url_xcx_params` | object | 小程序跳转参数，采集可忽略 | — |
| `webStatusUrl` | string | 小程序页面地址，过长且无采集价值，可忽略 | — |

### `rawData` 字段

| 字段 | 类型 | 单位 / 口径 |
| --- | --- | --- |
| `lastPx` | number | 元 |
| `pxChange` | number | 元，可直接为负 |
| `pxChangeRate` | number | 百分数，`0.83` 即 `+0.83%` |
| `volume` | number | 手 |
| `amount` | number | 元 |
| `turnoverRatio` | number | 百分数 |
| `marketValue` | number | 元 |

`rawData` 的数值未做单位换算，也不会出现 `"3万手"`、`"12.65亿"` 这类字符串，
因此应以 `rawData` 为唯一数值来源，缺失时才回退到顶层字符串字段。

### 示例

以下为实测响应节选（`code=000300`、`rn=2`，行情数值会随时间变化）：

```json
{
  "QueryID": "12457052143672835826",
  "ResultCode": 0,
  "Result": {
    "list": {
      "headers": [
        { "canSort": 1, "key": "lastPx", "name": "最新价" },
        { "canSort": 1, "key": "pxChangeRate", "name": "涨跌幅" }
      ],
      "body": [
        {
          "code": "601939",
          "name": "建设银行",
          "exchange": "SH",
          "market": "ab",
          "financeType": "stock",
          "hasQuote": 1,
          "lastPx": "12.11",
          "pxChange": "+0.10",
          "pxChangeRate": "+0.83%",
          "volume": "128万手",
          "amount": "15.42亿",
          "turnoverRatio": "1.33%",
          "marketValue": "3.17万亿",
          "rawData": {
            "lastPx": 12.11,
            "pxChange": 0.1,
            "pxChangeRate": 0.83,
            "volume": 1276070.86,
            "amount": 1542319057,
            "turnoverRatio": 1.33,
            "marketValue": 3167980619468
          },
          "block": { "code": "480200", "name": "国有大型银行Ⅱ", "market": "ab", "financeType": "block" },
          "logo": { "type": "normal", "logo": "https://baidu-finance.cdn.bcebos.com/imgs/logo/stocks/stock_ab_601939.svg" },
          "pcUrl": "https://finance.baidu.com/stock/ab-601939"
        }
      ]
    },
    "pcUrl": "https://finance.baidu.com/index/ab-000300?mainTab=成分股"
  }
}
```

## 业务码与边界

| 情形 | 表现 | 处理建议 |
| --- | --- | --- |
| 正常 | `ResultCode=0` 且 `Result.list.body` 有数据 | 取 `body`，按 `code` 去重合并 |
| 偏移量越界 | `ResultCode=0`，`Result.list` 为 `{}` 或 `body` 为空数组（实测 `pn=999999` 返回 `list: []`） | 视为无更多数据，停止翻页 |
| 末页 | `body` 长度 < `rn` | 停止翻页 |
| 参数错误 | 代码 / `financeType` 不匹配时 `body` 为空 | 核对 `code` 与 `financeType` 的组合 |

调用方应先校验 `Result` 为对象且 `Result.list` 为对象，再取 `body`；类型异常时按失败处理，
不要把异常响应当作空页继续翻页（另建议对偏移量设置上限，防止死循环）。

