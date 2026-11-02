# GetTradeCalendar

获取 A 股交易日历。数据来自新浪财经的静态文本资源，`txt` 中是站点自定义编码的日期
序列，需要解码后才能得到交易日列表。机器可读规格见
[Meta/OpenAPI/Sina/TradeCalendar/GetTradeCalendar.yaml](../../../../Meta/OpenAPI/Sina/TradeCalendar/GetTradeCalendar.yaml)。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://finance.sina.com.cn` |
| 路径 | `/realstock/company/klc_td_sh.txt` |
| operationId | `getTradeCalendar` |
| 代码入口 | `cnatradecalendar.Client.GetTradeCalendar`（`component/quote/sina/finance/cnatradecalendar` 包） |
| 备注 | 静态文本资源，非 JSON；无查询参数、无请求体、无接口级签名。 |

## 请求参数

### 查询参数

无。

### 请求头

| 请求头 | 必填 | 说明 |
| --- | --- | --- |
| `user-agent` | 否 | 浏览器 UA，建议带上，非浏览器 UA 可能被拒绝 |
| `referer` | 否 | 来源页，建议带上站点页面地址 |
| `accept` | 否 | 期望的响应类型，默认 `text/plain,*/*` |

### 请求体

`GET` 请求，无请求体，不发送 `content-type`。

## 签名规则

该资源没有签名要求，`quote-token` 之类的请求签名不适用。

## 响应

### 响应头

| 响应头 | 说明 |
| --- | --- |
| `content-type` | 固定 `text/plain` |
| `cache-control` | 站点缓存策略，实测 `max-age=120` |
| `last-modified` | 资源生成时间，说明该文件由站点定期生成 |

响应头不属于业务契约，成败以能否解析出合法日期序列为准。

### 响应体

响应是一段 JS 变量声明：

```javascript
var datelist="LC/AAApNDXCw6mHbaPgkryxXv10eAJP1LW0SD39aT7+NV44Xba3PxCgTdrp5BkYVAc11hWvg0c/...";var KLC_TD_SH=datelist;
```

引号内为 469 个字符的编码串（随数据量增长），不是明文日期，不能按 JSON 或纯文本
日期解析。

### 编码格式

编码把日期序列按位打包，并用游程描述连续区间：

| 部分 | 位宽 | 说明 |
| --- | --- | --- |
| 字符映射 | 6 位/字符 | 按 Base64 字符表 `A-Za-z0-9+/` 映射，低位在前串成位流 |
| 头部校验 | 12 位 + 6 位 | 第二个字段与 `63` 异或得到校验和，只接受 `0` 或 `1` |
| 起始游标 | 18 位 | 解出的值减 1 作为起始日游标 |
| 结束游标 | 18 位 | 循环在游标达到该值时结束 |
| 游程标志 | 1 位 | 为 `1` 时继续读符号位，调整累计游程长度 |
| 游程符号 | 变长 | 首位表示方向（`1` 正、`0` 负），随后连续 `1` 的个数决定增量 |
| 游程长度 | 游程 ×3 位 | 读入本段重复的天数，减 1 后作为实际重复次数 |

日期换算：游标每日加 1，若落在被折叠的周末位置则跳过，最终日期为
`1970-01-01` 加 `7657 + 游标` 天。`7657` 天正好对应 1990-12-19，即上交所开市日，
可作为算法正确性的交叉校验。

### 数据特征（实测）

| 项 | 值 |
| --- | --- |
| 交易日数量 | 8796 |
| 日期范围 | 1990-12-19 至 2026-12-31 |
| 日期粒度 | 日，UTC 零点表示，仅表日历日期 |
| 更新方式 | 站点定期重新生成文件，`last-modified` 指示生成时间 |
| 已知缺口 | 1992-05-04 未收录，需由调用方按需处理，代码不擅自补 |

### 示例

解码后（代码中 `TradeCalendar.Dates`，UTC 零点）：

```
1990-12-19, 1990-12-20, 1990-12-21, 1990-12-24, 1990-12-25, ...
```

查询结果示例：

| 日期 | 星期 | 交易日 |
| --- | --- | --- |
| `2026-11-02` | 周一 | 是 |
| `2026-11-07` | 周六 | 否 |
| `2026-10-01` | 周四（国庆） | 否 |
| `1990-12-19` | 周三（开市首日） | 是 |

## 错误处理

| 情况 | 表现 | 代码返回 |
| --- | --- | --- |
| 正常 | 200，含 `datelist` | 解析成功 |
| 缺少 `datelist` 变量 | 200，内容为 HTML 或其他文本 | `ErrNoData` |
| 编码字符非法或数据提前结束 | 200 | `ErrEncoding` |
| 头部校验和不匹配 | 200 | `ErrChecksum` |
| 非 200 | 403、5XX 等 | `APIError`，带 `HTTPStatus` |
| 网络错误或超时 | 无响应 | `APIError`，`HTTPStatus` 为 0 |
| 上下文取消 | 请求中止 | 原始 `context.Canceled` |

解析失败一律返回错误，不会用空列表或零值日期兜底，避免上游把数据源失效当成
「没有交易日」。

## 相关实现

- `client.go`：配置、请求发送与错误类型
- `decode.go`：编码串提取、位流读取与日期还原
- `calendar.go`：`TradeCalendar` 的查询方法（`Len`、`First`、`Last`、`IsTradeDay`）
- `decode_test.go`：真实样本（8796 个交易日）、校验和、截断、边界游标
- `client_test.go`：请求头、非 200、网络错误、上下文取消、读取失败
- `calendar_test.go`：空日历、首尾日期、时区语义
