# GetQuoteSnapshotCNAETF

一次请求拿到沪深两市全量 ETF 行情快照（`action=20192` + 板块中文名 `沪深ETF基金`），
返回「表头 + 竖线分隔记录」的行情表；默认 45 个字段，可扩展到 462 个。
机器可读规格见
[Meta/OpenAPI/Citics/XetApp/GetQuoteSnapshotCNAETF.yaml](../../../../Meta/OpenAPI/Citics/XetApp/GetQuoteSnapshotCNAETF.yaml)。

姊妹接口：[A 股全市场](GetQuoteSnapshotCNAStock.md) / [全量指数](GetQuoteSnapshotCNAIndex.md)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `POST`（也接受 `GET` + query，参数相同） |
| 站点 | 上游行情网关（地址由 `Config.GatewayURL` 指定） |
| 路径 | `/reqxml` |
| 站点动作 | `action=20192`（按**板块中文名**取成分行情；`20703` 等价） |
| operationId | `getQuoteSnapshotCNAETF` |
| 代码入口 | `xetapp.Client.GetQuoteSnapshotCNAETF`（`component/quote/citics/xetapp` 包） |
| 鉴权 | 无（无需登录态、无需 cookie、无签名） |
| 备注 | 一次请求即可拿全量 1727 只；**无「池自身」行**（与 `20199` 不同）；`MaxCount` 必须显式传 |

实测覆盖度（2026-11-13）：

| 代码前缀 | 只数 | 市场 |
| --- | --- | --- |
| `15` | 779 | 深市 |
| `51` / `52` / `53` / `55` | 449 / 54 / 14 / 12 | 沪市 |
| `56` / `58` | 268 / 151 | 沪市 |
| 合计 | **1727** | 沪深两市，**无北交所 ETF** |

## 请求参数

### 表单参数

参数顺序无关；可选参数缺省时整键省略，不要传空串。

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `action` | 是 | int | 站点动作号，本接口固定 `20192` | `20192` |
| `ReqlinkType` | 是 | int | 行情通道固定 `0`；传 `2` 会走业务网关并报错 | `0` |
| `StockCode` | 是 | string | **板块中文名**，全量 ETF 用 `沪深ETF基金` | `沪深ETF基金` |
| `MaxCount` | 是 | int | 返回行数上限。**不传只回 1 行**；超过成分数按实际返回 | `60000` |
| `StartPos` | 否 | int | 分页起点，从 `1` 开始，是**成分序号**而非行号 | `1` |
| `NeedCheck` | 否 | string | 字段白名单，竖线分隔的字段码（`1`–`462`），末尾同样带竖线 | `1\|2\|4\|84\|347\|` |

`StockCode` 可用性（实测）：

| 取值 | 结果 |
| --- | --- |
| `沪深ETF基金` | 可用，`MAXCOUNT=1727`，全量沪深 ETF |
| `沪市ETF基金` / `深市ETF基金` / `沪市ETF` / `深市ETF` | 不可用，返回 `-206021 查无相应的排名信息` |
| `科创板` / `创业板_注册制` 等 | 可用，但返回的是该板块**股票**而非 ETF，不要误用 |

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `content-type` | 是 | `POST` 用 `application/x-www-form-urlencoded` |
| `user-agent` | 否 | 浏览器 UA，站点不校验 |

`StockCode` 含中文，需按 UTF-8 做表单编码（Go 的 `url.Values.Encode` 已处理）。

### 请求体

```
action=20192&ReqlinkType=0&StockCode=沪深ETF基金&MaxCount=60000&NeedCheck=1|2|4|10|11|84|347|
```

## 字段选择（`NeedCheck`）

规则与 A 股接口完全一致（同一套字段码 `1`–`462`），列顺序以响应回显的 `NEEDCHECK`
为准，不按传参顺序；缺省时返回该接口默认的 20 列。

