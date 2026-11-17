# GetIndexKLine

万得官网指数日线接口。按官网的 `indexId` 取**最近约 60 个交易日**的开高低收、成交量、
成交额与涨跌幅，本项目用它把「万得微盘股指数」等基准写入本地数据。机器可读规格见
[Meta/OpenAPI/Wind/IndexAPI/GetIndexKLine.yaml](../../../../Meta/OpenAPI/Wind/IndexAPI/GetIndexKLine.yaml)。

> 与 Wind 终端的 `WindPy` 不同：本接口是公开网页 API，**不需要安装 Wind 终端、不需要登录**，
> 因此可以直接在 CI 里跑。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://index_api.wind.com.cn` |
| 路径 | `/indexofficialwebsite/Kline` |
| operationId | `getIndexKLine` |
| 代码入口 | `indexapi.Client.GetIndexKLine`（`component/quote/wind/indexapi` 包） |
| 鉴权 | 无（无需 cookie、无 token） |
| 备注 | 只回最近约 60 个交易日；最高价字段名是 **`hight`**；TLS 主机名不匹配 |

要点：

- 只返回最近约 60 个交易日，增量更新时必须与本地数据按日期合并去重；
- `indexId` 是官网内部编码（Base64 形态），与 `windCode` 不是一套，本库内置映射表；
- SSL 证书是 `*.wind.com.cn`，与 `index_api.wind.com.cn` 不匹配：本库关闭主机名校验，
  但仍通过 `VerifyConnection` 做证书链校验。

## 请求参数

### 查询参数

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `indexId` | 是 | string | 官网内部编码，需 URL 编码（含 `+` `/` `=`） | `2O5u8vTCa4KiLTLQ7DT/9w==` |
| `lan` | 是 | string | 语言，固定 `cn` | `cn` |

内置映射（`indexapi.DefaultIndexIDs`）：

| windCode | 名称 | indexId |
| --- | --- | --- |
| `868008.WI` | 万得微盘股指数 | `tShUKnLO4cLQ8v+TnUd2uQ==` |
| `8841431.WI` | 万得微盘股日频等权指数 | `2O5u8vTCa4KiLTLQ7DT/9w==` |

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `user-agent` | 是 | 浏览器 UA |

### 请求体

`GET` 请求，无请求体。

## 签名规则

无签名、无 token、无 cookie。唯一特殊之处是 TLS：站点证书为 `*.wind.com.cn`，
与 `index_api.wind.com.cn` 不匹配，需关闭主机名校验但保留证书链校验。

## 响应

### 响应头

| 响应头 | 说明 |
| --- | --- |
| `content-type` | `application/json` |

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `Result` | array | 日线数组；为空表示无数据 |
| `Message` / `Code` | string / int | 失败时的提示与错误码 |

### 响应字段

| 字段 | `IndexBar` | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- | --- |
| `tradeDate` | `Date` | 不出现 | `int64`，毫秒 | 按东八区换算 `yyyy-MM-dd` |
| `open` | `Open` | `"6201.33"` | `float64`，点 | |
| `hight` | `High` | `"6230.11"` | `float64`，点 | **字段名是 `hight`** |
| `low` | `Low` | `"6170.55"` | `float64`，点 | |
| `close` | `Close` | `"6188.07"` | `float64`，点 | `≤0` 的行丢弃 |
| `volume` | `Volume` | 不出现 | `int64` | 站点原始口径 |
| `amount` | `Amount` | 不出现 | `int64`，元 | |
| `pctChange` | `ChangeRatio` | `"-0.21"` | `float64`，% | 可能为 `null` |

### 示例

```json
{
  "Result": [
    {"tradeDate": 1794499200000, "open": "6201.33", "hight": "6230.11",
     "low": "6170.55", "close": "6188.07", "volume": 12345678,
     "amount": 9876543210, "pctChange": "-0.21"}
  ]
}
```

## 业务码

| 现象 | 含义 | 处理建议 |
| --- | --- | --- |
| `Result` 非空 | 正常 | 按日期合并进本地数据 |
| `Result=[]` 或缺失 | 编码错误或站点异常 | 返回空 `Bars`，保留本地已有数据 |
| SSL 报 `hostname mismatch` | 证书主机名不匹配 | 本库已只关闭主机名校验 |
| HTTP 非 200 | 限流或站点异常 | 退避重试 |

## 限制与配额

站点未公布官方配额，下表是「接口能力边界」；未实测的项已明确标注。

| 维度 | 结论 | 依据 |
| --- | --- | --- |
| 证券范围 | 官网收录的万得指数；本项目内置 `868008.WI`、`8841431.WI` 两个 `indexId` 映射 | 代码常量 |
| 分页 | **无分页** | 接口语义 |
| 历史区间 | **只返回最近约 60 个交易日**，不支持指定起止日期 | 上游实测 + 代码 |
| 最早数据 | 仅最近约 60 个交易日；更早的历史需要其他数据源 | 上游实测 |
| 限流 | 站点未公布配额 | — |
| 其他 | `indexId` 是硬编码的内部编码，**官网改版可能失效**；TLS 证书主机名不匹配 | 上游实测 |

## 客户端用法

```go
client := indexapi.New(indexapi.Config{})
result, err := client.GetIndexKLine(ctx, indexapi.GetIndexKLineRequest{
    IndexCode: "8841431.WI",
})
for _, bar := range result.Bars {
    fmt.Println(bar.Date, bar.Close, bar.Amount)
}
```

## 相关实现

- `component/quote/wind/indexapi/client.go`：配置、TLS 配置、`APIError`
- `component/quote/wind/indexapi/indexkline.go`：`GetIndexKLine`、日期换算与去重
- `component/quote/wind/indexapi/indexkline_test.go`：映射表、`hight` 拼写、TLS 校验
