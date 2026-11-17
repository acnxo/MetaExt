# GetStockList

上交所 A 股列表接口。一次请求返回沪市全部 A 股的代码、简称与**上市日期**，是沪市上市
日期的权威来源。机器可读规格见
[Meta/OpenAPI/SSE/Query/GetStockList.yaml](../../../../Meta/OpenAPI/SSE/Query/GetStockList.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `http://query.sse.com.cn`（**只支持 HTTP**） |
| 路径 | `/sseQuery/commonQuery.do` |
| operationId | `getStockList` |
| 代码入口 | `query.Client.GetStockList`（`component/quote/sse/query` 包） |
| 鉴权 | 无（无需 cookie、无签名） |
| 备注 | 单页取全部，`pageSize` 设 3000 足够；`sqlId` 固定 |

要点：

- 站点只支持 HTTP；必须带 `referer: http://www.sse.com.cn/`，否则可能返回空 `result`；
- 返回包含科创板与 B 股，调用方按代码前缀过滤；
- `LIST_DATE` 不是八位数字的行会被丢弃（宁可漏也不要写入错误上市日）。

## 请求参数

### 查询参数

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `sqlId` | 是 | string | 固定 `COMMON_SSE_CP_GPJCTPZ_GPLB_GP_L` | 同左 |
| `type` | 是 | string | 固定 `inParams` | `inParams` |
| `STOCK_TYPE` | 是 | int | 证券类型，`1` 为 A 股 | `1` |
| `COMPANY_STATUS` | 是 | string | 公司状态过滤 | `2,4,5,7,8` |
| `pageHelp.pageSize` | 是 | int | 每页条数，默认 `3000` | `3000` |
| `pageHelp.pageNo` | 是 | int | 页码，本库只取第 1 页 | `1` |
| `pageHelp.beginPage` | 是 | int | 固定 `1` | `1` |
| `pageHelp.endPage` | 是 | int | 固定 `1` | `1` |
| `pageHelp.cacheSize` | 是 | int | 固定 `1` | `1` |

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `user-agent` | 是 | 浏览器 UA |
| `referer` | 是 | 固定 `http://www.sse.com.cn/` |

### 请求体

`GET` 请求，无请求体。

## 签名规则

无签名、无 token。站点按 `Referer` 做来源校验。

## 响应

### 响应头

| 响应头 | 说明 |
| --- | --- |
| `content-type` | `application/json`（可能被标成 `text/html`，内容仍是 JSON） |

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `pageHelp` | object | 分页信息，本库不使用 |
| `result` | array | 证券列表 |

### 响应字段

| 字段 | `StockItem` | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- | --- |
| `A_STOCK_CODE` | `Code` | `"600000"` | 不出现 | 六位，可能带空格（已 trim） |
| `SEC_NAME_CN` | `Name` | `"浦发银行"` | 不出现 | 可能含 ST / 退 |
| `LIST_DATE` | `ListDate` | `"19991110"` | 不出现 | `yyyyMMdd`；非八位数字则丢弃 |

### 示例

```json
{
  "pageHelp": {"pageNo": 1, "pageSize": 3000, "total": 2263},
  "result": [
    {"A_STOCK_CODE": "600000", "SEC_NAME_CN": "浦发银行", "LIST_DATE": "19991110"}
  ]
}
```

## 业务码

| 现象 | 含义 | 处理建议 |
| --- | --- | --- |
| HTTP `200` + `result` 非空 | 正常 | 逐条过滤与解析 |
| HTTP `200` + `result=[]` | 参数不被接受或来源校验失败 | 检查 `Referer` 与 `sqlId` |
| HTTP 非 200 / 连接被重置 | 限流或网络问题 | 退避重试 |

## 限制与配额

站点未公布官方配额，下表是「接口能力边界」；未实测的项已明确标注。

| 维度 | 结论 | 依据 |
| --- | --- | --- |
| 证券范围 | 沪市全部 A 股，含科创板（`688`）与 B 股；调用方按代码前缀过滤 | 接口语义 |
| 分页 | 本库 `pageSize=3000` 一次取全、**只取第 1 页**；沪市 A 股总数尚未超过 3000 | 代码常量 |
| 历史区间 | 只返回**当前上市清单**，没有历史快照 | 接口语义 |
| 最早数据 | 不适用 | — |
| 限流 | 站点未公布配额 | — |
| 其他 | 只支持 HTTP | 站点行为 |

## 客户端用法

```go
client := query.New(query.Config{})
items, err := client.GetStockList(ctx, query.GetStockListRequest{})
for _, item := range items {
    if strings.HasPrefix(item.Code, "600") || strings.HasPrefix(item.Code, "601") {
        fmt.Println(item.Code, item.Name, item.ListDate)
    }
}
```

## 相关实现

- `component/quote/sse/query/client.go`：配置、`APIError`
- `component/quote/sse/query/stocklist.go`：`GetStockList`、日期校验
- `component/quote/sse/query/stocklist_test.go`：参数、非法行剔除、错误分类
