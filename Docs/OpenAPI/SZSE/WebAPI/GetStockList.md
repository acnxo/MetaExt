# GetStockList

深交所 A 股列表接口。按页返回深市证券的代码、简称与**上市日期**；本站点每页只返回 20 条、
全量约 146 页，因此需要先取第 1 页拿到总页数再并发翻页。机器可读规格见
[Meta/OpenAPI/SZSE/WebAPI/GetStockList.yaml](../../../../Meta/OpenAPI/SZSE/WebAPI/GetStockList.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `http://www.szse.cn`（**只支持 HTTP**） |
| 路径 | `/api/report/ShowReport/data` |
| operationId | `getStockList` |
| 代码入口 | `webapi.Client.GetStockList`（`component/quote/szse/webapi` 包） |
| 鉴权 | 无（无需 cookie、无签名） |
| 备注 | 每页 20 条；`All=true` 时并发翻完并按页序拼接；任一分页失败即整体报错 |

要点：

- 先请求第 1 页拿 `metadata.pagecount`，再并发请求第 2 页至末页（默认 8 并发）；
- 简称字段带 `<u>` / `<a>` 高亮标签，需两步清理；
- 任一分页失败即返回错误，不返回残缺结果。

## 请求参数

### 查询参数

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `SHOWTYPE` | 是 | string | 固定 `JSON` | `JSON` |
| `CATALOGID` | 是 | string | 固定 `1110`（上市公司列表） | `1110` |
| `TABKEY` | 是 | string | 固定 `tab1`（A 股页签） | `tab1` |
| `PAGENO` | 是 | int | 页码，从 `1` 开始 | `1` |

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `user-agent` | 是 | 浏览器 UA |
| `referer` | 是 | 固定 `http://www.szse.cn/` |

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

顶层是**数组**（按页签分组），本库只取第 0 个元素：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `[0].metadata.pagecount` | string | 总页数 |
| `[0].metadata.recordcount` | string | 总记录数 |
| `[0].data` | array | 当前页记录 |

### 响应字段

| 字段 | `StockItem` | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- | --- |
| `agdm` | `Code` | `"000001"` | 不出现 | 六位字符串 |
| `agjc` | `Name` | `"<u>平安银行</u>"` | 不出现 | 含高亮标签，见下 |
| `agssrq` | `ListDate` | `"1991-04-03"` | 不出现 | `yyyy-MM-dd` |

`agjc` 清理规则：

```text
"<u>平安银行</u>"                              → 平安银行
"<a href='...' target='_blank'>XD平安银行</a>"  → XD平安银行
```

先去掉 `<u>` / `</u>`，再以 `'>` 分割取最后一段，最后以 `<` 分割取第一段并 `strip`。

### 示例

```json
[
  {
    "metadata": {"pagecount": "146", "recordcount": "2912", "pageno": "1"},
    "data": [
      {"agdm": "000001", "agjc": "<u>平安银行</u>", "agssrq": "1991-04-03"}
    ]
  }
]
```

## 业务码

| 现象 | 含义 | 处理建议 |
| --- | --- | --- |
| HTTP `200` + `[0].data` 非空 | 正常 | 解析并翻下一页 |
| `metadata.pagecount` 缺失 | 参数或页签不对 | 检查 `CATALOGID` 与 `TABKEY` |
| 单页失败 | 限流或网络抖动 | 本库整体报错，调用方可重试 |

## 限制与配额

站点未公布官方配额，下表是「接口能力边界」；未实测的项已明确标注。

| 维度 | 结论 | 依据 |
| --- | --- | --- |
| 证券范围 | 深市全部证券，含创业板与 B 股；调用方按代码前缀过滤 | 接口语义 |
| 分页 | 每页固定 20 条，全量约 146 页；本库按 `Workers`（默认 8）并发翻完 | 上游实测 |
| 历史区间 | 只返回**当前上市清单**，没有历史快照 | 接口语义 |
| 最早数据 | 不适用 | — |
| 限流 | 站点未公布配额；单页失败会整体报错，调用方可重试 | 代码行为 |
| 其他 | 只支持 HTTP；简称带高亮标签 | 站点行为 |

## 客户端用法

```go
client := webapi.New(webapi.Config{})
page, err := client.GetStockList(ctx, webapi.GetStockListRequest{All: true})
fmt.Println(page.PageCount, page.RecordCount, len(page.Items))
```

## 相关实现

- `component/quote/szse/webapi/client.go`：配置、`APIError`
- `component/quote/szse/webapi/stocklist.go`：`GetStockList`、并发翻页、名称清理
- `component/quote/szse/webapi/stocklist_test.go`：单页/全量、页序、名称清理
