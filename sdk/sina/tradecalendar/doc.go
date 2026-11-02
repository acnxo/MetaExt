// Package tradecalendar 预留新浪财经 A 股交易日历接口的客户端。
//
// 接口地址：https://finance.sina.com.cn/realstock/company/klc_td_sh.txt
//
// 该地址返回的是一段 JS 变量赋值文本，内容为站点自定义编码的日期序列，
// 需要先解码才能得到交易日列表，不能当作 JSON 或纯文本日期处理。
//
// 当前仅为占位，尚未实现：
//
//   - 文本格式与解码算法待实测确认；
//   - 请求头要求（referer、user-agent）与缓存策略待验证；
//   - 日期覆盖范围、更新频率、异常响应的表现待记录。
//
// 实现时按 Docs/DevSpec/HTTPAPIDocSpec.md 的约定，同步补齐 OpenAPI 规格、
// 接口说明与单元测试，不要只提交代码。
package tradecalendar
