# GetKLine

拉取指定标的的全量 K 线数据，对应 OpenAPI 描述见同目录的 [GetKLine.yaml](GetKLine.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 路径 | `/quote-api/quote-v2/get-kline` |
| 站点 | `https://www.moomoo.com` |
| operationId | `getKLine` |
| 代码入口 | `Client.GetKLine`（`component/quote/moomoo/quoteapiv2` 包） |

接口没有分页，也没有起止时间参数，一次返回全部历史数据；返回顺序为时间升序
（最早在前），区间过滤由调用方完成。

## 请求参数

### 查询参数

参数书写顺序就是签名输入的键序，不能重排，也不能改用会按键名排序的序列化方式。

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `stockId` | 是 | string | 标的编号，不能为空 | `70000294` |
| `marketType` | 是 | string | 市场类型编号，原样透传 | `2` |
| `type` | 否 | string | K 线周期，为空按日 K 处理，日 K 取值 `2` | `2` |
| `marketCode` | 是 | string | 市场代码，原样透传 | `70` |
| `instrumentType` | 是 | string | 标的类型，原样透传 | `9` |
| `subInstrumentType` | 是 | string | 标的子类型，原样透传 | `9008` |
| `_` | 是 | string | 毫秒级 UNIX 时间戳，防缓存并参与签名，13 位数字 | `1780507278122` |

五个定位参数（`stockId`、`marketType`、`marketCode`、`instrumentType`、
`subInstrumentType`）来自行情元数据，属于不同编号体系，必须原样透传，不要做语义换算，
也不要把一个接口的编号套用到另一个接口上。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `quote-token` | 是 | 请求签名，10 个字符，生成方式见下节 |
| `futu-x-csrf-token` | 是 | CSRF 标记，该接口不校验取值但必须存在，且与 Cookie 中的 `csrfToken` 一致 |
| `cookie` | 是 | 需包含 `csrfToken` 与 `locale` 两项，例如 `csrfToken=0000000000; locale=zh-cn` |
| `referer` | 是 | 来源页，取站点页面地址 |
| `user-agent` | 是 | 浏览器 UA，非浏览器 UA 可能被直接拒绝 |
| `accept` | 否 | 固定 `application/json` |

## 签名规则

`quote-token` 由全部查询参数决定，分四步：

1. **拼装签名输入**：按参数在请求中出现的顺序，把名称与值写成紧凑 JSON，值一律先
   字符串化，形如：

   ```json
   {"stockId":"70000294","marketType":"2","type":"2","marketCode":"70","instrumentType":"9","subInstrumentType":"9008","_":"1780507278122"}
   ```

2. **第一段摘要**：以字面量 `quote_web` 为密钥对签名输入做 HMAC-SHA512，取十六进制
   结果的前 10 个字符。
3. **第二段摘要**：对上一步的结果字符串做 SHA-256，取十六进制结果的前 10 个字符，
   即为 `quote-token`。
4. **空输入**：签名输入为空时，以字面量 `quote` 代替，其余步骤不变。

上面示例的签名输入对应 `quote-token = 45671b44c1`，可用于校验实现是否正确。

`quote_web` 取自网页前端产物，不是用户凭据。

## 响应

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `code` | integer | 业务码，`0` 表示成功 |
| `message` | string | 业务提示，成功时通常为「成功」 |
| `data` | object | 业务数据，失败时可能为 `null` |
| `data.list` | array | K 线序列，按时间升序 |

### K 线字段

`data.list` 的每个元素是一根 K 线：

| 字段 | 字符串 | 数值 | 说明 |
| --- | --- | --- | --- |
| `k` | 否 | 是（整数） | K 线时间，秒级 UNIX 时间戳 |
| `o` | 是 | 是 | 开盘价 |
| `h` | 是 | 是 | 最高价 |
| `l` | 是 | 是 | 最低价 |
| `c` | 是 | 是 | 收盘价 |
| `v` | 是 | 是 | 成交量 |
| `t` | 是 | 是 | 成交额 |
| `r` | 是 | 是 | 涨跌幅，可能为 `null`，样本中均为 `null` |
| `lc` | 是 | 是 | 昨收价 |
| `cp` | 是 | 是 | 涨跌额，含符号，通常为字符串 |

「字符串」与「数值」两列分别表示该字段是否可能以该形式出现；两列都是「是」时，
以哪一列为准取决于当次响应，不能假定固定类型。接口对类型存在混用：开高低收与涨跌额
通常给字符串，量与额通常给数值，缺失时可能为 `null`。实现上按原始文本保留价格，
避免浮点精度损失。

### 示例

成功返回：

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "list": [
      {
        "k": 157870800,
        "o": "187.5",
        "c": "178.2",
        "h": "187.5",
        "l": "176",
        "v": 0,
        "t": 0,
        "r": null,
        "lc": 187.5,
        "cp": "-9.3"
      }
    ]
  }
}
```

签名被拒：

```json
{
  "code": -403,
  "message": "非法请求",
  "data": null
}
```

## 业务码

| code | 含义 | 处理建议 |
| --- | --- | --- |
| `0` | 成功 | 读取 `data.list` |
| `-403` | 签名被拒 | 检查签名输入的键序、取值与字符串化方式 |
| 其它非 0 | 业务失败 | 按 `message` 排查，不要按成功处理 |

HTTP 状态码只表示传输层结果，业务结果一律以 `code` 为准；网关层故障时可能直接返回
非 200 的纯文本响应。

## 相关实现

- `client.go`：客户端、请求装配与响应解析
- `signature.go`：签名输入拼装与 `quote-token` 计算
- `client_test.go`、`signature_test.go`：单元测试与固定签名向量
