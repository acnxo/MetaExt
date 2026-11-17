# 新浪财经 K 线接口

本项目用到的新浪财经端点：长历史日线，用于基准指数。站点为
`money.finance.sina.com.cn`，按站点名归为 `Finance` 组下的 `KLine`。

## 文件

| 文件 | 接口 | 用途 |
| --- | --- | --- |
| [GetKLine.md](GetKLine.md) | `GetKLine` | 国证 2000（`sz399303`）等基准的日收盘 |

## 公共约定

- 必须带 `Referer: https://finance.sina.com.cn/`；
- 返回 JSON 数组，数值字段以**字符串**形式给出，需显式转换；
- **不返回成交额**，需要成交额时应改用雪球或东财日线；
- `volume` 原生单位是**股**（东财/腾讯是「手」）。

## 代码入口

- `component/quote/sina/finance/kline/client.go`：配置、`APIError`
- `component/quote/sina/finance/kline/kline.go`：`GetKLine`、行解析与数组校验
- 同目录上级的 `cnatradecalendar` 是新浪的另一组接口（交易日历），互不影响
