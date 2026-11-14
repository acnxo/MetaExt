# GetQuoteFiveDay

获取单只 A 股 / ETF / 指数（同一接口也用于港股 / 美股，市场由 `market_type` 指定）
近五个交易日的分钟分时（价格、成交量、涨跌），按天分组返回。与 [GetQuoteKLine.md](GetQuoteKLine.md)
同路径、同参数集，仅 `group` 不同，且不需要 `ktype` / `count`。期货五日不走这条路径，
而是 `/selfselect/getstockquotation`，见 [GetQuoteFuturesFiveDay.md](GetQuoteFuturesFiveDay.md)。
机器可读规格见
[Meta/OpenAPI/Baidu/FinScope/GetQuoteFiveDay.yaml](../../../../Meta/OpenAPI/Baidu/FinScope/GetQuoteFiveDay.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://finance.pae.baidu.com` |
| 路径 | `/vapi/v1/getquotation` |
| operationId | `getQuoteFiveDay`（`group=quotation_fiveday_ab`） |
| 备注 | 风控比 K 线接口更严；返回的分钟点是「按天分组的字符串数组」，不是对象数组。 |

## 请求参数

### 查询参数

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `srcid` | 是 | string | 来源标识，固定 `5353` | `5353` |
| `pointType` | 是 | string | 点位类型，固定 `string` | `string` |
| `group` | 是 | string | 接口分组，本接口固定 `quotation_fiveday_ab` | `quotation_fiveday_ab` |
| `newFormat` | 是 | integer | 新返回格式，固定 `1` | `1` |
| `finClientType` | 是 | string | 客户端类型，固定 `pc` | `pc` |
| `code` | 是 | string | 证券代码，格式随 `market_type` 变化（A 股 6 位、港股 5 位、美股字母） | `301618` |
| `query` | 是 | string | 搜索词，取与 `code` 相同的值 | `301618` |
| `market_type` | 是 | string | 市场类型：`ab` A 股 / `hk` 港股 / `us` 美股 | `ab` |
| `name` | 否 | string | 证券简称，可空；传了更贴近浏览器真实请求 | `长联科技` |
| `is_kc` | 是 | integer | 注册制板块标记，规则同 K 线接口（688/689/30x/8xx/4xx → `1`）；港股 / 美股固定 `0` | `1` |

本接口没有 `ktype`、`count`：周期固定为分钟、窗口固定为近五个交易日。

### `market_type` 取值

取值字典见 [Market.md](Market.md)：`ab` A 股、`hk` 港股、`us` 美股，含代码格式、
个股页地址与验证状态。`code` 格式随市场变化（A 股 6 位、港股 5 位、美股字母）。

- `ab` 为本机实测值；本目录只验证了 A 股。
- `hk` / `us` 未实测（`getquotation` 本机持续风控）；`group` 名称带 A 股后缀
  （`quotation_fiveday_ab`），改市场时需同时确认 `group` 是否仍可用，不能只改 `market_type`。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `User-Agent` | 是 | 浏览器 UA，建议 Chrome 131 桌面版 |
| `Referer` | 是 | `https://gushitong.baidu.com/`；个股页地址亦可，如 `https://finance.baidu.com/stock/hk-01810` |

### 请求体

`GET` 请求，无请求体。

### 示例

```bash
curl -sS \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36' \
  -H 'Referer: https://gushitong.baidu.com/' \
  'https://finance.pae.baidu.com/vapi/v1/getquotation?srcid=5353&pointType=string&group=quotation_fiveday_ab&newFormat=1&finClientType=pc&code=399001&query=399001&market_type=ab&name=%E6%B7%B1%E8%AF%81%E6%88%90%E6%8C%87&is_kc=0'
```

## 响应

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `QueryID` | string | 请求追踪 ID |
| `ResultCode` | integer | 网关层结果，`0` 表示正常处理（**不代表业务成功**） |
| `Result` | object | 业务数据；风控时被替换为错误对象，见「业务码」 |
| `Result.StdStg` / `Result.StdStl` / `Result.provider` | string | 上游模块标识（实测 `hundsun_fivedays` / `8` / `BAIDU`），排障用 |
| `Result.code` / `Result.key` | string | 证券代码 / 查询键，两者相同 |
| `Result.update` | object | 数据时间：`timezone`（`北京时间`）、`text`、`time`（秒级）、`time_diff` |
| `Result.newMarketData` | object | 分时容器 |
| `Result.newMarketData.headers` | array\<string\> | 中文列名，与 `keys` 一一对应 |
| `Result.newMarketData.keys` | array\<string\> | 字段名，**决定 `p` 每条的列顺序** |
| `Result.newMarketData.marketData` | array | 按交易日分组的数组，元素见下节 |
| `Result.newMarketData.maxPoints` | string | 单日最大点数（实测 `"1201"`），排障用 |
| `Result.newMarketData.cx` | array\<string\> | 横轴日期标签，如 `["11-09", "11-10", ...]` |

### 每日元素

`Result.newMarketData.marketData[]` 的一个元素代表一个交易日（或一天内的一组分钟点）。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `date` | string | 交易日，`YYYY-MM-DD`，如 `2026-11-09` |
| `p` | string \| array | 该交易日的全部分时点。实测为**字符串**（`;` 分记录、`,` 分字段），列序由 `keys` 给出；旧数据 / 示例中可能是数组 |
| `preClose` | number | 兼容字段：同级带昨收时用于派生涨跌额 / 涨跌幅，并作为 `lastClose`；实测响应未带，取 `None` |
| `code` / `name` / `dates` | string / string / array | 部分形态会带代码、名称与日期列表；实测元素只有 `date` 与 `p` |

### `p` 编码

实测响应为单个字符串：记录用 `;` 分隔，字段用 `,` 分隔。每条记录的列名与列序由
`newMarketData.keys` 给出，`headers` 是其中文名，**解析时对标 `keys`，不要硬编码列位**：

```text
"1794187860,11-09 09:31,11256.33,+80.55,+0.72,2784792751.00,41449256787.00;1794187920,..."
```

实测列序（2026-11 核对为 **10 列**；早期响应为 7 列，多出均价与累积量额三列）：

| 下标 | `keys` | `headers` | 建议输出 | 说明 |
| --- | --- | --- | --- | --- |
| 0 | `timestamp` | 时间戳 | `ts` | UTC 秒级时间戳 |
| 1 | `time` | 时间 | `date` + `time` | `MM-DD HH:MM`，**无年份**，年份取自同级 `date` |
| 2 | `price` | 价格 | `close` | 该分钟价格 |
| 3 | `avgPrice` | 均价 | `avgPrice` | 该分钟均价；7 列响应无此列 |
| 4 | `range` | 涨跌额 | `changePrice` | 涨跌额 |
| 5 | `ratio` | 涨跌幅 | `changeRatio` | 涨跌幅（%） |
| 6 | `volume` | 成交量 | `volume` | 成交量 |
| 7 | `amount` | 成交额 | `amount` | 成交额 |
| 8 | `totalVolume` | 累积成交量 | `totalVolume` | 当日累积成交量；7 列响应无此列 |
| 9 | `totalAmount` | 累积成交额 | `totalAmount` | 当日累积成交额；7 列响应无此列 |

单条记录按 `keys` 逐项拆开（示例为早期 7 列响应，10 列响应同理，只是多出均价与累积列）：

```text
"1794187860" → timestamp   = 1794187860
"11-09 09:31" → time        = 11-09 09:31（年份取同级 date）
"11256.33"   → price       = 11256.33
"+80.55"     → range       = +80.55
"+0.72"      → ratio       = +0.72%
"2784792751.00" → volume    = 2784792751
"41449256787.00" → amount   = 41449256787
```

把每条记录按 `,` 拆成字段数组，再与 `keys` 逐位对应即可。列数少于 `keys` 时对应字段取空值，
多出的列丢弃。

兼容形态（旧数据可能是以下形式）：

| 形态 | 示例 | 说明 |
| --- | --- | --- |
| 字符串（实测） | `"1794187860,11-09 09:31,11256.33,..."` | `;` 分记录，列序看 `keys` |
| 二维数组 | `[["0930,10.50,15200,0.12,1.16", ...], [...]]` | 外层下标对应 `dates`，内层是一天 |
| 一维数组 | `["0930,10.50,15200,0.12,1.16", ...]` | 直接是一天的分钟点 |
| 对象数组 | `[{"t": "0930", "p": 10.5, "v": 15200}]` | 键名兼容 `t/time`、`p/close/price`、`v/volume` |

> **注意**：不要按旧示例把列序硬编码为 5 列或 7 列（`time, close, volume, changePrice, changeRatio`）；
> 2026-11 实测为 10 列（比早期多 `avgPrice`、`totalVolume`、`totalAmount`），
> 且百度可能继续调整，必须按 `keys` 对齐，否则 `time` / `price` 会错位。

### 派生字段

| 字段 | 来源 | 说明 |
| --- | --- | --- |
| `date` | 元素 `date`（或 `dates[下标]`） | 去掉 `-` 转整数，如 `20261109` |
| `time` | `time` 列 | 取空格后的 `HH:MM`，去冒号转整数，如 `931` |
| `ts` | `timestamp` 列 | 接口原值（UTC 秒），如 `1794187860` |

`ts` 取接口 `timestamp`；`date` / `time` 同时保留，便于直接阅读与按交易日分组。
`time` 列不带年份，`date` 必须从元素级 `date` 取，不能从 `time` 拼接。
也可由 `date` + `time`（北京时间）换算出 `ts`，结果与接口 `timestamp` 一致；直接用接口原值更省事。

### 示例

以下为**早期 7 列**响应的节选（`code=399001`，行情摘自 2026-11-09 开盘两条，其余字段保留原结构；
数值为当时快照，不代表实时行情）。`p` 内每条记录都按 `keys` 解读，不要按本文的列数硬编码。

```json
{
  "QueryID": "15325184726536934777",
  "ResultCode": 0,
  "ResultNum": 0,
  "Result": {
    "StdStg": "hundsun_fivedays",
    "code": "399001",
    "key": "399001",
    "update": { "timezone": "北京时间", "text": "2026-11-13 15:00:00", "time": 1794553200 },
    "newMarketData": {
      "headers": ["时间戳", "时间", "价格", "涨跌额", "涨跌幅", "成交量", "成交额"],
      "keys": ["timestamp", "time", "price", "range", "ratio", "volume", "amount"],
      "marketData": [
        {
          "date": "2026-11-09",
          "p": "1794187860,11-09 09:31,11256.33,+80.55,+0.72,2784792751.00,41449256787.00;1794187920,11-09 09:32,11245.94,+70.16,+0.63,1252108048.00,18677359165.00"
        }
      ],
      "maxPoints": "1201",
      "cx": ["11-09"]
    }
  }
}
```

`marketData` 只保留了首日、每日只保留两条记录，`cx` 同理截断；真实响应为 5 个交易日 × 每日约 240 条。

解析后（字段名按 `keys` 映射；实测元素未带 `preClose`，故 `lastClose` 为 `null`）：

```json
[
  {
    "date": 20261109,
    "time": 931,
    "ts": 1794187860,
    "close": 11256.33,
    "changePrice": 80.55,
    "changeRatio": 0.72,
    "volume": 2784792751,
    "amount": 41449256787,
    "lastClose": null
  },
  {
    "date": 20261109,
    "time": 932,
    "ts": 1794187920,
    "close": 11245.94,
    "changePrice": 70.16,
    "changeRatio": 0.63,
    "volume": 1252108048,
    "amount": 18677359165,
    "lastClose": null
  }
]
```

## 业务码与异常

| 情形 | 表现 | 处理建议 |
| --- | --- | --- |
| 正常 | `ResultCode=0`，`Result.newMarketData.marketData` 为非空数组 | 按 `keys` 解析每日 `p`，展开成逐条分钟行情 |
| 风控 | 外层 `ResultCode=0`，`Result` 为 `{"code":403,"isCaptchaEnabled":true,"msg":"hit risk"}` | 视为失败 |
| 无数据 | `marketData` 缺失或为空数组 | 返回空结果 |
| 停牌 / 新券 | 某些交易日无分钟点 | 跳过该天，不补空行 |

该接口实测风控比 K 线接口更敏感：请求参数已按浏览器请求构造，稳定采集需降低频率或更换出口网络。

