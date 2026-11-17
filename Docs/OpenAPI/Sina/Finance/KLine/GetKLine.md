# GetKLine

新浪财经日线接口，返回长历史日线（一次最多数千根）。本项目用它取基准指数的长历史收盘
（如国证 2000 `sz399303`）。机器可读规格见
[Meta/OpenAPI/Sina/Finance/KLine/GetKLine.yaml](../../../../../Meta/OpenAPI/Sina/Finance/KLine/GetKLine.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://money.finance.sina.com.cn` |
| 路径 | `/quotes_service/api/json_v2.php/CN_MarketData.getKLineData` |
| operationId | `getKLine` |
| 代码入口 | `kline.Client.GetKLine`（`component/quote/sina/finance/kline` 包） |
| 鉴权 | 无（无需 cookie、无签名） |
| 备注 | **没有成交额字段**；`volume` 单位是**股** |

要点：

- 该接口只返回 OHLC 与成交量，需要成交额时必须改用雪球或东财日线；
- `volume` 原生单位是**股**（东财/腾讯是「手」），混用会差 100 倍；
- 代码必须带 `sh` / `sz` 前缀，指数与个股同一套规则。

## 请求参数

### 查询参数

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `symbol` | 是 | string | 带市场前缀的代码 | `sz399303` |
| `scale` | 是 | int | 周期（分钟数），`240` 为日线 | `240` |
| `ma` | 是 | string | 是否附带均线，固定 `no` | `no` |
| `datalen` | 是 | int | 返回根数，默认 `4000`（约 16 年） | `4000` |

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `user-agent` | 是 | 浏览器 UA |
| `referer` | 是 | 固定 `https://finance.sina.com.cn/` |

### 请求体

`GET` 请求，无请求体。

## 签名规则

无签名、无 token。仅需 `Referer` 与浏览器 UA。

## 响应

### 响应头

| 响应头 | 说明 |
| --- | --- |
| `content-type` | `application/json`（可能是 `text/plain` 承载 JSON） |

### 外层结构

顶层是数组，每个元素是一根 K 线；按对象键取值，不依赖字段顺序。
**响应不是数组时本库返回 `*APIError`**（站点限频会返回 `null`）。

### 响应字段

| 字段 | `KLineBar` | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- | --- |
| `day` | `Date` | `"2026-11-13"` | 不出现 | `yyyy-MM-dd` |
| `open` | `Open` | `"11.760"` | `float64` | 元 / 点 |
| `high` | `High` | `"12.100"` | `float64` | |
| `low` | `Low` | `"11.700"` | `float64` | |
| `close` | `Close` | `"12.070"` | `float64` | `≤0` 的行丢弃 |
| `volume` | `Volume` | `"218942998"` | `float64`，**股** | 不换算 |

### 示例

```json
[
  {"day": "2026-11-13", "open": "11.760", "high": "12.100",
   "low": "11.700", "close": "12.070", "volume": "218942998"}
]
```

## 业务码

| 现象 | 含义 | 处理建议 |
| --- | --- | --- |
| HTTP `200` + 数组非空 | 正常 | 逐条解析 |
| HTTP `200` + `[]` | 代码无效或区间无数据 | 返回空 `Bars` |
| HTTP `200` + `null` / 错误页 | 限频或被拦 | 返回 `*APIError`，调用方可重试或换源 |

## 限制与配额

站点未公布官方配额，下表是「接口能力边界」与上游实测经验；未实测的项已明确标注。

| 维度 | 结论 | 依据 |
| --- | --- | --- |
| 证券范围 | 沪深（`sh` / `sz`）；**不支持北交所（`bj`）与 B 股**。本库按首位推断前缀：`6` → 沪、`4`/`8` → 北交所、其余 → 深 | 代码 `resolveSymbol` |
| 条数 | `datalen` 默认 `4000`；上游实测取 `6000` 可用，**上限未验证** | 上游 `fetch_data.py` |
| 历史区间 | 由 `datalen` 决定，**没有起止日期参数**；4000 根约 16 年、6000 根约 24 年 | 上游实测 |
| 最早数据 | **未实测**；实测样本（平安银行）最早可到 2001-06-26 | 上游样本数据 |
| 限流 | 站点未公布配额；限频时返回 `null` 或错误页，本库据此返回 `*APIError` | 上游实测 |
| 其他 | 不返回成交额 | 接口字段 |

## 客户端用法

```go
client := kline.New(kline.Config{})
result, err := client.GetKLine(ctx, kline.GetKLineRequest{
    Code:    "399303",
    DataLen: 4000,
})
for _, bar := range result.Bars {
    fmt.Println(bar.Date, bar.Close)
}
```

## 相关实现

- `component/quote/sina/finance/kline/client.go`：配置、`APIError`
- `component/quote/sina/finance/kline/kline.go`：`GetKLine`、行解析与数组校验
- `component/quote/sina/finance/kline/kline_test.go`：参数、剔除规则、非数组响应
