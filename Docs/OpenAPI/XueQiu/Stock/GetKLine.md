# GetKLine

雪球日线接口。一次请求返回全历史（最多 8000 根），`volume` 原生是**股**、`amount` 原生是
**元**，是唯一「成交量 + 成交额」都无需换算的来源。机器可读规格见
[Meta/OpenAPI/XueQiu/Stock/GetKLine.yaml](../../../../Meta/OpenAPI/XueQiu/Stock/GetKLine.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://stock.xueqiu.com` |
| 路径 | `/v5/stock/chart/kline.json` |
| operationId | `getKLine` |
| 代码入口 | `stock.Client.GetKLine`（`component/quote/xueqiu/stock` 包） |
| 鉴权 | **需要 cookie**：先访问 `https://xueqiu.com/` 预热会话 |
| 备注 | 列式结构（`column` + `item`）；`item` 为空通常表示 cookie 失效 |

要点：

- 首次调用会先预热会话拿 cookie；预热只做一次（互斥锁保护）；
- 返回的 `item` 为空时，本库会重建会话并重试一次，仍为空才报错；
- `count` 传负数表示「向前取 N 根」，8000 根覆盖 1999 年至今。

## 请求参数

### 查询参数

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `symbol` | 是 | string | 带市场前缀的大写代码 | `SZ000001` |
| `begin` | 是 | int64 | 毫秒时间戳，本库传当前时间 | `1794499200000` |
| `period` | 是 | string | 固定 `day` | `day` |
| `count` | 是 | int | 根数，负数表示向前取 | `-8000` |
| `indicator` | 是 | string | 固定 `kline` | `kline` |
| `type` | 是 | string | 固定 `normal`（不复权） | `normal` |

`symbol` 前缀规则：首位为 `6` → `SH`，其余 → `SZ`。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `user-agent` | 是 | 浏览器 UA |
| `referer` | 是 | 固定 `https://xueqiu.com/` |
| `cookie` | 是 | 预热首页后由 cookie jar 自动携带 |

### 请求体

`GET` 请求，无请求体。

## 签名规则

无签名、无 token 参数。鉴权完全依赖首页预热得到的 cookie；cookie 失效时表现为
返回空 `item`，本库会自动重建会话（最多重试一次）。

## 响应

### 响应头

| 响应头 | 说明 |
| --- | --- |
| `content-type` | `application/json` |

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `error_code` | int | `0` 表示成功；非 0 时 `error_description` 给原因 |
| `data.column` | array\<string\> | 列名数组，决定 `item` 每个下标的含义 |
| `data.item` | array\<array\> | 二维数组，每行是一根 K 线 |

### 响应字段（按列名）

| 列名 | `KLineBar` | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- | --- |
| `timestamp` | `Date` | 不出现 | `int64`，毫秒 | 按东八区换算 `yyyy-MM-dd` |
| `volume` | `Volume` | 不出现 | `int64`，**股** | 不换算 |
| `open` | `Open` | 不出现 | `float64` | 缺失回落 `close` |
| `high` | `High` | 不出现 | `float64` | 缺失回落 `close` |
| `low` | `Low` | 不出现 | `float64` | 缺失回落 `close` |
| `close` | `Close` | 不出现 | `float64` | `≤0` 的行丢弃 |
| `amount` | `Amount` | 不出现 | `float64`，**元** | 不换算 |

要点：

- `column` 顺序由服务端决定，必须按列名查下标；
- 单元格可能是 `null`，按缺失处理。

### 示例

```json
{
  "error_code": 0,
  "data": {
    "symbol": "SZ000001",
    "period": "day",
    "column": ["timestamp", "volume", "open", "high", "low", "close", "amount"],
    "item": [[1794499200000, 218942998, 11.76, 12.10, 11.70, 12.07, 2593456789]]
  }
}
```

## 业务码

| 现象 | 含义 | 处理建议 |
| --- | --- | --- |
| `error_code=0` + `item` 非空 | 正常 | 按 `column` 解析 |
| `error_code≠0` | 参数或登录态问题 | 读 `error_description` |
| `item=[]` | cookie 失效或代码无效 | 本库重建会话重试一次；仍空则报错 |

## 限制与配额

站点未公布官方配额，下表是「接口能力边界」与上游实测经验；未实测的项已明确标注。

| 维度 | 结论 | 依据 |
| --- | --- | --- |
| 证券范围 | 沪深（`SH` / `SZ`）；**不支持北交所（`BJ`）与 B 股**，卡片前缀规则同其他源 | 代码 `resolveSymbol` |
| 条数 | `count` 一次最多 `8000` 根（负数表示向前取） | 代码常量 |
| 历史区间 | 8000 根，上游结论为覆盖 1999 年至今 | 上游 `Core/History.py` |
| 最早数据 | 1999 年（上游结论，未逐标的验证） | 上游文档 |
| 限流 | 站点未公布配额；cookie 失效时返回空 `item`，本库重建会话重试一次 | 代码 + 上游实测 |
| 其他 | 需要首页预热拿 cookie，否则返回空数据 | 接口要求 |

## 客户端用法

```go
client := stock.New(stock.Config{})
result, err := client.GetKLine(ctx, stock.GetKLineRequest{Code: "000001"})
for _, bar := range result.Bars {
    fmt.Println(bar.Date, bar.Close, bar.Volume, bar.Amount)
}
```

## 相关实现

- `component/quote/xueqiu/stock/client.go`：配置、cookie jar、预热与失效重建、`APIError`
- `component/quote/xueqiu/stock/kline.go`：`GetKLine`、列式解析、日期换算
- `component/quote/xueqiu/stock/kline_test.go`：预热次数、失效重建、列序无关性
