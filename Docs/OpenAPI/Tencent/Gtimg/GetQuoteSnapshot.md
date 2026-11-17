# GetQuoteSnapshot

腾讯批量行情快照接口。一次请求可带上百个代码，返回最新价、今开、成交量、成交额、
**总市值与流通市值**；本项目用它作为每日成分计算的主路径，也是总市值/流通市值的锚点。
机器可读规格见
[Meta/OpenAPI/Tencent/Gtimg/GetQuoteSnapshot.yaml](../../../../Meta/OpenAPI/Tencent/Gtimg/GetQuoteSnapshot.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://qt.gtimg.cn` |
| 路径 | `/q=<symbols>` |
| operationId | `getQuoteSnapshot` |
| 代码入口 | `gtimg.Client.GetQuoteSnapshot`（`component/quote/tencent/gtimg` 包） |
| 鉴权 | 无（无需 cookie、无签名） |
| 备注 | 单批最多约 900 只，超过触发 HTTP 414；响应是 **GBK 文本**，不是 JSON |

实测要点：

- 900 只（URL 约 8.1 KB）可用，950 只触发 `414 Request-URI Too Long`；
- 本库自动分批（默认 900），某批失败时**二分降级**到单批不超过 100 只；
- 必须带 `referer: https://finance.qq.com/`。

## 请求参数

### 查询参数

代码以路径形式出现（`/q=` 后接代码列表）。

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `symbols` | 是 | string | 带市场前缀的代码，逗号分隔 | `sz000001,sh600000` |

代码前缀规则（`gtimg.symbolOf`）：首位为 `6` → `sh`，其余 → `sz`。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `user-agent` | 是 | 浏览器 UA |
| `referer` | 是 | 固定 `https://finance.qq.com/` |

### 请求体

`GET` 请求，无请求体。

## 签名规则

无签名、无 token。仅需 `Referer` 与浏览器 UA。响应按 **GBK** 解码，按 UTF-8 解码会乱码。

## 响应

### 响应头

| 响应头 | 说明 |
| --- | --- |
| `content-type` | `text/html; charset=GBK` |

### 外层结构

纯文本，每只证券一行赋值语句，行间以 `;` 分隔：

```text
v_sz000001="51~平安银行~000001~12.07~11.76~11.76~2189429~...~20261113150000~...~259345.67~...~2309.88~2309.88~...";
```

按 `~` 拆分得到字段数组；**少于 46 段或代码为空的行直接丢弃**。

### 响应字段

| 下标 | `QuoteItem` | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- | --- |
| `1` | `Name` | `"平安银行"` | 不出现 | GBK 解码后的中文 |
| `2` | `Code` | `"000001"` | 不出现 | 六位字符串 |
| `3` | `Last` | `"12.07"` | `float64`，元 | `≤0` 视为无行情，整行丢弃 |
| `5` | `Open` | `"11.76"` | `float64`，元 | |
| `6` | `Volume` | `"2189429"` | `float64`，**手** | 折算成股需 × 100 |
| `30` | `Timestamp` | `"20261113150000"` | `string` | `yyyyMMddHHmmss` |
| `37` | `Amount` | `"259345.67"` | `float64`，**万元** | 乘 `1e4` 得元 |
| `44` | `FloatMV` | `"2309.88"` | `float64`，**亿元** | 乘 `1e8` 得元 |
| `45` | `TotalMV` | `"2309.88"` | `float64`，**亿元** | 乘 `1e8` 得元，`≤0` 视为无行情 |

要点：

- 与东财不同，腾讯的市值单位是**亿元**；成交额是**万元**；
- 本库只使用前 46 段，更后面的段未使用也未文档化；
- 该接口**不返回上市日期与股本**。

### 示例

```text
v_sz000001="51~平安银行~000001~12.07~11.76~11.76~2189429~...~20261113150000~...~259345.67~...~2309.88~2309.88~...";
```

解析后的 `QuoteItem`：

```json
{
  "Code": "000001", "Name": "平安银行", "Last": 12.07, "Open": 11.76,
  "Volume": 2189429, "Amount": 2593456700,
  "TotalMV": 230988000000, "FloatMV": 230988000000,
  "Timestamp": "20261113150000"
}
```

## 业务码

该接口没有业务码字段：

| 现象 | 含义 | 处理建议 |
| --- | --- | --- |
| HTTP `200` + 正常赋值语句 | 正常 | 按 `~` 拆分 |
| HTTP `414` | 批量过大（URL 太长） | 二分代码列表，降到 100 只以内 |
| 某代码无对应行 | 代码不存在或长期停牌 | 跳过该代码 |
| 取值 `"-"` 或空串 | 该字段无数据 | 按缺失处理 |

## 限制与配额

站点未公布官方配额，下表是「接口能力边界」与实测经验；未实测的项已明确标注。

| 维度 | 结论 | 依据 |
| --- | --- | --- |
| 证券范围 | 沪深（`sh` / `sz`）；**不支持北交所（`bj`）与 B 股**。本库按首位推断前缀：`6` → 沪、`4`/`8` → 北交所、其余 → 深；`920` 段（北交所新代码）需显式传 `Symbol` | 代码 `symbolOf` |
| 批量 | 900 只/次（URL 约 8.1 KB），950 只触发 `414`；本库自动二分到单批不超过 100 只 | 实测 + 代码 |
| 历史区间 | 不适用（只返回当日快照） | — |
| 最早数据 | 不适用 | — |
| 限流 | 站点未公布配额，本项目未观测到明确的 QPS 限制 | — |
| 其他 | 响应是 GBK 编码，按 UTF-8 解码会乱码 | 代码 |

## 客户端用法

```go
client := gtimg.New(gtimg.Config{})
snapshot, err := client.GetQuoteSnapshot(ctx, gtimg.GetQuoteSnapshotRequest{
    Codes: []string{"000001", "600000"},
})
for _, item := range snapshot.Items {
    fmt.Println(item.Code, item.Name, item.TotalMV)
}
```

## 相关实现

- `component/quote/tencent/gtimg/client.go`：配置、GBK 解码接入、`APIError`
- `component/quote/tencent/gtimg/snapshot.go`：分批、二分降级、字段解析
- `component/quote/tencent/gtimg/snapshot_test.go`：请求装配、414 降级、剔除规则