| 规则 | 说明 |
| --- | --- |
| 多段字段 | 只有码 `33`–`42`（卖五…买五）每码两段 `价格\|数量`，其余均为单段 |
| 五档推荐 | 用 `93`–`102`（价格，元）+ `180`–`189`（数量，手），单段可靠 |
| 列索引 | 顶层 `*INDEX` 是默认 20 列的索引，传自定义 `NeedCheck` 后不更新 |

Go 客户端默认取 45 个字段（`xetapp.DefaultFields`），ETF 专属指标用
`ETFSnapshotRequest.ExtraFields` 追加。

## 响应

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `GRID0` | array\<string\> | `[0]` 为表头；**`[1]` 即第一只 ETF**（无池自身行） |
| `ACTION` | string | 动作号回显 `"20192"` |
| `MAXCOUNT` | string | 成分总数（= ETF 只数 1727） |
| `ERRORNO` | string | **非标准错误码**：正值随行数变化，**只有负值才是错误** |
| `NEEDCHECK` | string | 服务端实际生效的字段码，列顺序以它为准 |
| `HQDATE` / `SERVERDATETIME` | string | 行情日期 `yyyyMMdd` / 服务器时间 |
| `STOCKCODE` / `STOCKNAME` | — | 本接口**不返回**（`20199` 才有） |
| `VOLUMEINFO` | string | 行情源标识 |
| `ERRORMESSAGE` | string | 参数错误时的中文提示，仅错误时出现 |

### 记录结构

```
GRID0[0] = 名称|最新|幅度|...|代码|                       ← 表头
GRID0[1] = 1.科创200ETF工银|1.194|4.01%|...|589200|      ← 第一只即成分，名称带序号前缀
GRID0[2] = 2.标普油气ETF富国|1.338|3.08%|...|513350|
```

- 记录末尾同样带 `|`，切分后要去掉最后一个空元素；
- `名称` 带 `1.` `2.` 递增序号前缀（板块内排名），需剥离。

### 条目字段（默认 45 个）

默认 45 个字段码与 A 股接口是**同一套**，字段码、`Values` key 与 `Row` 成员的对应关系见
[GetQuoteSnapshotCNAStock.md](GetQuoteSnapshotCNAStock.md) 的「条目字段」表。ETF 口径的差异：

| 码 | 字段 | 差异 |
| --- | --- | --- |
| `92` | `pe` | ETF 恒为 `null` |
| `208` | `pb` | ETF 恒为 `null` |
| `227` | `dynamic_pe` | ETF 恒为 `null` |
| `228` | `bvps` | ETF 恒为 `null` |
| `233` | `eps` | ETF 恒为 `0` |
| `256` | `roe` | ETF 恒为 `0` |
| `306` | `pe_ttm` | ETF 恒为 `null` |
| `335` | `dividend_yield` | ETF 恒为 `null` |
| `408` | `dividend` | ETF 恒为 `null` |
| `453` | `dividend_yield_ttm` | ETF 恒为 `null` |
| `229` | `total_shares` | **总份额**（服务端列名仍为「总股本」） |
| `231` | `float_shares` | **流通份额**（服务端列名仍为「流通盘」） |
| `225` / `226` | `limit_up_price` / `limit_down_price` | ETF 为 ±10% |
| `342` | `change_pct_week` | 实测与 `298`（5 日）同值 |

### ETF 专属扩展字段

这些码不在默认 45 个字段内，需要用 `ExtraFields` 追加才会返回：

