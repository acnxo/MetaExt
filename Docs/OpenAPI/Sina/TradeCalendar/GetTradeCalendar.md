# GetTradeCalendar（占位）

获取 A 股交易日历。机器可读规格见
[Meta/OpenAPI/Sina/TradeCalendar/GetTradeCalendar.yaml](../../../../Meta/OpenAPI/Sina/TradeCalendar/GetTradeCalendar.yaml)，
当前为占位骨架，尚未实测确认。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://finance.sina.com.cn` |
| 路径 | `/realstock/company/klc_td_sh.txt` |
| operationId | `getTradeCalendar` |
| 代码入口 | `sdk/sina/tradecalendar`（当前只有包注释占位，未实现） |
| 响应类型 | `text/plain`，非 JSON |

## 已知情况

- HTTP 200，`content-type: text/plain`，`cache-control: max-age=120`；
- 响应体很小（约 500 字节），形如 `var datelist="...";var KLC_TD_SH=datelist;`；
- 引号内为站点自定义编码的字符串，需解码后才能得到日期序列；
- 响应头带 `last-modified`，说明由站点定期生成。

## 待确认

| 项 | 状态 |
| --- | --- |
| 解码算法与边界处理 | 未确认 |
| 是否需要 `referer` / `user-agent` | 未确认 |
| 日期覆盖范围、是否含未来交易日 | 未确认 |
| 非 200 与格式异常时的表现 | 未确认 |

## 说明

本文件与同目录 `README.md` 都处于占位状态：`README.md` 记录目录级的调研结论与实现前
准备，本文件对应单个接口。等实测确认后再补齐请求头、响应字段、示例与解码步骤。
