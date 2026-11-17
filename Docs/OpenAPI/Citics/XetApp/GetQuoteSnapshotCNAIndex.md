# GetQuoteSnapshotCNAIndex

拿到沪深京全量指数行情快照（**合并 4 个板块池**，去重后 **879 只**，默认 45 个字段）。
站点没有「全部指数」的单一入口，因此由客户端串起 4 组请求后合并。
机器可读规格见
[Meta/OpenAPI/Citics/XetApp/GetQuoteSnapshotCNAIndex.yaml](../../../../Meta/OpenAPI/Citics/XetApp/GetQuoteSnapshotCNAIndex.yaml)。

姊妹接口：[A 股全市场](GetQuoteSnapshotCNAStock.md) / [沪深 ETF](GetQuoteSnapshotCNAETF.md)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `POST`（也接受 `GET` + query，参数相同） |
| 站点 | 上游行情网关（地址由 `Config.GatewayURL` 指定） |
| 路径 | `/reqxml` |
| 站点动作 | `action=20191`（按市场号）+ `action=20192`（按板块中文名）+ `action=20198`（大盘指数表） |
| operationId | `getQuoteSnapshotCNAIndex` |
| 代码入口 | `xetapp.Client.GetQuoteSnapshotCNAIndex`（`component/quote/citics/xetapp` 包） |
| 鉴权 | 无（无需登录态、无需 cookie、无签名） |
| 备注 | 4 次请求、约 3.8 秒、无分页；四个池**有重叠**，按代码去重（先出现的池优先） |

参与合并的 4 个池（实测，2026-11-13）：

| 池 | 请求 | 成分数 | 说明 |
| --- | --- | --- | --- |
| 中证指数 | `action=20191` + `StockCode=1100` | 832 | 含上证指数 `1A0001`、沪深300 `1A0300`、科创50 `1B0688` |
| 国证指数 | `action=20192` + `StockCode=国证指数` | 20 | 国证系列（`980028` 等） |
| 股转指数 | `action=20192` + `StockCode=股转指数` | 10 | 新三板 / 北交所指数（`899304` 等） |
| 大盘指数 | `action=20198` | 42 | 含深证成指 `2A01`、创业板指 `399006`、北证50 `899050` |
| **去重并集** | — | **879** | 904 行 → 879 只（重叠 25 只） |

覆盖的常见指数：上证指数 `1A0001`、深证成指 `2A01`、创业板指 `399006`、
科创50 `1B0688`、北证50 `899050`、沪深300 `1A0300`、中证2000 `932000`。

## 请求参数

### 四组请求

`【A】中证指数`（`20191`）：

| 参数 | 值 |
| --- | --- |
| `action` | `20191` |
| `ReqlinkType` | `0` |
| `StockCode` | `1100`（**内部市场号**，不是指数代码） |
| `MaxCount` | `60000`（不传只回 1 行） |

`【B】国证指数`与 `【C】股转指数`（`20192`）：

| 参数 | 值 |
| --- | --- |
| `action` | `20192` |
| `ReqlinkType` | `0` |
| `StockCode` | 板块中文名：`国证指数` / `股转指数` |
| `MaxCount` | `60000`（不传只回 1 行） |

`【D】大盘指数`（`20198`）：

| 参数 | 值 |
| --- | --- |
| `action` | `20198` |
| `ReqlinkType` | `0` |
| `StockCode` | 不使用该参数（整键省略，传空串会被忽略） |
| `MaxCount` | `60000`（不传只回 1 行） |

四组共用 `NeedCheck`（见「字段选择」）。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `content-type` | 是 | `POST` 用 `application/x-www-form-urlencoded` |
| `user-agent` | 否 | 浏览器 UA，站点不校验 |

### 不可用的池（实测）

