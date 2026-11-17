# GetKLine

东财日线接口。本项目有两处用法，共用同一个端点：

1. **交易日历**：`GetTradingCalendar` 拉上证指数最近 260 个交易日，取日期序列当日历，
   避免在仓库里打包节假日表；
2. **个股日线备源**：`GetKLine` 直接拉个股日线，`f57` 给出**真实成交额**。

机器可读规格见
[Meta/OpenAPI/EastMoney/Push2/GetKLine.yaml](../../../../Meta/OpenAPI/EastMoney/Push2/GetKLine.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://push2his.eastmoney.com`（另有 `1.` / `82.` 两个镜像子域） |
| 路径 | `/api/qt/stock/kline/get` |
| operationId | `getKLine` |
| 代码入口 | `push2.Client.GetKLine` / `push2.Client.GetTradingCalendar`（`component/quote/eastmoney/push2` 包） |
| 鉴权 | 无（无需 cookie、无签名） |
| 备注 | 长区间请求易被限频；`f56` 是**手**，`f57` 是**元** |

## 请求参数

### 查询参数

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `secid` | 是 | string | `1.` 开头为沪市、`0.` 开头为深市 | `1.000001`、`0.000001` |
| `ut` | 否 | string | 站点公开固定参数（非用户凭据），默认 `fa5fd1943c7b386f172d6893dbfba10b` | 同左 |
| `fields1` | 是 | string | 外层字段，固定 `f1,f2,f3,f4,f5,f6` | 同左 |
| `fields2` | 是 | string | K 线字段，固定 `f51..f57` | `f51,f52,f53,f54,f55,f56,f57` |
| `klt` | 是 | int | `101` 为日线 | `101` |
| `fqt` | 是 | int | 复权方式，本库只用 `0`（不复权） | `0` |
| `beg` | 否 | string | 起始日期 `yyyyMMdd` | `20041201` |
| `end` | 否 | string | 结束日期 `yyyyMMdd` | `20500101` |
| `lmt` | 否 | int | 返回条数上限；日历用 `260`，个股用 `1000000` | `260` |

`GetKLineRequest` 里 **`SecID` 优先**：指数等「代码无法区分市场」的标的（如上证指数
`1.000001` 与平安银行 `0.000001` 同代码）必须显式传 `SecID`；`Code` 只在 `SecID`
为空时使用，市场按首位推断（`6` → 沪市，其余 → 深市）。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `user-agent` | 是 | 浏览器 UA |
| `referer` | 是 | 固定 `https://quote.eastmoney.com/` |

### 请求体

`GET` 请求，无请求体。

## 签名规则

无签名、无 token。`ut` 是站点前端的公开固定参数，不是用户凭据；个股日线请求携带它即可。

## 响应

### 响应头

| 响应头 | 说明 |
| --- | --- |
| `content-type` | `application/json` |

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `rc` | int | `0` 表示成功 |
| `data` | object / null | 业务数据；被限频或代码不存在时为 `null` |
| `data.code` | string | 证券代码 |
| `data.name` | string | 证券名称 |
| `data.klines` | array\<string\> | 每根 K 线一个字符串，逗号分隔 |

### 响应字段（`data.klines` 每行的位置）

| 位置 | 字段码 | `KLineBar` | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- | --- | --- |
| 0 | `f51` | `Date` | `"2026-11-13"` | 不出现 | `yyyy-MM-dd` |
| 1 | `f52` | `Open` | `"11.760"` | `float64`，元 | |
| 2 | `f53` | `Close` | `"12.070"` | `float64`，元 | `≤0` 的行丢弃 |
| 3 | `f54` | `High` | `"12.100"` | `float64`，元 | |
| 4 | `f55` | `Low` | `"11.700"` | `float64`，元 | |
| 5 | `f56` | `Volume` | `"2189429"` | `float64`，**手** | 本库**不**换算成股 |
| 6 | `f57` | `Amount` | `"2593456789.000"` | `float64`，**元** | 真实成交额 |

要点：

- 本库把 `Volume` 保留为**手**，与腾讯快照一致；新浪/雪球是「股」，跨源对比需自行换算；
- `data.klines=[]` 表示区间内无数据，返回空 `Bars` 而不是错误；
- `GetTradingCalendar` 只取每行第 0 段日期，升序返回。

### 示例

```json
{
  "rc": 0,
  "data": {
    "code": "000001",
    "name": "平安银行",
    "klines": [
      "2004-12-31,6.580,6.590,6.660,6.580,17521,11568524.157",
      "2026-11-13,11.760,12.070,12.100,11.700,2189429,2593456789.000"
    ]
  }
}
```

## 业务码

| 现象 | 含义 | 处理建议 |
| --- | --- | --- |
| HTTP `200` + `data.klines` 非空 | 正常 | 继续处理 |
| HTTP `200` + `data=null` | 限频、代码不存在或被拦 | 换镜像域名退避重试 |
| HTTP `502` / 连接重置 | 镜像故障或限流 | 换域名重试 |
| HTTP `200` + `klines=[]` | 区间内无数据 | 记录后跳过 |

## 限制与配额

站点未公布官方配额，下表是「接口能力边界」与上游实测经验；未实测的项已明确标注。

| 维度 | 结论 | 依据 |
| --- | --- | --- |
| 证券范围 | 沪市（`1.` 前缀）、深市与北交所（`0.` 前缀，北交所在东财同属 0 市场） | 东财 `secid` 规则 |
| 条数 | `lmt` 控制；本项目个股用 `1000000`、日历用 `260` | 代码常量 |
| 历史区间 | `beg` ~ `end` 任意区间；**区间越长越容易触发限流** | 上游实测 |
| 最早数据 | **未实测**。上游只验证过 2004 年起的样本 | — |
| 限流 | 站点未公布配额；3 个镜像子域轮换，长区间请求失败后由调用方决定是否重试 | 代码常量 |
| 其他 | `ut` 是站点前端的公开固定值，不是用户凭据 | 上游脚本 |

## 客户端用法

```go
client := push2.New(push2.Config{})

// 个股日线（不复权）
result, err := client.GetKLine(ctx, push2.GetKLineRequest{
    Code:  "000001",
    Begin: "20240101",
    End:   "20261130",
})

// 交易日历（内部用上证指数 1.000001）
days, err := client.GetTradingCalendar(ctx, 260)
```

## 相关实现

- `component/quote/eastmoney/push2/client.go`：配置、镜像轮换、`APIError`
- `component/quote/eastmoney/push2/kline.go`：`GetKLine`、`GetTradingCalendar`、行解析
- `component/quote/eastmoney/push2/quotelist_test.go`：日线参数、`secid` 推断与日历用例
