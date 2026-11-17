# GetIndexPerf

中证指数官网的指数表现接口。按指数代码取指定区间的日行情，本项目用它取基准指数
（中证 2000 `932000`、中证全指 `000985`）的收盘序列。机器可读规格见
[Meta/OpenAPI/CSIndex/IndexAPI/GetIndexPerf.yaml](../../../../Meta/OpenAPI/CSIndex/IndexAPI/GetIndexPerf.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://www.csindex.com.cn` |
| 路径 | `/csindex-home/perf/index-perf` |
| operationId | `getIndexPerf` |
| 代码入口 | `indexapi.Client.GetIndexPerf`（`component/quote/csindex/indexapi` 包） |
| 鉴权 | 无（无需 cookie、无签名） |
| 备注 | 指数有发布日；`tradeDate` 两种格式混用；官网偶有重复行 |

要点：

- 指数有**发布日**，早于发布日的区间没有数据（中证 2000 发布日 2023-08-11）；
- `tradeDate` 可能是 `yyyyMMdd` 或 `yyyy-MM-dd`，本库统一归一化为 `yyyy-MM-dd`；
- 重复日期按「后者覆盖前者」去重，最终按日期升序返回。

## 请求参数

### 查询参数

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `indexCode` | 是 | string | 指数代码（六位数字） | `932000` |
| `startDate` | 否 | string | 起始日期 `yyyyMMdd` | `20230811` |
| `endDate` | 否 | string | 结束日期 `yyyyMMdd` | `20261113` |

本项目使用的指数代码：

| 代码 | 名称 | 起始约束 |
| --- | --- | --- |
| `932000` | 中证 2000 | 发布日 `2023-08-11` 起 |
| `000985` | 中证全指 | 官网数据自 2013 年起 |

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `user-agent` | 是 | 浏览器 UA |
| `referer` | 是 | 固定 `https://www.csindex.com.cn/` |

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
| `code` | int | 站点不同版本取值不同（`200` 或 `0`），本库只判断 `data` |
| `data` | array | 日行情数组 |

### 响应字段

| 字段 | `IndexBar` | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- | --- |
| `tradeDate` | `Date` | `"20261113"` | 不出现 | 归一化为 `yyyy-MM-dd` |
| `open` | `Open` | `"6201.33"` | `float64`，点 | |
| `high` | `High` | `"6230.11"` | `float64`，点 | |
| `low` | `Low` | `"6170.55"` | `float64`，点 | |
| `close` | `Close` | `"6188.07"` | `float64`，点 | `≤0` 的行丢弃 |
| `change` | `Change` | `"-13.26"` | `float64`，点 | |
| `changeRatio` | `ChangeRatio` | `"-0.21"` | `float64`，% | |

### 示例

```json
{
  "code": 200,
  "data": [
    {"tradeDate": "20261113", "open": "6201.33", "high": "6230.11",
     "low": "6170.55", "close": "6188.07", "change": "-13.26", "changeRatio": "-0.21"}
  ]
}
```

## 业务码

| 现象 | 含义 | 处理建议 |
| --- | --- | --- |
| `data` 非空 | 正常 | 去重排序后取用 |
| `data=[]` 或缺失 | 代码错误或区间早于发布日 | 返回空 `Bars` |
| HTTP 非 200 | 限流或站点异常 | 退避重试 |

## 限制与配额

站点未公布官方配额，下表是「接口能力边界」；未实测的项已明确标注。

| 维度 | 结论 | 依据 |
| --- | --- | --- |
| 证券范围 | 中证指数系列；本项目使用 `932000`（中证 2000）与 `000985`（中证全指） | 上游配置 |
| 分页 | **无分页**，一次返回请求区间内的全部交易日 | 接口语义 |
| 历史区间 | `startDate` ~ `endDate` | 接口参数 |
| 最早数据 | 中证 2000 自发布日 `2023-08-11`；中证全指官网数据自 2013 年起 | 上游文档 |
| 限流 | 站点未公布配额 | — |
| 其他 | `tradeDate` 两种格式混用；官网偶有非交易日重复行 | 上游实测 |

## 客户端用法

```go
client := indexapi.New(indexapi.Config{})
result, err := client.GetIndexPerf(ctx, indexapi.GetIndexPerfRequest{
    IndexCode: "932000",
    Begin:     "20230811",
    End:       "20261113",
})
for _, bar := range result.Bars {
    fmt.Println(bar.Date, bar.Close)
}
```

## 相关实现

- `component/quote/csindex/indexapi/client.go`：配置、`APIError`
- `component/quote/csindex/indexapi/indexperf.go`：`GetIndexPerf`、日期归一化与去重
- `component/quote/csindex/indexapi/indexperf_test.go`：参数、去重、日期归一化
