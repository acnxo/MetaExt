# TradeCalendar 接口规格（占位）

本目录预留给新浪财经 A 股交易日历接口的规格说明，目前只有占位，尚未编写 OpenAPI 规格
与接口说明。等实测确认响应格式后再补 `GetTradeCalendar.yaml` 与 `GetTradeCalendar.md`。

## 接口

| 项 | 内容 |
| --- | --- |
| 方法 | `GET` |
| 站点 | `https://finance.sina.com.cn` |
| 路径 | `/realstock/company/klc_td_sh.txt` |
| 响应类型 | `text/plain`（非 JSON） |
| 建议方法名 | `GetTradeCalendar`（对应代码包 `sdk/sina/tradecalendar`） |

## 已确认的事实

对接口请求过一次，只记录可复核的客观现象：

- HTTP 状态 200，`content-type: text/plain`，`cache-control: max-age=120`；
- 响应体很小（约 500 字节），形如 `var datelist="...";var KLC_TD_SH=datelist;`；
- 引号内是站点自定义编码的字符串，内容不是明文日期，需要解码后才能得到日期序列；
- 响应头带 `last-modified`，说明该文件由站点定期生成，更新时间不确定。

## 待确认

| 项 | 状态 |
| --- | --- |
| 解码算法（编码方式、是否差分编码、边界处理） | 未确认 |
| 请求头是否必须带 `referer` / `user-agent` | 未确认 |
| 日期覆盖范围（起始日期、是否含未来交易日） | 未确认 |
| 文件更新频率与失效时的表现 | 未确认 |
| 是否存在其它等价接口 | 未确认 |

## 实现前的准备

1. 实测抓取一次样本，确认编码格式与解码步骤；
2. 按 `Docs/DevSpec/HTTPAPIDocSpec.md` 编写 `Meta/OpenAPI/Sina/TradeCalendar/GetTradeCalendar.yaml`
   与本目录下的 `GetTradeCalendar.md`，两者同名；
3. 在 `sdk/sina/tradecalendar` 中实现客户端，解析逻辑单独成文件，便于数据源替换；
4. 用 `httptest` 覆盖正常解析、格式异常、非 200、上下文取消与空结果；
5. 结果建议做缓存（站点侧 `max-age` 为 120 秒，全量日期不常变）。

## 风险

- 该资源不是厂商文档化的 API，格式可能随时变化；
- 解析失败必须返回明确错误，不能用空列表或零值日期兜底，否则调用方难以发现数据源已失效。
