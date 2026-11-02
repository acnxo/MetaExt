# GetNewsList

获取指定个股的新闻列表，返回标题、来源、链接与摘要。
机器可读规格见
[Meta/OpenAPI/MooMoo/QuoteAPIV2/GetNewsList.yaml](../../../../Meta/OpenAPI/MooMoo/QuoteAPIV2/GetNewsList.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://www.moomoo.com` |
| 路径 | `/quote-api/quote-v2/get-news-list` |
| operationId | `getNewsList` |
| 代码入口 | `Client.GetNewsList`（`component/quote/moomoo/quoteapiv2` 包） |
| 备注 | 参数命名风格混用：`stock_id`、`market_type` 为 snake_case，`subType` 为 camelCase；响应中的 `seq_mark` 用于翻页，但翻页参数名尚未确认。 |

## 请求参数

### 查询参数

参数书写顺序就是签名输入的键序，键名保持接口原始的命名风格，不能为了统一风格而改名。

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `stock_id` | 是 | string | 服务端内部股票 ID，snake_case | `76033806042898` |
| `market_type` | 否 | string | 市场类型，snake_case；缺省时补默认值 `1` | `1` |
| `type` | 否 | string | 新闻类型；缺省时补默认值 `0` | `0` |
| `subType` | 否 | string | 新闻子类型，camelCase；缺省时补默认值 `0` | `0` |
| `_` | 否 | string | 毫秒级时间戳；缺省时整键省略 | `1781592384789` |

`market_type` 与 `type` 虽然标为可选，但服务端对缺少它们的请求会直接报错：缺少 `type`
返回 `code=404`，缺少 `market_type` 返回 `code=500`。因此实现中缺省时会显式补上默认值，
而不是省略该键。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `quote-token` | 是 | 请求签名，10 个字符，生成方式见 [Signature.md](Signature.md) |
| `futu-x-csrf-token` | 是 | CSRF 标记，该接口不校验取值但必须存在，且与 Cookie 中的 `csrfToken` 一致 |
| `cookie` | 是 | 需包含 `csrfToken` 与 `locale` 两项，例如 `csrfToken=0000000000; locale=zh-cn` |
| `accept` | 否 | 固定 `application/json` |
| `referer` | 否 | 来源页，建议传对应的新闻页地址 |
| `user-agent` | 否 | 浏览器 UA，非浏览器 UA 可能被拒绝 |

### 请求体

`GET` 请求，无请求体，不发送 `content-type`。

## 签名规则

完整算法、校验向量与排查步骤见 [Signature.md](Signature.md)。

本接口的签名输入为查询参数的紧凑 JSON，键名保留原始的 snake_case / camelCase 混用：

```json
{"stock_id":"76033806042898","market_type":"1","type":"0","subType":"0","_":"1781592384789"}
```

对应 `quote-token = 055b0960ec`。省略时间戳时签名输入为前四个键，此时 `_` 也不出现在 URL 中。

## 响应

### 响应头

| 响应头 | 说明 |
| --- | --- |
| `content-type` | `application/json` |

响应头不属于接口契约，业务成败一律以响应体的 `code` 为准。

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `code` | integer | 业务码，`0` 表示成功 |
| `message` | string | 业务提示，成功时通常为「成功」 |
| `data` | object | 业务数据，失败时可能为 `null` |
| `data.server_time` | integer | 服务端时间戳，毫秒，`int64` |
| `data.has_more` | boolean | 是否还有更多新闻 |
| `data.seq_mark` | string | 分页标记，翻页时回传；对应查询参数名尚未确认 |
| `data.list` | array | 新闻条目，按服务端顺序返回 |

### 条目字段

| 字段 | 字符串形式 | 数值形式 | 说明 |
| --- | --- | --- | --- |
| `id` | `"news-1"` | 不出现 | 新闻唯一 ID |
| `title` | `"小米集团发布新品"` | 不出现 | 新闻标题 |
| `time` | 不出现 | `int64`，秒级时间戳，如 `1781537154` | 发布时间 |
| `source` | `"新浪财经"` | 不出现 | 新闻来源 |
| `url` | `"https://example.com/news/1"` | 不出现 | 详情链接，原样保留 |
| `summary` | `"摘要内容"` | 不出现 | 新闻摘要，可能为空串 |

### 示例

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "server_time": 1781592387840,
    "has_more": true,
    "seq_mark": "1_1781537154",
    "list": [
      {
        "id": "news-1",
        "title": "小米集团发布新品",
        "time": 1781537154,
        "source": "新浪财经",
        "url": "https://example.com/news/1",
        "summary": "摘要内容"
      }
    ]
  }
}
```

## 业务码

| code | 含义 | 处理建议 |
| --- | --- | --- |
| `0` | 成功 | 读取 `data.list`，按 `has_more` 判断是否还有下一页 |
| `-403` | 签名被拒 | 检查签名输入的键序、键名风格与值形式 |
| `404` | 必填参数缺失 | 常见原因是缺少 `type`；实现中已补默认值 |
| `500` | 参数错误 | 常见原因是缺少 `market_type` 或取值不被接受 |
| 其它非 0 | 业务失败 | 按 `message` 排查，不要按成功处理 |

## 相关实现

- `newslist.go`：参数默认值、参数构造与响应解析
- `client.go`：配置、错误类型与请求发送，`CodeParamsMissing` 用于识别 404
- `signature.go`：签名输入拼装与 `quote-token` 计算
- `newslist_test.go`：键序、默认值补全、404 分类与解析