| 请求 | 结果 |
| --- | --- |
| `20192` + `StockCode=中证通指数` | 不可用，返回 `-206021 查无相应的排名信息` |
| `20192` + `StockCode=上证指数` / `深证指数` / `沪深指数` | 板块名不存在 |
| `20191` + `StockCode=1101` 等股票市场号 | 可用，但返回**股票**而非指数 |

## 字段选择（`NeedCheck`）

`NeedCheck` 是字段白名单，码范围 **1–462**（与 A 股 / ETF 接口同一套码）。

| 规则 | 说明 |
| --- | --- |
| 列顺序 | 以响应回显的 `NEEDCHECK` 为准，**不按传参顺序** |
| 各池默认表头 | 不传 `NeedCheck` 时：`20191` 回 **9 列**，`20192` / `20198` 回 **20 列** |
| 传了 `NeedCheck` 后 | 四个池的表头**统一**为同一组列，可共用解析逻辑 |
| 多段字段 | 只有码 `33`–`42` 每码两段 `价格\|数量`，指数场景用不到 |
| 列索引 | 顶层 `*INDEX` 是默认列集的索引，自定义 `NeedCheck` 后不更新 |

## 响应

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `GRID0` | array\<string\> | `[0]` 为表头，`[1..]` 为指数（**无池自身行**） |
| `ACTION` | string | 动作号回显（`20191` / `20192` / `20198`） |
| `MAXCOUNT` | string | 该池成分总数（832 / 20 / 10 / 42） |
| `ERRORNO` | string | **非标准错误码**：正值随行数变化，**只有负值才是错误** |
| `NEEDCHECK` | string | 服务端实际生效的字段码，列顺序以它为准 |
| `HQDATE` / `SERVERDATETIME` | string | 行情日期 `yyyyMMdd` / 服务器时间 |
| `STOCKCODE` / `STOCKNAME` | — | `20198` 不返回；`20191` 即使返回也是市场号，不代表某只指数 |

### 记录结构

```
GRID0[0] = 名称|最新|幅度|...|代码|
GRID0[1] = 1.180能源|2617.02|2.28%|...|H50001|        ← 中证池，序号 1..832
GRID0[1] = 1.龙头家电|11150.85|0.98%|...|980028|      ← 国证池，序号从 1 重新开始
```

- 记录末尾带 `|`，切分后去掉最后一个空元素；
- 名称带 `1.` `2.` 序号前缀（**各池独立编号**，跨池会重复），客户端已剥离。

### 条目字段（默认 45 个，指数口径）

默认 45 个字段码与 A 股接口是**同一套**，字段码、`Values` key 与 `Row` 成员的对应关系见
[GetQuoteSnapshotCNAStock.md](GetQuoteSnapshotCNAStock.md) 的「条目字段」表。指数口径的差异：

| 码 | 字段 | 指数场景说明 |
| --- | --- | --- |
| `2` / `3` / `5` / `6` / `7` / `8` | `last` / `change_price` / `pre_close` / `open` / `high` / `low` | 单位是**点** |
| `9` | `avg_price` | 指数多为 `null` |
| `10` / `11` | `volume` / `amount` | **成分合计**成交量 / 成交额 |
| `14` / `15` | `bid_ask_ratio` / `bid_ask_diff` | 有值，语义为成分合计（委差可能异常大） |
| `32` | `code` | **内部码**：`1A0001` / `2A01` / `399006` / `899050` / `H50001` / `930622` |
| `48` / `49` | `outer_volume` / `inner_volume` | 指数多为 `null` |
| `92` / `208` / `227` / `228` / `233` / `306` | 各类估值字段 | **恒为 `null`** |
| `225` / `226` | `limit_up_price` / `limit_down_price` | **恒为 `null`** |
| `229` / `231` | `total_shares` / `float_shares` | **成分合计**总股本 / 流通盘 |
| `230` / `232` | `market_cap` / `float_market_cap` | **成分合计**总市值 / 流通市值 |
| `256` | `roe` | 实测 `0.0` |
| `342` | `change_pct_week` | 实测与 `298`（5 日）同值 |

