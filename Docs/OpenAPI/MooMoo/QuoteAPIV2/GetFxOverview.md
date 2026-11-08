# GetFxOverview

一次返回 9 种货币两两组合的 **72 个货币对**最新价，供网页端汇率换算器使用；无业务参数、
无分页。机器可读规格见
[Meta/OpenAPI/MooMoo/QuoteAPIV2/GetFxOverview.yaml](../../../../Meta/OpenAPI/MooMoo/QuoteAPIV2/GetFxOverview.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://www.moomoo.com` |
| 路径 | `/quote-api/quote-v2/get-fx-overview` |
| operationId | `getFxOverview` |
| 代码入口 | `Client.GetFxOverview`（`component/quote/moomoo/quoteapiv2` 包） |
| 备注 | `data` 是**以货币对代码为键的对象**，不是列表；键没有顺序语义。 |

## 请求参数

### 查询参数

| 参数 | 必填 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `_` | 否 | string | 毫秒级时间戳；**可省略**，省略时 URL 不带查询串 | `1793859303524` |

除 `_` 外无其它查询参数：货币对清单由服务端固定返回，无法按币种筛选。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `quote-token` | 是 | 请求签名，10 个字符，生成方式见 [Signature.md](Signature.md) |
| `futu-x-csrf-token` | 是 | CSRF 标记，观测到不校验取值但需携带，且与 Cookie 中的 `csrfToken` 一致 |
| `cookie` | 是 | 需包含 `csrfToken` 与 `locale` 两项，例如 `csrfToken=0000000000; locale=zh-cn` |
| `referer` | 否 | 来源页；影响返回数据的语言与数值单位口径，默认取简体中文站 `https://www.moomoo.com/hans/` |
| `user-agent` | 否 | 浏览器 UA，非浏览器 UA 可能被拒绝 |
| `accept` | 否 | 固定 `application/json` |

### 请求体

`GET` 请求，无请求体，不发送 `content-type`。

## 签名规则

完整算法与排查步骤见 [Signature.md](Signature.md)。带时间戳时签名输入为：

```json
{"_":"1793859303524"}
```

对应 `quote-token = d1bea85060`。

省略时间戳时请求不带查询串，**签名输入为空对象**（不是空串）：

```json
{}
```

对应 `quote-token = 279c67896d`。

> 空串输入在 `Sign` 里有「降级为字面量 `quote`」的分支，会得到 `58af97268c` 从而被服务端
> 拒绝。实现中省略参数时显式按 `{}` 计算，走不到那条分支。

## 响应

### 外层结构

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `code` | integer | 业务码，`0` 表示成功 |
| `message` | string | 业务提示，成功时通常为「成功」 |
| `data` | object | 以货币对代码为键的对象，固定 72 个键 |

### `data.<货币对代码>` 字段

| 字段 | 类型 | 说明 | 示例 |
| --- | --- | --- | --- |
| `time` | int64 | 报价时间，毫秒 Unix 时间戳 | `1793859536104` |
| `priceNominal` | string | 最新价，小数位随货币对精度变化 | `6.66623` |
| `price` | string | 最新价；已观测与 `priceNominal` 始终相同，用途差异未验证 | `6.66623` |

### 货币对键名规则

键名由「基础货币 + 报价货币」各 3 位大写代码直接拼接，共 6 位。已观测覆盖 9 种货币：

| 货币代码 | 含义 |
| --- | --- |
| `CNH` | 离岸人民币 |
| `HKD` | 港币 |
| `USD` | 美元 |
| `EUR` | 欧元 |
| `GBP` | 英镑 |
| `AUD` | 澳元 |
| `SGD` | 新加坡元 |
| `CAD` | 加元 |
| `JPY` | 日元 |

9 种货币两两组合（不含同币种）共 `9 × 8 = 72` 个键，已观测键数与组合数一致。

`data` 的键**没有顺序语义**：服务端不保证顺序，读取时按需取键，不要依赖迭代顺序。
已观测样本**不含**贵金属（`XAUUSD` / `XAGUSD`）与美元指数（`USDindex` / `EURindex`），
这几类只在 [GetFxRateList](GetFxRateList.md) 的分组中出现。

### 示例

节选 4 个键，实际返回全部 72 个：

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "USDCNH": {"time": 1793859536104, "priceNominal": "6.66623", "price": "6.66623"},
    "EURUSD": {"time": 1793859536488, "priceNominal": "1.12614", "price": "1.12614"},
    "USDJPY": {"time": 1793859536470, "priceNominal": "157.971", "price": "157.971"},
    "JPYUSD": {"time": 1793859509942, "priceNominal": "0.006330", "price": "0.006330"}
  }
}
```

## 业务码

| code | 含义 | 处理建议 |
| --- | --- | --- |
| `0` | 成功 | 按货币对代码取 `data` 中的对象 |
| `-403` | 签名被拒 | 检查省略参数时是否误用空串计算 token |
| 其它非 0 | 业务失败 | 按 `message` 排查，不要按成功处理 |

## 相关实现

- `fxoverview.go`：请求构造（含空参数 `{}` 签名特例）与响应解析
- `client.go`：配置、错误类型与请求发送
- `signature.go`：签名输入拼装与 `quote-token` 计算
- `fxoverview_test.go`：两种签名向量、无查询串、空 `data` 与错误分类
