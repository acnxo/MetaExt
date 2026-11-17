# GetStockList

巨潮资讯的深市证券清单接口。一次请求拿到深市全部代码与简称，是「代码清单」的首选来源；
**不含上市日期**。机器可读规格见
[Meta/OpenAPI/CNINFO/DataAPI/GetStockList.yaml](../../../../Meta/OpenAPI/CNINFO/DataAPI/GetStockList.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `http://www.cninfo.com.cn`（**只支持 HTTP**） |
| 路径 | `/new/data/szse_stock.json` |
| operationId | `getStockList` |
| 代码入口 | `dataapi.Client.GetStockList`（`component/quote/cninfo/dataapi` 包） |
| 鉴权 | 无（无需 cookie、无签名） |
| 备注 | 一次返回全部深市证券，无分页；无上市日期字段 |

要点：

- 相对深交所官方列表（约 146 页），本接口只要 1 次请求；
- 返回包含创业板、B 股等，需按代码前缀过滤；
- 响应可能是 `{"stockList": [...]}`，也可能直接是数组，本库两种都兼容。

## 请求参数

### 查询参数

无查询参数，路径即全部输入。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `user-agent` | 是 | 浏览器 UA |
| `referer` | 是 | 固定 `http://www.cninfo.com.cn/` |

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

顶层有两种形态，本库都兼容：

| 形态 | 说明 |
| --- | --- |
| `{"stockList": [...]}` | 常见形态 |
| `[...]` | 旧版或镜像行为 |

### 响应字段

| 字段 | `StockItem` | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- | --- |
| `code` | `Code` | `"000001"` | 不出现 | 六位；为空的行丢弃 |
| `zwjc` | `Name` | `"平安银行"` | 不出现 | 可能含 ST / 退 |
| `orgId` | — | `"9900001234"` | 不出现 | 本库不使用 |
| `category` | — | `"A股"` | 不出现 | 本库不使用，改按代码前缀过滤 |

### 示例

```json
{
  "stockList": [
    {"code": "000001", "zwjc": "平安银行", "orgId": "9900001234", "category": "A股"}
  ]
}
```

## 业务码

| 现象 | 含义 | 处理建议 |
| --- | --- | --- |
| HTTP `200` + 列表非空 | 正常 | 按前缀过滤后使用 |
| HTTP `200` + 空列表 | 站点异常或结构变化 | 返回 `*APIError`，调用方可换源 |
| HTTP 非 200 | 限流或网络问题 | 退避重试 |

## 限制与配额

站点未公布官方配额，下表是「接口能力边界」；未实测的项已明确标注。

| 维度 | 结论 | 依据 |
| --- | --- | --- |
| 证券范围 | 深市全部证券，含创业板与 B 股 | 接口语义 |
| 分页 | **无分页**，一次返回全量 | 接口语义 |
| 历史区间 | 只返回**当前清单**，且**不含上市日期** | 接口字段 |
| 最早数据 | 不适用 | — |
| 限流 | 站点未公布配额 | — |
| 其他 | 只支持 HTTP；响应有两种外层形态 | 站点行为 |

## 客户端用法

```go
client := dataapi.New(dataapi.Config{})
items, err := client.GetStockList(ctx)
for _, item := range items {
    fmt.Println(item.Code, item.Name)
}
```

## 相关实现

- `component/quote/cninfo/dataapi/client.go`：配置、`APIError`
- `component/quote/cninfo/dataapi/stocklist.go`：`GetStockList`、双形态解析
- `component/quote/cninfo/dataapi/stocklist_test.go`：两种响应形态、空列表、错误分类
