# GetQuoteKLine

获取单只 A 股 / ETF / 指数的分钟级（或日级）K 线（同一接口也用于港股 / 美股，市场由
`market_type` 指定）。返回结构为「`keys` + 一整个字符串」的紧凑表格，而不是 JSON 数组，
需要按 `keys` 逐列解析。
机器可读规格见
[Meta/OpenAPI/Baidu/FinScope/GetQuoteKLine.yaml](../../../../Meta/OpenAPI/Baidu/FinScope/GetQuoteKLine.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://finance.pae.baidu.com` |
| 路径 | `/vapi/v1/getquotation` |
| operationId | `getQuoteKLine`（`group=quotation_kline_ab`） |
| 备注 | 与 [GetQuoteFiveDay.md](GetQuoteFiveDay.md) 同一路径，仅 `group` 不同；期货分钟（`quotation_futures_minute`）与外汇（`huilv_minute`）也走同路径但参数不同，见 [GetQuoteFuturesMinute.md](GetQuoteFuturesMinute.md)、[GetQuoteForexMinute.md](GetQuoteForexMinute.md)。 |

## 请求参数

### 查询参数

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `srcid` | 是 | string | 来源标识，固定 `5353` | `5353` |
| `pointType` | 是 | string | 点位类型，固定 `string`（决定 `marketData` 用字符串表格返回） | `string` |
| `group` | 是 | string | 接口分组，本接口固定 `quotation_kline_ab` | `quotation_kline_ab` |
| `newFormat` | 是 | integer | 新返回格式，固定 `1` | `1` |
| `finClientType` | 是 | string | 客户端类型，固定 `pc` | `pc` |
| `code` | 是 | string | 证券代码，格式随 `market_type` 变化（A 股 6 位、港股 5 位、美股字母） | `600009` |
| `query` | 是 | string | 搜索词，取与 `code` 相同的值 | `600009` |
| `market_type` | 是 | string | 市场类型：`ab` A 股 / `hk` 港股 / `us` 美股 | `ab` |
| `name` | 否 | string | 证券简称，可空；传了更贴近浏览器真实请求 | `上海机场` |
| `is_kc` | 是 | integer | 注册制板块标记，见下表 | `0` |
| `ktype` | 是 | string | K 线周期 | `min1` |
| `count` | 是 | integer | 返回条数，建议 `241`（覆盖一个交易日的分钟线） | `241` |

前 5 个为固定参数，后 7 个为业务参数；参数顺序不影响结果（接口无签名）。

### `market_type` 取值

取值字典见 [Market.md](Market.md)：`ab` A 股、`hk` 港股、`us` 美股，含代码格式、
个股页地址与验证状态。`code` 格式随市场变化（A 股 6 位、港股 5 位、美股字母）。

- `ab` 为本机实测值；本目录只验证了 A 股。
- `hk` / `us` 未实测（`getquotation` 本机持续风控）；`group` 名称带 A 股后缀
  （`quotation_kline_ab`），改市场时需同时确认 `group` 是否仍可用，不能只改 `market_type`。

### `is_kc` 取值

| 代码段 | 取值 | 说明 |
| --- | --- | --- |
| `688` / `689` | `1` | 科创板 |
| `30` | `1` | 创业板 |
| `8` / `4` | `1` | 北交所 |
| 其它（含 `60` / `00` / 沪深指数） | `0` | 主板 |

按代码段前缀推断；股票与指数同码（如 `000001`）时由接口自行判定。
港股 / 美股无注册制概念，固定传 `0`。

### `ktype` 取值

| 取值 | 说明 |
| --- | --- |
| `min1` | 1 分钟线，默认值；配合 `count=241` 取一个交易日 |
| `min5` | 5 分钟线 |
| `day` | 日线 |

`min15` / `min30` / `min60` / `week` / `month` 未在百度侧逐一验证；如需其它周期，
先小流量试请求并核对 `keys` 是否变化。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `User-Agent` | 是 | 浏览器 UA，建议 Chrome 131 桌面版 |
| `Referer` | 是 | `https://gushitong.baidu.com/`；个股页地址亦可，如 `https://finance.baidu.com/stock/ab-600009`、`.../stock/hk-01810`、`.../stock/us-NVDA` |

### 请求体

`GET` 请求，无请求体。

### 示例

```bash
curl -sS \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36' \
  -H 'Referer: https://gushitong.baidu.com/' \
  'https://finance.pae.baidu.com/vapi/v1/getquotation?srcid=5353&pointType=string&group=quotation_kline_ab&newFormat=1&finClientType=pc&code=600009&query=600009&market_type=ab&name=%E4%B8%8A%E6%B5%B7%E6%9C%BA%E5%9C%BA&is_kc=0&ktype=min1&count=241'
```

## 响应

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `QueryID` | string | 请求追踪 ID |
| `ResultNum` | integer | K 线接口实测恒为 `0` |
| `ResultCode` | integer | 网关层结果，`0` 表示正常处理（**不代表业务成功**） |
| `Result` | object | 业务数据；风控时被替换为错误对象，见「业务码」 |
| `Result.newMarketData` | object | K 线容器 |
| `Result.newMarketData.headers` | array\<string\> | 中文列名，仅供阅读 |
| `Result.newMarketData.keys` | array\<string\> | 字段名，**决定 `marketData` 的列顺序** |
| `Result.newMarketData.marketData` | string | 全部 K 线，见下节 |

### `marketData` 编码

`marketData` 是单个字符串：记录用 `;` 分隔，字段用 `,` 分隔，字段顺序与 `keys` 一一对应，缺失值写 `--`。

```
1794449220,2026-11-12 10:07,23.94,23.94,26300,23.95,23.92,629596.00,+0.01,+0.04,--,23.93,--,--,--,--,--,--;1794449280,...
```

解析要点：

- 记录数等于分钟数，末尾可能带空记录，需跳过残缺行（字段数 < `len(keys)`）。
- `time` 列是 `"YYYY-MM-DD HH:MM"`，可拆成整数 `date`（`20261112`）与 `time`（`1007`）。
- 数值带 `+` 前缀（`+0.01`）或百分号语义（`0.04` 即 `+0.04%`），直接 `float` 即可。
- `--` 统一转 `None`；`turnoverratio` 在分钟线上常为 `--`。

### 字段顺序

| 下标 | `keys` | 中文列名 | 建议输出 | 说明 |
| --- | --- | --- | --- | --- |
| 0 | `timestamp` | 时间戳 | `ts` | UTC 秒级时间戳，接口原值 |
| 1 | `time` | 时间 | `date` + `time` | 拆成交易日整数与 `HHMM` 整数 |
| 2 | `open` | 开盘 | `open` | 开盘价 |
| 3 | `close` | 收盘 | `close` | 收盘价 |
| 4 | `volume` | 成交量 | `volume` | 成交量 |
| 5 | `high` | 最高 | `high` | 最高价 |
| 6 | `low` | 最低 | `low` | 最低价 |
| 7 | `amount` | 成交额 | `amount` | 成交额 |
| 8 | `range` | 涨跌额 | `changePrice` | 涨跌额 |
| 9 | `ratio` | 涨跌幅 | `changeRatio` | 涨跌幅（%） |
| 10 | `turnoverratio` | 换手率 | `turnoverRatio` | 换手率（%），常为 `--` |
| 11 | `preClose` | 昨收 | `lastClose` | 昨收 |
| 12 | `ma5avgprice` | ma5均价 | `ma05ap` | 5 周期均线价格 |
| 13 | `ma5volume` | ma5成交量 | `ma05vol` | 5 周期均量 |
| 14 | `ma10avgprice` | ma10均价 | `ma10ap` | 10 周期均线价格 |
| 15 | `ma10volume` | ma10成交量 | `ma10vol` | 10 周期均量 |
| 16 | `ma20avgprice` | ma20均价 | `ma20ap` | 20 周期均线价格 |
| 17 | `ma20volume` | ma20成交量 | `ma20vol` | 20 周期均量 |

列序唯一依据是响应里的 `keys`；百度调整列顺序时无需改代码，按 `keys` 重新对应即可。
`keys` 未列出的新字段可沿用原名保留（只有新增语义字段才需要补说明）。

### 示例

以下为 `Min1KLineExample.json` 节选（真实响应快照，`marketData` 截取两条；示例数据，非实时行情）：

```json
{
  "QueryID": "1034999479576860943",
  "ResultNum": 0,
  "ResultCode": 0,
  "Result": {
    "newMarketData": {
      "headers": ["时间戳", "时间", "开盘", "收盘", "成交量", "最高", "最低", "成交额", "涨跌额", "涨跌幅", "换手率", "昨收", "ma5均价", "ma5成交量", "ma10均价", "ma10成交量", "ma20均价", "ma20成交量"],
      "keys": ["timestamp", "time", "open", "close", "volume", "high", "low", "amount", "range", "ratio", "turnoverratio", "preClose", "ma5avgprice", "ma5volume", "ma10avgprice", "ma10volume", "ma20avgprice", "ma20volume"],
      "marketData": "1794449220,2026-11-12 10:07,23.94,23.94,26300,23.95,23.92,629596.00,+0.01,+0.04,--,23.93,--,--,--,--,--,--;1794449280,2026-11-12 10:08,23.95,23.94,16300,23.96,23.94,390346.00,0.00,0.00,--,23.94,--,--,--,--,--,--"
    }
  }
}
```

解析后：

```json
[
  {
    "date": 20261112,
    "time": 1007,
    "ts": 1794449220,
    "open": 23.94,
    "close": 23.94,
    "volume": 26300,
    "high": 23.95,
    "low": 23.92,
    "amount": 629596.0,
    "changePrice": 0.01,
    "changeRatio": 0.04,
    "turnoverRatio": null,
    "lastClose": 23.93,
    "ma05ap": null,
    "ma05vol": null,
    "ma10ap": null,
    "ma10vol": null,
    "ma20ap": null,
    "ma20vol": null
  }
]
```

## 业务码与异常

| 情形 | 表现 | 处理建议 |
| --- | --- | --- |
| 正常 | `ResultCode=0`，`Result.newMarketData.marketData` 为字符串 | 按 `keys` 解析 |
| 风控 | 外层 `ResultCode=0`，`Result` 为 `{"code":403,"isCaptchaEnabled":true,"msg":"hit risk"}` | 视为失败，不返回半成品 |
| 无数据 / 停牌 | `marketData` 为空串或 `newMarketData` 缺失 | 返回空结果 |
| 记录残缺 | 某条记录字段数 < `len(keys)` | 跳过该条 |
| 周期不支持 | `ktype` 非法时返回空 `marketData` 或风控 | 先用 `min1` 验证参数集 |

`Result` 不是对象（如被降级为数组）同样应视为失败。

