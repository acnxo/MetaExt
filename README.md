# MetaExt

[![CICD](https://github.com/acnxo/MetaExt/actions/workflows/CICD.yml/badge.svg?branch=dev)](https://github.com/acnxo/MetaExt/actions/workflows/CICD.yml)

Meta External Impl —— 外部数据接口的 Go 实现集合。

## 定位

本项目是**公共基础库**，供其他 Go 项目以模块依赖的方式引用，目的是复用其中已有的
公共逻辑与代码实现，而不是各自重复写一份。

- 库本身不提供独立服务，`_cmd/starter` 只是可执行示例，供本地验证与调试；
- 下游项目通过 import 引用所需包，只取自己需要的部分，不强制引入全部依赖；
- 包内实现保持通用，不绑定某一个调用方的业务假设、表结构或运行环境。

因此编码时遵循以下约定：

- 对外 API 语义清晰、尽量保持稳定，破坏性变更通过版本号体现；
- 包之间保持单向依赖，不出现循环引用；
- 不隐式读写全局状态，避免下游复用时互相干扰；
- 可配置项通过 `Config` 之类的结构显式传入，零值即合理的默认行为。

## 作为依赖引用

```bash
# 拉取最新版本
go get github.com/acnxo/MetaExt@latest
```

```go
import (
    "github.com/acnxo/MetaExt/util"
    "github.com/acnxo/MetaExt/model"
)
```

仅导入用到的包即可，未引用的包不会进入下游的编译产物。

## 环境要求

- Go 1.25 及以上

## 可执行示例

`_cmd/starter/` 是一个最小可执行示例，用于验证模块能正常构建运行。目录名以下划线开头，
会被 Go 工具链忽略：`go build ./...`、`go list ./...`、`go test ./...` 都不会包含它，
不影响本库作为纯包被下游引用。

```bash
# 直接运行
go run ./_cmd/starter

# 编译到当前目录
go build -o MetaExt ./_cmd/starter

# 注入版本号
go build -ldflags "-X main.version=v0.1.0" -o MetaExt ./_cmd/starter
```

## 目录结构

```
_cmd/
└── starter/            可执行示例，不参与 ./... 构建
component/              具体外部接口的实现，按品类与渠道分层组织
base/                   最底层公共基座，不依赖本项目其他包
model/                  跨层共享的数据模型
sdk/                    外部服务客户端封装
ext/                    对外扩展点的接口契约与注册机制
util/                   无状态通用工具函数
common/                 跨包共享的常量、枚举与轻量值对象
infra/                  配置、日志、存储等基础设施实现与装配
```

## 包职责

| 包 | 职责 |
| --- | --- |
| `component` | 具体外部接口的实现，按品类与渠道分层组织 |
| `base` | 最底层公共基座（时间口径、标识约定、错误包装），不依赖本项目其他包 |
| `model` | 纯数据结构模型，不含网络与存储行为 |
| `sdk` | 外部服务客户端：请求构造、签名、重试与响应解析 |
| `ext` | 扩展点契约与注册机制，运行期替换或增补行为 |
| `util` | 无状态的纯函数工具，不引用 model、sdk、infra |
| `common` | 公共常量、枚举与轻量值对象 |
| `infra` | 配置、日志、存储、缓存等实现，仅由入口层装配引用 |

## 扩展新的接口实现

1. 在 `component/` 下按「品类 → 渠道」新建包，或在 `sdk/` 下按外部服务新建包；
2. 只做协议适配：请求构造、签名、重试与响应解析，把外部格式转换为 `model` 中的结构；
3. 错误按 HTTP 状态与业务码分类，并提供可判定的错误类型；
4. 不承载业务流程，也不直接访问存储；
5. 新增能力优先考虑放进 `util`、`model`、`common` 等公共包，便于其他项目直接复用。

## 持续集成

仓库内置两个 GitHub Actions 工作流，位于 `.github/workflows/`：

| 工作流 | 触发条件 | 内容 |
| --- | --- | --- |
| `CICD` | 推送或 PR 到 `main`、`dev`，或手动触发 | `gofmt` 检查、`go vet`、`go build ./...`、`go test -race` 并上传覆盖率报告 |
| `Release` | 推送 `v*` 标签 | 先跑格式检查、`go vet`、`go test -race`，通过后创建 Release 并自动生成变更说明 |

工作流文件为 `.github/workflows/CICD.yml` 与 `.github/workflows/Release.yml`。CI 使用的
Go 版本从 `go.mod` 读取，不另外固定；覆盖率汇总会写入 Actions 的运行摘要。

## 许可

本项目采用 GPL-3.0 许可，详见 [LICENSE](LICENSE)。
