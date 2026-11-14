# 中信证券 XetApp 接口规格

中信证券 XetApp 行情网关（`https://xetapp.citics.com`）的接口规格与说明。该网关用
`action` 编号区分能力，响应是「表头 + 竖线分隔记录」的字符串表，而不是常规 JSON 数组。

每个接口由两份文件描述：机器可读的 OpenAPI 3.0.3 规格（`.yaml`）保存在仓库的
`Meta/OpenAPI/Citics/XetApp/` 下，面向人阅读的说明（`.md`）保存在本目录，两份文件同名。

## 文件

| 文件 | 位置 | 说明 |
| --- | --- | --- |
| `GetQuoteSnapshotCNAStock.yaml` | `Meta/OpenAPI/Citics/XetApp/` | `POST /reqxml`（`action=20199`）的 OpenAPI 规格 |
| `GetQuoteSnapshotCNAStock.md` | 本目录 | 同一接口的可读说明，含字段码表与实测记录 |

## 接口

| 接口 | 方法 | 路径 | action | 用途 |
| --- | --- | --- | --- | --- |
| `GetQuoteSnapshotCNAStock` | `POST` | `/reqxml` | `20199` | A 股全市场行情快照，一次请求拿全量 |

## 与代码的对应关系

- `operationId` 与 Go 方法名一致：`getQuoteSnapshotCNAStock` 对应
  `component/quote/citics/xetapp` 包中的 `Client.GetQuoteSnapshotCNAStock`；
- 命令行工具：`_cmd/snapshot`，按池拉取并输出表格或 JSON；
- 该网关无鉴权、无签名，因此没有单独的签名说明文档。

## 命名约定

- 文件名取 `action` 能力的语义名，与 Go 方法名一致；
- 新增能力时按同一规则分别添加 `.yaml` 与同名 `.md`：`.yaml` 放 `Meta/OpenAPI/Citics/XetApp/`，
  `.md` 放本目录。
