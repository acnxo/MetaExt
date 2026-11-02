# 提交信息规范

本规范约定 Git 提交信息（commit message）的写法，适用于本仓库的所有提交，包括
本地提交、PR 中的提交与合并提交。目的是让 `git log` 能直接回答「这次改了什么、
为什么改、影响哪些范围」，而不是靠翻 diff 猜。

与 PR 标题的关系：PR 标题沿用同一格式，合并方式为 squash 时以 PR 标题为准。

## 一、总则

- 采用 Conventional Commits 的 `type(scope): 摘要` 结构，摘要与正文使用简体中文；
- 一个提交只做一件事，能用一个 type 概括；跨多个主题的改动拆成多个提交；
- 提交信息描述**结果与动机**，不复述 diff，不写「修改代码」「更新文件」这类无信息量的话；
- 不提交「临时保存」「wip」「fix typo」这类草稿信息；
- 提交前自查：`git status` 无多余文件、`gofmt`、`go vet`、`go test ./...` 通过。

## 二、结构

```
<type>(<scope>): <摘要>

<正文，说明动机与做法，可多段>

<脚注，如 BREAKING CHANGE、Refs>
```

| 部分 | 要求 |
| --- | --- |
| 标题行 | 不超过 72 个字符，摘要不超过 50 个字，不以句号结尾 |
| 标题与正文 | 之间空一行 |
| 正文 | 每行不超过 72 个字符，可用 `-` 列表 |
| 脚注 | 与正文空一行，`BREAKING CHANGE:` 与 `Refs:` 各占一行 |

## 三、类型

| type | 用途 | 示例摘要 |
| --- | --- | --- |
| `feat` | 新增功能或接口能力 | `feat(tradecalendar): 新增交易日历按周对齐输出` |
| `fix` | 修复缺陷或错误行为 | `fix(quoteapiv2): 修正 POST 签名使用请求体字节` |
| `docs` | 仅文档变更 | `docs(devspec): 新增提交信息规范` |
| `test` | 仅测试变更 | `test(tradecalendar): 补充校验和异常用例` |
| `refactor` | 不改变外部行为的重构 | `refactor(quoteapiv2): 按接口拆分源文件` |
| `perf` | 性能优化 | `perf(cashtrend): 减少重复解码` |
| `style` | 格式调整，不改逻辑 | `style: 统一表格列宽` |
| `chore` | 构建、依赖、工具、杂项 | `chore(module): 初始化 Go 模块` |
| `ci` | CI/CD 工作流与脚本 | `ci: 修正 setup-go 缓存键` |

选择规则：

- 同时涉及代码与文档时，以代码变更的 type 为准，文档写进正文；
- 只有测试变更时用 `test`，不要用 `feat` 夹杂；
- 不确定时用能最小描述主要影响的那个，不要造新 type。

## 四、scope

- 取包名、模块名或目录名，用小写，不加路径；
- 常用取值：`quoteapiv2`、`tradecalendar`、`devspec`、`metadata`、`ci`、`module`；
- 跨包改动可省略 scope，写成 `feat: 新增行情接口基类`；
- 不使用中文 scope，也不把多级路径塞进 scope。

## 五、摘要

- 用祈使或陈述语气说明「做了什么」，不写「我」「本提交」；
- 具体的名词优先：写「新增 A 股交易日历客户端」，不写「新增功能」；
- 涉及数量、接口名时写清楚，如「新增七个行情接口实现」；
- 结尾不加句号，不与 type 重复，例如 `feat: 新增 feat` 这类废话；
- 中文与英文之间不留空格以外的特殊处理，技术名词保持原文大小写。

## 六、正文

正文说明动机与影响，按需使用以下顺序：

1. **为什么改**：现状的问题、触发原因、需求来源；
2. **改了什么**：关键改动点，用 `-` 列表逐条列出；
3. **影响面**：行为变化、兼容性、对调用方的要求。

不写的内容：

- 逐行解释 diff；
- 与本次改动无关的设想与待办（这些放 PR 描述或 issue）；
- 未经验证的结论，如「应该没问题」。

## 七、脚注

| 脚注 | 用途 | 写法 |
| --- | --- | --- |
| `BREAKING CHANGE` | 不兼容变更 | `BREAKING CHANGE: 说明受影响的范围与迁移方式` |
| `Refs` | 关联 issue 或 PR | `Refs: #6` |
| `Co-authored-by` | 协作署名 | `Co-authored-by: 名称 <邮箱>` |

只要存在不兼容变更，无论 type 是什么，都必须写 `BREAKING CHANGE` 脚注，并在正文说明
迁移方式。是否升主版本按发版规范判断。

## 八、示例

正确示例：

```
feat(tradecalendar): 新增新浪 A 股交易日历客户端

数据源是 klc_td_sh.txt 静态文本资源，返回站点自定义编码的日期序列，
需按位流解码后才能得到交易日列表。

- 新增 client.go：配置、请求发送与错误类型
- 新增 decode.go：位流读取与日期还原
- 新增 calendar.go：IsTradeDay 等查询方法

实测覆盖 8796 个交易日，单包覆盖率 100%。
```

```
fix(quoteapiv2): 修正 POST 签名使用请求体字节

原先对查询串签名，服务端校验失败。改为对紧凑 JSON 请求体的原始字节签名，
与站点前端行为一致，并补充签名向量测试。

Refs: #2
```

不兼容变更示例：

```
refactor(tradecalendar): 把日期类型改为自定义 Date

BREAKING CHANGE: TradeCalendar.Dates 由 []time.Time 改为 []Date，
调用方需把 time.Time 转成 Date 或改用 IsTradeDay。
Refs: #6
```

错误示例与问题：

| 提交信息 | 问题 |
| --- | --- |
| `更新` | 无 type、无内容 |
| `修复 bug` | 没说修什么、影响哪里 |
| `feat: 新增功能并修复若干问题` | 一个提交多主题，应拆分 |
| `feat(quoteapiv2): 新增接口。` | 摘要以句号结尾 |
| `Feat(TradeCalendar): 新增接口` | type 与 scope 大小写不符 |
| `feat: 新增接口\n\n- 改了 client.go` | 正文复述 diff，没有动机 |

## 九、检查清单

提交前逐项确认：

- [ ] 标题符合 `type(scope): 摘要`，type 在允许列表内
- [ ] 摘要为中文、不超过 50 字、不以句号结尾
- [ ] 标题与正文间空一行，正文每行不超过 72 个字符
- [ ] 一个提交只做一件事，无夹带的无关文件
- [ ] 不兼容变更已写 `BREAKING CHANGE` 脚注并说明迁移方式
- [ ] 关联 issue 或 PR 已写 `Refs`
- [ ] `gofmt`、`go vet`、`go test ./...` 通过
- [ ] 提交信息不使用「临时」「wip」「fix typo」等草稿措辞