### 示例

原始响应片段（中证池）：

```json
{"GRID0":[
 "名称|最新|幅度|涨跌|昨收|开盘|最高|最低|均价|总手|总额|现手|换手|振幅|委比|委差|量比|内盘|外盘|代码|",
 "1.180能源|2617.02|2.28%|58.33|2558.69|2560.11|2621.44|2556.32|--|597.46万手|89.23亿|--|0.20%|2.54%|--|--|1.02|--|--|H50001|"],
 "ACTION":"20191","MAXCOUNT":"832","ERRORNO":"833"}
```

合并后的上证指数（`1A0001`，节选）：

```json
{
  "Code": "1A0001",
  "Name": "上证指数",
  "Last": 3634.35,
  "ChangePrice": -21.48,
  "ChangeRatio": -0.59,
  "Volume": 472000000,
  "Amount": 770614000000,
  "TurnoverRate": 0.97,
  "NetInflow": -18053000000,
  "MarketCap": 77620000000000,
  "ChangePct5D": -0.41,
  "ChangePct20D": -2.2
}
```

板块不存在 / 不可用的响应：

```json
{"ERRORNO":"-206021","ERRORMESSAGE":"查无相应的排名信息!"}
```

## 业务码与异常

以响应体 `ERRORNO` 为准，**HTTP 状态码恒为 200**。

| `ERRORNO` | 含义 | 处理建议 |
| --- | --- | --- |
| 正值（如 `833`、`43`） | **正常**，数值随返回行数变化 | 按成功处理 |
| `-206021` | 查无相应的排名信息 | 板块名 / 市场号不存在（如 `中证通指数`） |
| `-206008` | 未找到匹配的证券代码 | `StockCode` 不合法 |
| `-989905` | 功能被服务端禁用 | 换接口或放弃该能力 |

传输层同前：偶发 HTTP 200 但响应体被截断，客户端内置 4 次退避重试。

客户端策略：**任一池失败即返回错误**（错误信息带池名），不会返回缺数据的「全量」结果。
需要容忍单池失败时，用 `IndexSnapshotRequest.Boards` 自行编排各池请求。

## 客户端用法

```go
client := xetapp.New(xetapp.Config{})
result, err := client.GetQuoteSnapshotCNAIndex(ctx, xetapp.IndexSnapshotRequest{
    // Boards 为空即默认 4 个池；ExtraFields 可追加字段码。
})
for _, row := range result.Rows {
    fmt.Println(row.Code, row.Name, row.Last, row.ChangeRatio)
}
// 各池的抓取与去重统计（排障用）
for _, pool := range result.Pools {
    fmt.Printf("%s: fetched=%d added=%d\n", pool.Name, pool.Fetched, pool.Added)
}
```

- `IndexSnapshotRequest.Boards` 可只取部分池（例如只拉中证 + 大盘）；
- `IndexSnapshotResult.Rows` 是去重后的并集，顺序为各池首次出现的顺序；
- `IndexSnapshotResult.Pools` 记录每个池的 `MaxCount` / `Fetched` / `Added` / `ErrorNo`。

## 实测记录

| 项 | 值 |
| --- | --- |
| 验证日期 | 2026-11 |
| 4 池总行数 | 904 |
| 去重后 | 879 只（重叠 25 只） |
| 耗时 | 约 3.8 秒（串行 4 次请求） |
| 数据核对 | 上证指数 `1A0001` 的 `last` / `change_ratio` / `amount` 与文档示例一致 |

## 相关实现

- `component/quote/citics/xetapp/snapshot_index.go`：`Client.GetQuoteSnapshotCNAIndex` 与 `IndexBoards`
- `component/quote/citics/xetapp/snapshot.go`：共用的请求组装、解析与字段转换
- `component/quote/citics/xetapp/snapshot_index_test.go`：池顺序、去重、统计与池失败的
  用例
