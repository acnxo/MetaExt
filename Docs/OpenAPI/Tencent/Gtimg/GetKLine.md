# GetKLine

腾讯日线接口，支持不复权与前复权。本项目保留该实现作为「指数收盘价」的备用取值分支。
机器可读规格见
[Meta/OpenAPI/Tencent/Gtimg/GetKLine.yaml](../../../../Meta/OpenAPI/Tencent/Gtimg/GetKLine.yaml)。

> **调用状态**：库已实现并测试，但本项目当前没有默认调用点；需要时直接调用即可。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://web.ifzq.gtimg.cn` |
| 路径 | `/appstock/app/fqkline/get` |
| operationId | `getKLine` |
| 代码入口 | `gtimg.Client.GetKLine`（`component/quote/tencent/gtimg` 包） |
| 鉴权 | 无（无需 cookie、无签名） |
| 备注 | 不返回成交额；成交量单位是**手** |

## 请求参数

### 查询参数

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `param` | 是 | string | 6 段逗号拼接：`代码,周期,起,止,根数,复权` | 见下 |

`param` 各段：

| 段 | 含义 | 本项目取值 |
| --- | --- | --- |
| 1 | 带前缀代码 | `sz000001` |
| 2 | 周期 | `day` |
| 3 | 起始日期 | `yyyy-MM-dd` |
| 4 | 结束日期 | `yyyy-MM-dd`（含当日） |
| 5 | 根数 | 默认 `1000` |
| 6 | 复权 | `qfq` 或留空（不复权） |

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `user-agent` | 是 | 浏览器 UA |
| `referer` | 是 | 固定 `https://finance.qq.com/` |

### 请求体

`GET` 请求，无请求体。

## 签名规则

无签名、无 token。仅需 `Referer` 与浏览器 UA。

## 响应

### 响应头

| 响应头 | 说明 |
| --- | --- |
| `content-type` | `application/json` |

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `code` | int | `0` 表示成功 |
| `msg` | string | 提示信息 |
| `data` | object | 以请求的带前缀代码为键 |
| `data.<代码>.qfqday` | array | 前复权数据（请求 `qfq` 时出现） |
| `data.<代码>.day` | array | 不复权数据（未请求复权时出现） |

### 响应字段（每行为数组）

| 位置 | `KLineBar` | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- | --- |
| 0 | `Date` | `"2026-11-13"` | 不出现 | `yyyy-MM-dd` |
| 1 | `Open` | `"11.76"` | `float64` | 股票为元、指数为点 |
| 2 | `Close` | `"12.07"` | `float64` | `≤0` 的行丢弃 |
| 3 | `High` | `"12.10"` | `float64` | |
| 4 | `Low` | `"11.70"` | `float64` | |
| 5 | `Volume` | `"2189429"` | `float64`，**手** | 本库不换算 |

要点：

- 行长度不足 6 段时直接跳过；
- 该接口**不返回成交额**，需要成交额时改用雪球或东财日线。

### 示例

```json
{
  "code": 0,
  "msg": "",
  "data": {
    "sz000001": {
      "qfqday": [["2026-11-13", "11.76", "12.07", "12.10", "11.70", "2189429"]]
    }
  }
}
```

## 业务码

| 现象 | 含义 | 处理建议 |
| --- | --- | --- |
| `code=0` + 存在 `qfqday` / `day` | 正常 | 解析数组 |
| `code=0` + 两个键都不存在 | 代码无效或区间内无数据 | 返回空 `Bars` |
| `code≠0` | 参数或代码错误 | 读 `msg` 排查 |

## 限制与配额

站点未公布官方配额，下表是「接口能力边界」与实测经验；未实测的项已明确标注。

| 维度 | 结论 | 依据 |
| --- | --- | --- |
| 证券范围 | 沪深（`sh` / `sz`）；**不支持北交所（`bj`）与 B 股**，规则同 `GetQuoteSnapshot` | 代码 `symbolOf` |
| 条数 | `count` 默认 `1000`；**更大的上限未实测** | 代码常量 |
| 历史区间 | 由 `param` 的起止日期决定 | 接口参数 |
| 最早数据 | **未实测** | — |
| 限流 | 站点未公布配额 | — |
| 其他 | 不返回成交额 | 上游实测 |

## 客户端用法

```go
client := gtimg.New(gtimg.Config{})
result, err := client.GetKLine(ctx, gtimg.GetKLineRequest{
    Code:   "000001",
    Begin:  "2026-11-01",
    End:    "2026-11-30",
    Adjust: true, // 前复权
})
```

## 相关实现

- `component/quote/tencent/gtimg/client.go`：配置与 `APIError`
- `component/quote/tencent/gtimg/kline.go`：`GetKLine` 与行解析
- `component/quote/tencent/gtimg/snapshot_test.go`：日线参数与复权分支用例
