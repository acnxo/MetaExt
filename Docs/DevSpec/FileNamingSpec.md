# 文件与目录命名规范

本规范约定仓库内文件、目录的命名方式，覆盖 Go 源码、文档、接口规格与 CI 工作流。
目的是让路径本身能说明内容，并避免跨平台（尤其大小写敏感的文件系统）出现歧义。

已有的命名约定（如接口规格与说明同名、工作流文件大驼峰）以本文为准；新增文件前先对照
本表，不要凭手感命名。

## 一、总则

- 文件名与目录名一律使用 **ASCII**：不使用中文、空格、括号等字符；
- 不使用仅靠大小写区分的名称（如 `Client.go` 与 `client.go` 同时存在），
  Linux 下是两个文件、Windows 下会冲突；
- 同一目录下同一用途的文件命名风格必须一致，不混用多种风格；
- 名称要能看出内容，避免 `util2.go`、`temp.md`、`new.yaml` 这类无语义的命名。

## 二、Go 代码

### 2.1 包目录

| 规则 | 说明 | 示例 |
| --- | --- | --- |
| 全小写 | 不使用大写字母 | `quoteapiv2`、`tradecalendar` |
| 不使用下划线与连字符 | 多词直接连写 | `tradecalendar`，不是 `trade_calendar` |
| 简短、单数 | 一个目录一个包，名称为该包的用途 | `sdk/github`、`component/security` |
| 分层表达归属 | 按「品类 → 渠道 → 接口组」或「服务 → 子能力」组织 | `component/quote/moomoo/quoteapiv2` |

包名与目录名保持一致（最后一级目录名即包名）。

### 2.2 源文件

| 规则 | 示例 |
| --- | --- |
| 全小写，单词间用下划线 | `client.go`、`signature.go` |
| 不使用大驼峰，避免与类型名、导出标识符混淆 | `Client.go` ✗ |
| 按职责拆分，文件名即职责 | `client.go`、`signature.go`、`parse.go` |
| 包文档可单独成文件 | `doc.go` |
| 一个文件只放一个主题 | 签名不要与请求发送混在 `client.go` |

### 2.3 测试文件

| 规则 | 示例 |
| --- | --- |
| 与被测文件同名，加 `_test.go` | `client.go` → `client_test.go`、`signature.go` → `signature_test.go` |
| 跨文件的行为测试可归入主文件的测试 | 接口装配、错误分类放 `client_test.go` |
| 不使用 `*_test2.go`、`test_*.go` 这类变体 | — |

### 2.4 可执行示例

以下划线开头的目录会被 Go 工具链忽略（不参与 `./...` 的构建、测试与 `go list`），
用于放置示例程序，避免公共库中出现 `main` 包：

| 目录 | 用途 |
| --- | --- |
| `_cmd/starter/` | 最小可执行示例，验证模块能构建运行 |
| `_examples/` | 其他按需添加的示例目录 |

## 三、文档

### 3.1 开发规范文档

| 规则 | 示例 |
| --- | --- |
| 大驼峰（UpperCamelCase），以 `Spec.md` 结尾 | `HTTPAPIDocSpec.md`、`FileNamingSpec.md` |
| 一份规范一个文件，主题写进文件名 | 不写成 `spec1.md`、`规范.md` |

### 3.2 接口规格与说明

| 类型 | 文件名规则 | 示例 |
| --- | --- | --- |
| OpenAPI 规格 | 与 Go 方法名、`operationId` 一致的大驼峰 + `.yaml` | `GetKLine.yaml` |
| 接口说明 | 与规格同名 + `.md` | `GetKLine.md` |
| 跨接口说明 | 主题名大驼峰 + `.md` | `Signature.md` |
| 目录索引 | 固定 `README.md` | `README.md` |

厂商、分类与接口组目录同样用大驼峰，层级与 Go 包目录一一对应：

```
# 代码 component/quote/moomoo/quoteapiv2
Meta/OpenAPI/MooMoo/QuoteAPIV2/GetKLine.yaml
Docs/OpenAPI/MooMoo/QuoteAPIV2/GetKLine.md

# 代码 component/quote/sina/finance/cnatradecalendar
Meta/OpenAPI/Sina/Finance/CnaTradeCalendar/GetTradeCalendar.yaml
Docs/OpenAPI/Sina/Finance/CnaTradeCalendar/GetTradeCalendar.md
```

完整路径约定见 [`HTTPAPIDocSpec.md`](HTTPAPIDocSpec.md) 第九节。

### 3.3 仓库级文档

| 文件 | 位置 | 说明 |
| --- | --- | --- |
| `README.md` | 仓库根、各文档目录 | 目录索引 |
| `LICENSE` | 仓库根 | 许可证，保持大写 |
| `AGENTS.md` | 仓库根（如需要） | 项目级代理约定 |

## 四、CI/CD 工作流

| 规则 | 示例 |
| --- | --- |
| 文件名大驼峰 + `.yml` | `CICD.yml`、`Release.yml` |
| 工作流 `name` 与文件名一致 | `name: CICD`、`name: Release` |
| 一个工作流一个文件，按用途命名 | 不写成 `build.yml`、`test.yml` 这类与内容不匹配的名称 |

## 五、其他

- 生成的产物（覆盖率、二进制、临时文件）不提交，通过 `.gitignore` 排除；
- 临时文件、样本数据不要放进仓库目录，必要时放 `/tmp` 或使用下划线前缀目录并加入忽略；
- 新增目录时同步更新对应的 `README.md` 索引，避免出现「有文件没入口」。

## 六、检查清单

- [ ] Go 目录与文件全部小写，无下划线目录、无驼峰文件名
- [ ] 测试文件与被测文件同名，仅多 `_test.go`
- [ ] 接口规格与说明同名，`yaml` 在 `Meta/`、`md` 在 `Docs/`
- [ ] 规范文档以 `Spec.md` 结尾且为大驼峰
- [ ] 工作流文件名与 `name` 均为大驼峰且一致
- [ ] 名称仅靠大小写无法区分的文件不存在
- [ ] 新增目录已在对应 `README.md` 中登记
