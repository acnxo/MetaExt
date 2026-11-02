# QuoteAPIV2 接口规格

本目录记录 moomoo（富途）网页端行情接口 `/quote-api/quote-v2/*` 的 HTTP 规格，
用 OpenAPI 3.0.3 描述每个接口的参数、请求头、响应结构与错误码。

## 文件

| 文件 | 说明 |
| --- | --- |
| `openapi.yaml` | quote-v2 接口规范，当前包含 `GET /quote-api/quote-v2/get-kline` |

## 与代码的对应关系

- 规范中的 `operationId` 与 Go 客户端方法名一致：`getKline` 对应
  `component/quote/moomoo/quoteapiv2` 包中的 `Client.GetKline`；
- 查询参数在规范中的书写顺序就是签名输入的键序，与代码里的参数构造顺序一致；
- 响应结构 `KlineResponse` 对应代码中的 `klinePayload`、`KlineItem` 对应 `klineItem`。

新增接口时，先在 `openapi.yaml` 中补上该接口的规格，再按同一命名规则实现客户端方法。

## 查看与校验

`openapi.yaml` 是标准 OpenAPI 3.0 文档，可用任意兼容 OpenAPI 3.0 的工具预览、
生成文档或做 lint，例如编辑器插件、Swagger UI、Redoc 一类的查看器。

改动后用 YAML 解析即可确认语法，并检查 `$ref` 指向的组件是否存在。

## 维护约定

- 规范依据网页端实际发出的请求与返回的响应整理，不是厂商公开发布的接口文档；
- 站点行为发生变化时，先实测确认，再同步更新本目录，最后修改实现，避免文档与代码互相矛盾；
- 字段口径以响应原始形态为准：数值可能是字符串、也可能是数字，缺失时可能为 `null`，
  规范中用 `FlexNumber` 统一描述。
