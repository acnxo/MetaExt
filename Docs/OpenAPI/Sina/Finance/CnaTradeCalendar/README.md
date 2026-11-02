# Sina Finance CnaTradeCalendar 接口规格

新浪财经 A 股交易日历的接口规格与说明。数据源是一个静态文本资源，返回站点自定义
编码的日期序列，需要解码后才能得到交易日。

## 文件

| 文件 | 位置 | 说明 |
| --- | --- | --- |
| `GetTradeCalendar.yaml` | `Meta/OpenAPI/Sina/Finance/CnaTradeCalendar/` | `GET /realstock/company/klc_td_sh.txt` 的 OpenAPI 规格 |
| `GetTradeCalendar.md` | 本目录 | 同一接口的可读说明，含解码步骤与数据特征 |

两份文件同名成对：`Meta/` 放机器可读规格，`Docs/` 放人读说明。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://finance.sina.com.cn` |
| 路径 | `/realstock/company/klc_td_sh.txt` |
| operationId | `getTradeCalendar` |
| 代码入口 | `cnatradecalendar.Client.GetTradeCalendar`、`cnatradecalendar.Client.GetCNATradeCalendar` |
| 响应类型 | `text/plain`（非 JSON） |

## 对外建议

优先使用 `GetCNATradeCalendar`：它在数据源内容上补齐了缺失的 `1992-05-04`，
返回 8797 个交易日，可直接用于「某天是否交易日」的判断。需要核对数据源原始状态时
再用 `GetTradeCalendar`（8796 个交易日，保留缺口）。

## 格式要点

响应体形如：

```javascript
var datelist="LC/AAApNDXCw6mHbaPgkryxXv10eAJP1LW0SD39...";var KLC_TD_SH=datelist;
```

- 引号内是 469 个字符的编码串，不是明文日期；
- 每个字符按 Base64 字符表映射为 6 位，低位在前串成位流；
- 位流依次是：12 位 + 6 位头部（含校验和）、两个 18 位游标、若干游程段；
- 游程段描述连续交易日区间，用变长的符号位调整游程长度；
- 日期基准是 `1970-01-01 + 7657 天`，即 1990-12-19（上交所开市日）。

完整解码步骤见 [GetTradeCalendar.md](GetTradeCalendar.md)。

## 数据特征（实测）

| 项 | 值 |
| --- | --- |
| 交易日数量 | 8796 |
| 日期范围 | 1990-12-19 至 2026-12-31 |
| 缓存策略 | `cache-control: max-age=120`，`last-modified` 指示生成时间 |
| 已知缺口 | 1992-05-04 未收录；`GetTradeCalendar` 如实返回，`GetCNATradeCalendar` 已补齐 |

## 实现与测试

`component/quote/sina/finance/cnatradecalendar` 包，纯标准库，文件职责：

| 文件 | 职责 |
| --- | --- |
| `client.go` | 配置、HTTP 请求、`APIError` 与 `ErrNoData` |
| `decode.go` | 编码串提取、位流读取、日期还原 |
| `calendar.go` | `TradeCalendar` 与 `IsTradeDay` 等查询方法 |
| `cna.go` | `GetCNATradeCalendar`、缺口补齐、两市开市日 |
| `decode_test.go` | 真实样本解析、校验和、截断、边界游标 |
| `client_test.go` | 请求头、非 200、网络错误、上下文取消、读取失败 |
| `calendar_test.go` | 空日历、首尾日期、时区语义 |
| `cna_test.go` | 缺口补齐、幂等性、不改入参、开市日 |

单包覆盖率 100%，测试不依赖网络：样本是抓取到的真实文本，内嵌在测试中。

## 风险

- 该资源不是厂商文档化的 API，编码格式可能随站点调整；
- 校验和不匹配、数据提前结束时直接报错，不返回部分结果；
- 数据源存在缺口（如 1992-05-04）：`GetTradeCalendar` 如实返回，`GetCNATradeCalendar`
  会补齐，对外建议用后者；
- 本资源是沪市日历，起点 `1990-12-19`；深市起点为 `1991-01-02`，两市早期交易日
  不一致，判断深市标的需另行处理；
- 站点按 `max-age=120` 缓存，调用方如需长期持有日历，建议自行加缓存并定期刷新。
