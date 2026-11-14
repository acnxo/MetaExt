# GetQuoteSnapshotCNAStock

一次请求拿到 A 股全市场行情快照（`action=20199` + 板块内部码 `994650`），返回
「表头 + 竖线分隔记录」的行情表；默认 45 个字段，可扩展到 462 个。
机器可读规格见
[Meta/OpenAPI/Hqgw/Snapshot/GetQuoteSnapshotCNAStock.yaml](../../../../Meta/OpenAPI/Hqgw/Snapshot/GetQuoteSnapshotCNAStock.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `POST`（也接受 `GET` + query，参数相同） |
| 站点 | 上游行情网关（地址由 `Config.GatewayURL` 指定） |
| 路径 | `/reqxml` |
| 站点动作 | `action=20199`（指数 / 板块 → 成分行情） |
| operationId | `getQuoteSnapshotCNAStock` |
| 代码入口 | `snapshot.Client.GetQuoteSnapshotCNAStock`（`component/quote/hqgw/snapshot` 包） |
| 鉴权 | 无（无需登录态、无需 cookie、无签名） |
| 备注 | **一次请求即可拿全市场**，无需分页；返回第一行恒为该池自身，需剔除；`MaxCount` 必须显式传 |

## 请求参数

### 表单参数

参数顺序无关；可选参数缺省时整键省略。

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `action` | 是 | int | 站点动作号，固定 `20199` | `20199` |
| `ReqlinkType` | 是 | int | 行情通道固定 `0`；传 `2` 会走业务网关并报错 | `0` |
| `StockCode` | 是 | string | 板块 / 指数**内部代码**，全市场用 `994650` | `994650` |
| `MaxCount` | 是 | int | 返回行数上限。**不传只回 2 行**；超过成分数按实际返回 | `60000` |
| `StartPos` | 否 | int | 分页起点，从 `1` 开始，是**成分序号**而非行号 | `1` |
| `NeedCheck` | 否 | string | 字段白名单，竖线分隔的字段码（`1`–`462`），末尾同样带竖线 | `1\|2\|4\|92\|208\|` |

常用 `StockCode`（包内已内置为别名，见 `snapshot.Pools`）：

| 内部代码 | 别名 | 标的 | 成分数（实测） |
| --- | --- | --- | --- |
| `994650` | `all` | 沪深京全A（含北交所 / ST） | 5579~5580 |
| `994649` | `sh_sz` | 沪深A股（不含北交所） | 5225 |
| `1B0985` | `all_share` | 中证全指（指数样本） | 5117 |
| `000300` | `hs300` | 沪深300 | 300 |
| `993665` | `army` | 军工（行业板块） | 448 |

> 内部代码与证券代码不是一套：请求 `000300` 时响应回显的 `STOCKCODE` 是 `1A0300`。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `content-type` | 是 | `POST` 用 `application/x-www-form-urlencoded` |
| `user-agent` | 否 | 浏览器 UA，站点不校验 |

无需 `cookie`、`quote-token`、`referer` 等任何鉴权或签名头。

### 请求体

```
action=20199&ReqlinkType=0&StockCode=994650&MaxCount=60000&NeedCheck=1|2|4|10|11|92|208|
```

## 字段选择（`NeedCheck`）

| 规则 | 说明 |
| --- | --- |
| 列顺序 | 以响应回显的 `NEEDCHECK` 为准（服务端顺序为 `1,4,3,2,5,6,...`），**不按传参顺序** |
| 非法码 | 作为列名原样返回，值为 `--`；`463` 起为占位，不报错但恒为 `--` |
| 缺省 | 不传时返回默认 20 列 |
| 多段字段 | 只有码 `33`–`42`（卖五…买五）每码两段 `价格\|数量`，其余均为单段 |
| 五档推荐 | 用 `93`–`102`（价格，元）+ `180`–`189`（数量，手），单段可靠 |
| 列索引 | 顶层 `STOCKCODEINDEX` 等 `*INDEX` 是默认 20 列的索引，传自定义 `NeedCheck` 后不更新 |

Go 客户端默认取 45 个字段（`snapshot.DefaultFields`），追加用 `SnapshotRequest.ExtraFields`，
完全自定义用 `Fields`。

## 响应

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `GRID0` | array\<string\> | `[0]` 为表头，`[1]` 恒为池自身，其后为成分 |
| `ACTION` | string | 动作号回显 `"20199"` |
| `MAXCOUNT` | string | 成分总数（不是返回行数） |
| `ERRORNO` | string | **非标准错误码**：正值随行数变化（实测 `行数 + 2`），**只有负值才是错误** |
| `NEEDCHECK` | string | 服务端实际生效的字段码，列顺序以它为准 |
| `STOCKCODE` / `STOCKNAME` | string | 池自身的内部代码与名称 |
| `HQDATE` / `SERVERDATETIME` | string | 行情日期 `yyyyMMdd` / 服务器时间 |
| `MARKETSTATUS` | string | 逐只市场状态串，与行数等长 |
| `ERRORMESSAGE` | string | 参数错误时的中文提示，仅错误时出现 |

### 记录结构

```
GRID0[0] = 名称|最新|幅度|...|代码|            ← 表头，列集合由 NEEDCHECK 决定
GRID0[1] = 沪深京全A|1488.03|-2.08%|...|994650|  ← 池自身（代码为内部码）
GRID0[2] = 1.幸福蓝海|17.99|20.01%|...|300528|   ← 成分，名称带递增序号前缀
```

- 记录末尾同样带 `|`，切分后要去掉最后一个空元素；
- `名称` 带 `1.` `2.` 递增序号前缀（跨行连续），需剥离；
- 列数 = 表头列数；含 `33`–`42` 时数据段数 = 列数 + 10，个别标的会少段。

### 条目字段（默认 45 个）

Go 客户端会把带单位的原始值转成数值：`%` 去掉，`万`/`亿`/`万亿` 还原成元，
`--` 转成零值（`Values` 中不出现该 key）。字段码与变量名的对应见下表。

| 码 | `Values` 中的 key | Go 字段 | 说明 |
| --- | --- | --- | --- |
| `1` | `name` | `Name` | 名称，已剥离 `1.` 序号 |
| `2` | `last` | `Last` | 最新价（元） |
| `3` | `change_price` | `ChangePrice` | 涨跌额（元） |
| `4` | `change_ratio` | `ChangeRatio` | 涨跌幅（%） |
| `5` | `pre_close` | `PreClose` | 昨收（元） |
| `6` | `open` | `Open` | 今开（元） |
| `7` | `high` | `High` | 最高（元） |
| `8` | `low` | `Low` | 最低（元） |
| `9` | `avg_price` | `AvgPrice` | 均价（元） |
| `10` | `volume` | `Volume` | 成交量（手） |
| `11` | `amount` | `Amount` | 成交额（元） |
| `12` | `last_volume` | `LastVolume` | 现手（手） |
| `13` | `turnover_rate` | `TurnoverRate` | 换手率（%） |
| `14` | `bid_ask_ratio` | `BidAskRatio` | 委比（%） |
| `15` | `bid_ask_diff` | `BidAskDiff` | 委差（手，可为小数） |
| `16` | `amplitude` | `Amplitude` | 振幅（%） |
| `32` | `code` | `Code` | 证券代码 |
| `48` | `outer_volume` | `OuterVolume` | 外盘（手） |
| `49` | `inner_volume` | `InnerVolume` | 内盘（手） |
| `50` | `volume_ratio` | `VolumeRatio` | 量比 |
| `92` | `pe` | `PE` | 市盈率（静态，亏损为负） |
| `162` | `net_inflow` | `NetInflow` | 当日净流入（元） |
| `208` | `pb` | `PB` | 市净率 |
| `209` | `suspend_flag` | `SuspendFlag` | 停牌标志（`0/1`，语义以 `210` 为准） |
| `210` | `trade_status` | `TradeStatus` | 停牌状态，如 `正常交易` |
| `225` | `limit_up_price` | `LimitUpPrice` | 涨停板价（元） |
| `226` | `limit_down_price` | `LimitDownPrice` | 跌停板价（元） |
| `227` | `dynamic_pe` | `DynamicPE` | 动态市盈率 |
| `228` | `bvps` | `BVPS` | 每股净资产（元） |
| `229` | `total_shares` | `TotalShares` | 总股本（股） |
| `230` | `market_cap` | `MarketCap` | 总市值（元） |
| `231` | `float_shares` | `FloatShares` | 流通股本（股，列名「流通盘」） |
| `232` | `float_market_cap` | `FloatMarketCap` | 流通市值（元） |
| `233` | `eps` | `EPS` | 每股收益（元） |
| `256` | `roe` | `ROE` | 净资产收益率（%） |
| `294` | `change_pct_20d` | `ChangePct20D` | 20 日涨跌幅（%） |
| `295` | `change_pct_60d` | `ChangePct60D` | 60 日涨跌幅（%） |
| `296` | `change_pct_120d` | `ChangePct120D` | 120 日涨跌幅（%） |
| `297` | `change_pct_250d` | `ChangePct250D` | 250 日涨跌幅（%） |
| `298` | `change_pct_5d` | `ChangePct5D` | 5 日涨跌幅（%） |
| `306` | `pe_ttm` | `PETTM` | 市盈率 TTM |
| `335` | `dividend_yield` | `DividendYield` | 股息率（%） |
| `342` | `change_pct_week` | `ChangePctWeek` | 本周涨幅（%，实测与 `298` 同值） |
| `408` | `dividend` | `Dividend` | 每股股息（元） |
| `453` | `dividend_yield_ttm` | `DividendYieldTTM` | 股息率 TTM（形如 `0.04`） |

其余字段码仍可通过 `NeedCheck` 请求，结果出现在 `Row.Values` 中，key 为英文字段名；
未命名的码 key 为 `f<码>`。五档（`93`–`102` 价 + `180`–`189` 量，或 `33`–`42` 复合段）
汇总在 `Values["depth"]`，类型为 `map[string]DepthLevel`。

### 示例

成功响应片段（`StockCode=994650`）：

```json
{"GRID0":[
 "名称|最新|幅度|涨跌|昨收|开盘|最高|最低|均价|总手|总额|现手|换手|振幅|委比|委差|量比|内盘|外盘|代码|",
 "沪深京全A|1488.03|-2.08%|-31.58|1519.61|1522.37|1526.09|1484.97|1504.97|10.37亿手|1.67万亿|1216.23万手|1.41%|2.71%|-2.04%|-125.33万手|1.02|--|--|994650|",
 "1.幸福蓝海|17.99|20.01%|3.00|14.99|14.90|17.99|14.90|17.05|60.75万手|10.36亿|8手|16.30%|20.61%|100.00%|7211.80手|1.92|27.02万|33.73万|300528|"],
 "ACTION":"20199","MAXCOUNT":"5579","ERRORNO":"5581",
 "NEEDCHECK":"1|2|4|3|5|6|7|8|9|10|11|12|13|16|14|15|50|49|48|32|",
 "STOCKCODE":"994650","STOCKNAME":"沪深京全A","HQDATE":"20261113"}
```

解析后的 `Row`（节选）：

```json
{
  "Code": "300528",
  "Name": "幸福蓝海",
  "Last": 17.99,
  "ChangeRatio": 20.01,
  "Volume": 607500,
  "Amount": 1036000000,
  "InnerVolume": 270200,
  "OuterVolume": 337300,
  "TurnoverRate": 16.3,
  "VolumeRatio": 1.92
}
```

参数错误的响应同样是 HTTP 200：

```json
{"ERRORNO":"-206008","ERRORMESSAGE":"未找到匹配的证券代码!代码:999999|0"}
```

## 业务码与异常

以响应体 `ERRORNO` 为准，**HTTP 状态码恒为 200**。

| `ERRORNO` | 含义 | 处理建议 |
| --- | --- | --- |
| 正值（如 `5581`） | **正常**，数值随返回行数变化（实测 `行数 + 2`） | 按成功处理 |
| `-206008` | 未找到匹配的证券代码 | 检查 `StockCode` 是否为内部代码 |
| `-206021` | 查无相应的排名信息 | 该代码不支持本接口（如 ETF 代码 `510300`） |
| `-989905` | 功能被服务端禁用 | 换接口或放弃该能力 |
| 其它负值 | 业务失败 | 读 `ERRORMESSAGE` |

传输层还有一个实测坑：全市场单次请求**偶发**返回 HTTP 200 但响应体被截断
（JSON 不完整）。客户端内置 4 次退避重试，这类错误通常第二次即成功；
业务错误（负值 `ERRORNO`）不重试。

## 客户端用法

```go
client := snapshot.New(snapshot.Config{})
result, err := client.GetQuoteSnapshotCNAStock(ctx, snapshot.SnapshotRequest{
    Pool: "hs300",              // 别名或内部代码，空取 all
    ExtraFields: []int{341, 372}, // 在默认 45 字段上追加：本月涨幅、5日净流入
})
for _, row := range result.Rows {
    fmt.Println(row.Code, row.Name, row.Last, row.ChangeRatio, row.MarketCap)
}
```

- `SnapshotRequest.Fields` 完全自定义字段集，`ExtraFields` 在默认集上追加；
- `Row` 的默认 45 个字段是强类型成员，其余字段在 `Row.Values` 里按英文 key 取；
- 池自身那一行（`GRID0[1]`）已剔除，名称的 `1.` 序号前缀已剥离。

## 实测记录

| 项 | 值 |
| --- | --- |
| 验证日期 | 2026-11 |
| 全市场（`994650`） | 5579 只，单次请求约 7.6 秒 |
| 沪深300（`000300`） | 300 只，响应回显 `STOCKCODE=1A0300`，约 1.8 秒 |
| 军工板块（`993665`） | 448 只，约 2.1 秒 |
| 行情日期 | 响应 `HQDATE` 为 `20261113`，与解析结果一致 |
| 数据核对 | 首条 `300528 幸福蓝海 17.99 +20.01%` 与接口文档中的示例逐字段一致 |
