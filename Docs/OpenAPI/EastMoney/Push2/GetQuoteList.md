# GetQuoteList

东财行情列表接口，返回全市场（或指定板块）的**当日快照**，可按总市值或上市日期升降序翻页。
本项目用它做「按总市值升序扫描、边翻边筛选」的无状态选股路径，也是补次新股上市日期的来源。
机器可读规格见
[Meta/OpenAPI/EastMoney/Push2/GetQuoteList.yaml](../../../../Meta/OpenAPI/EastMoney/Push2/GetQuoteList.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://push2.eastmoney.com`（另有 `82.` / `1.` / `7.` 三个镜像子域） |
| 路径 | `/api/qt/clist/get` |
| operationId | `getQuoteList` |
| 代码入口 | `push2.Client.GetQuoteList`（`component/quote/eastmoney/push2` 包） |
| 鉴权 | 无（无需 cookie、无签名） |
| 备注 | 分页从 `1` 开始；镜像域名按配置顺序轮换；`data=null` 视为被限频并换镜像 |

实测要点：

- 主域 `push2.eastmoney.com` 经常 502，四个镜像域名按顺序轮换，单页失败只影响该页；
- 连续翻页会被限频，本项目在 UI 层按需留间隔（库本身不 sleep，便于调用方控制节奏）；
- 无行情（退市、长期停牌）的记录会被剔除，因此返回条数可能少于 `PageSize`。

## 请求参数

### 查询参数

参数顺序无关（该接口不签名）。

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `pn` | 是 | int | 页码，从 `1` 开始 | `1` |
| `pz` | 是 | int | 每页条数 | `100` |
| `po` | 是 | int | 排序方向：`0` 升序、`1` 降序 | `0` |
| `fid` | 是 | string | 排序字段：`f20` 总市值、`f26` 上市日期 | `f20` |
| `fs` | 是 | string | 市场过滤串，见下表 | `m:0+t:6,m:1+t:2` |
| `fields` | 是 | string | 需要的字段码，逗号分隔 | `f2,f5,f6,...` |
| `np` | 是 | int | 固定 `1`，原样透传 | `1` |
| `fltt` | 是 | int | 固定 `2`，原样透传 | `2` |
| `invt` | 是 | int | 固定 `2`，原样透传 | `2` |

`fs` 常量（`push2.UniverseMainBoard` / `push2.UniverseAllA`）：

| 常量 | 取值 | 含义 |
| --- | --- | --- |
| `UniverseMainBoard` | `m:0+t:6,m:1+t:2` | 沪深主板（深 `t:6`、沪 `t:2`），不含创业板、科创板、北交所 |
| `UniverseAllA` | `m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23` | 全部 A 股，含创业板 `t:80` 与科创板 `t:23` |

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `user-agent` | 是 | 浏览器 UA |
| `referer` | 是 | 固定 `https://quote.eastmoney.com/` |

### 请求体

`GET` 请求，无请求体。

## 签名规则

无签名、无 token。站点仅按 `Referer` 与 `User-Agent` 做基本来源校验；
`np` / `fltt` / `invt` 是站点前端的固定参数，原样透传即可。

## 响应

### 响应头

| 响应头 | 说明 |
| --- | --- |
| `content-type` | `application/json`（部分镜像返回 `text/plain`，内容仍是 JSON） |

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `rc` | int | 返回码，`0` 表示成功 |
| `data` | object / null | 业务数据；被限频时为 `null` |
| `data.total` | int | 符合条件的总条数 |
| `data.diff` | array | 当前页记录 |

### 响应字段

`QuoteListItem` 的每个成员都有对应的字段码，原始文本保留在 `Raw`（`map[string]string`）。

| 码 | `QuoteListItem` | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- | --- |
| `f2` | `Last` | `"11.76"` | `float64`，元 | 缺失为 `"-"`，此时整行被剔除 |
| `f5` | `Volume` | `"2189429"` | `float64`，**手** | 折算成股需 × 100 |
| `f6` | `Amount` | `"2593456789"` | `float64`，元 | 真实成交额 |
| `f12` | `Code` | `"000001"` | 不出现 | 六位字符串 |
| `f13` | `Market` | `"0"` / `"1"` | `int` | `0` 深市、`1` 沪市 |
| `f14` | `Name` | `"平安银行"` | 不出现 | 可能含 `ST` / `退` |
| `f15` | `High` | `"12.10"` | `float64`，元 | |
| `f16` | `Low` | `"11.70"` | `float64`，元 | |
| `f17` | `Open` | `"11.76"` | `float64`，元 | |
| `f20` | `TotalMV` | `"230987654321"` | `float64`，**元** | 排序依据 |
| `f21` | `FloatMV` | `"220000000000"` | `float64`，**元** | |
| `f26` | `ListDate` | `"1991-04-03"` | 不出现 | 缺失为空串或 `"-"` |
| `f124` | `Timestamp` | `"1763006400"` | `int64`，秒 | |

要点：

- 市值单位是**元**，与腾讯快照的「亿元」差 1 亿倍；
- `"-"` 与空串统一按缺失处理（`parseNum` 返回 0）；
- `f2` 或 `f20` 非正的行视为无行情，直接丢弃。

### 示例

```json
{
  "rc": 0,
  "data": {
    "total": 3210,
    "diff": [
      {"f2": 11.76, "f5": 2189429, "f6": 2593456789, "f12": "000001",
       "f13": 0, "f14": "平安银行", "f15": 12.10, "f16": 11.70,
       "f17": 11.76, "f20": 230987654321, "f21": 220000000000,
       "f26": "1991-04-03", "f124": 1763006400}
    ]
  }
}
```

## 业务码

该接口没有业务码字段，以 HTTP 状态与 `data` 是否存在判断：

| 现象 | 含义 | 处理建议 |
| --- | --- | --- |
| HTTP `200` + `data.diff` 非空 | 正常 | 继续处理 |
| HTTP `200` + `data=null` | 限频或被拦 | 换镜像域名退避重试 |
| HTTP `502` | 主域故障 | 立即换下一个镜像子域 |
| HTTP `200` + `data.diff=[]` | 页码越界 | 停止翻页 |

## 限制与配额

站点未公布官方配额，下表是「接口能力边界」与上游实测经验；未实测的项已明确标注。

| 维度 | 结论 | 依据 |
| --- | --- | --- |
| 证券范围 | 沪深主板（`m:0+t:6,m:1+t:2`）或全部 A 股（含创业板 `t:80`、科创板 `t:23`）；**不含北交所** | `fs` 常量 |
| 分页 | `pn` 从 `1` 开始；`pz` 本项目固定 `100`，**更大值未实测** | 代码常量 |
| 历史区间 | 不适用（只返回当日快照） | — |
| 最早数据 | 不适用 | — |
| 限流 | 站点未公布配额。上游实测连续翻页会被限频（表现为 `data=null`）；CNAMCSI 的做法是每轮间隔 1 秒、并发 2、连续两轮全失败即放弃。**本库不主动 sleep**，节奏由调用方控制 | 上游 `Core/Sources.py` |
| 其他 | 4 个镜像子域轮换；单页失败只影响该页 | 代码常量 |

## 客户端用法

```go
client := push2.New(push2.Config{})
page, err := client.GetQuoteList(ctx, push2.GetQuoteListRequest{
    Page:      1,
    PageSize:  100,
    SortField: push2.SortByTotalMV, // 按总市值升序
    Universe:  push2.UniverseMainBoard,
})
for _, item := range page.Items {
    fmt.Println(item.Code, item.Name, item.TotalMV)
}
```

## 相关实现

- `component/quote/eastmoney/push2/client.go`：配置、镜像轮换、`APIError`
- `component/quote/eastmoney/push2/quotelist.go`：`GetQuoteList` 与行解析
- `component/quote/eastmoney/push2/quotelist_test.go`：请求装配、镜像降级、剔除规则