| 码 | `Values` 中的 key | Go 字段 | 说明 | `510300` 实测 |
| --- | --- | --- | --- | --- |
| `19` | `unit_net_value` | `UnitNetValue` | 单位净值（T-1） | 4.092 |
| `20` | `acc_net_value` | `AccNetValue` | 累计净值 | 1.844 |
| `21` | `growth_value` | — | 净值增长值 | `null` |
| `22` | `growth_rate` | — | 净值增长率 | `null` |
| `23` | `date` | `Date` | 净值日期 | `20261113` |
| `24` | `growth_date` | — | 增长值日期 | `null` |
| `84` | `fund_nav` | `FundNav` | 基金净值（盘中实时参考净值，约等于 IOPV） | 4.0927 |
| `326` | `fund_scale` | `FundScale` | 基金规模（净资产） | 1.87481 万亿 |
| `347` | `discount_rate` | `DiscountRate` | 折价率（负=折价） | −0.24 |
| `364` / `365` | `etf_class1` / `etf_class2` | `ETFClass1` / `ETFClass2` | ETF 分类 1 / 2 | `null` |
| `387` / `388` | `track_index_code` / `track_index_name` | `TrackIndexCode` / `TrackIndexName` | 跟踪指数代码 / 名称 | `null`（该标的未下发） |

规模、净值、折价率都是直接字段，不需要自己用 T-1 净值反推。

### 示例

原始响应片段：

```json
{"GRID0":[
 "名称|最新|幅度|涨跌|昨收|开盘|最高|最低|均价|总手|总额|现手|换手|振幅|委比|委差|量比|内盘|外盘|代码|",
 "1.科创200ETF工银|1.194|4.01%|0.046|1.148|1.161|1.201|1.101|1.143|29925手|342.10万|1手|7.39%|8.71%|13.76%|2345.00手|0.61|1.33万|1.68万|589200|"],
 "ACTION":"20192","MAXCOUNT":"1727","ERRORNO":"1728",
 "NEEDCHECK":"1|2|4|3|5|6|7|8|9|10|11|12|13|16|14|15|50|49|48|32|"}
```

解析后的 `Row`（`ExtraFields: []int{19, 20, 23, 84, 326, 347}`，节选）：

```json
{
  "Code": "510300",
  "Name": "沪深300ETF华泰柏瑞",
  "Last": 4.083,
  "ChangeRatio": -0.75,
  "Volume": 8510500,
  "Amount": 3489000000,
  "TotalShares": 45818000000,
  "MarketCap": 187074000000,
  "Date": "20261113",
  "UnitNetValue": 4.092,
  "AccNetValue": 1.844,
  "FundNav": 4.0927,
  "FundScale": 187481000000,
  "DiscountRate": -0.24
}
```

板块名不存在的响应：

```json
{"ERRORNO":"-206021","ERRORMESSAGE":"查无相应的排名信息!"}
```

## 业务码与异常

以响应体 `ERRORNO` 为准，**HTTP 状态码恒为 200**。

| `ERRORNO` | 含义 | 处理建议 |
| --- | --- | --- |
| 正值（如 `1728`） | **正常**，数值随返回行数变化 | 按成功处理 |
| `-206021` | 查无相应的排名信息 | 板块名不存在（如 `沪市ETF基金`），改用 `沪深ETF基金` |
| `-206008` | 未找到匹配的证券代码 | `StockCode` 不是合法板块名 |
| `-989905` | 功能被服务端禁用 | 换接口或放弃该能力 |
| 其它负值 | 业务失败 | 读 `ERRORMESSAGE` |

传输层同 `20199`：全量请求**偶发**返回 HTTP 200 但响应体被截断，客户端内置 4 次退避重试。

## 客户端用法

```go
client := xetapp.New(xetapp.Config{})
result, err := client.GetQuoteSnapshotCNAETF(ctx, xetapp.ETFSnapshotRequest{
    // Board 为空即默认「沪深ETF基金」。
    ExtraFields: []int{84, 326, 347}, // 基金净值 / 基金规模 / 折价率
})
for _, row := range result.Rows {
    fmt.Println(row.Code, row.Name, row.Last, row.FundNav, row.DiscountRate)
}
```

## 相关实现

- `component/quote/citics/xetapp/snapshot_etf.go`：`Client.GetQuoteSnapshotCNAETF`
- `component/quote/citics/xetapp/snapshot.go`：共用的请求组装、解析与字段转换
- `component/quote/citics/xetapp/snapshot_etf_test.go`：表单参数、字段扩展与错误分类用例
