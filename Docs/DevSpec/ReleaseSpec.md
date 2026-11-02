# 版本与发版规范

本规范约定本库的版本号、变更日志、发版流程与兼容策略。本库面向第三方依赖，
发版必须可预期、可追溯：下游能否安全升级，取决于这里写的规则是否被遵守。

## 一、版本号

采用语义化版本 `MAJOR.MINOR.PATCH`，格式为 `vX.Y.Z`（tag 带 `v` 前缀）。

| 阶段 | 版本形态 | 约定 |
| --- | --- | --- |
| 初始开发 | `v0.y.z` | 公开 API 仍在成形，**不承诺向后兼容**；`y` 递增表示新增能力或调整接口 |
| 稳定阶段 | `v1.y.z` 及以上 | `X`：不兼容变更；`Y`：向后兼容的新增；`Z`：向后兼容的修复 |
| 预发布 | `vX.Y.Z-rc.1`、`vX.Y.Z-beta.1` | 用于发布前验证，不保证与正式版兼容 |

补充规则：

- 一旦发布 `v1.0.0`，删除或改变已导出的标识符、改变函数签名与结构体字段语义，
  都必须升 `MAJOR`；
- `PATCH` 只用于修复，不引入新导出标识符，也不改变已有行为；
- 版本号只由 `main` 上的 tag 决定，`go.mod` 里不写版本号。

### Go 模块路径规则

- `v0` 与 `v1` 的模块路径保持不变：`github.com/acnxo/MetaExt`；
- 若将来发布 `v2` 及以上，模块路径必须加 `/v2` 后缀（或在主分支之外维护版本分支），
  这是 Go 的硬性要求，届时需要单独评估；
- 每次发版后确认 `go get github.com/acnxo/MetaExt@vX.Y.Z` 可以正常拉取。

## 二、分支与合并

| 分支 | 用途 | 规则 |
| --- | --- | --- |
| `main` | 仅存放已发布或待发布的稳定代码 | 只接受 PR；每次发布都从 `main` 打 tag |
| `dev` | 日常集成分支 | 可自由提交，合并到 `main` 需走 PR |
| `feat/*`、`fix/*`、`docs/*`、`chore/*` | 单个变更 | 从 `dev` 或 `main` 拉出，完成后提 PR |

要求：

- 任何进入 `main` 的变更都必须经过 PR 与 `CICD` 工作流；
- 不在 `main` 上直接提交，不在 `main` 上强推；
- 发布后不再修改已推送的 tag，需要修正时发新的 `PATCH` 版本。

## 三、变更日志

仓库根目录维护 `CHANGELOG.md`，格式参考 Keep a Changelog：

- 顶部固定保留 `## [Unreleased]` 段，日常变更先写在这里；
- 发版时把 `Unreleased` 内容移入新版本段，并补上发布日期；
- 分类固定为：`Added`（新增）、`Changed`（变更）、`Deprecated`（弃用）、
  `Removed`（移除）、`Fixed`（修复）、`Security`（安全）；
- 每条变更写清影响面，涉及不兼容时在条目末尾标注 `BREAKING`；
- 版本段的顺序与 tag 保持一致，最新的在上。

## 四、发版流程

按顺序执行，每一步都不可跳过：

1. **确认基线**：`main` 上的 `CICD` 运行成功，本地 `git status` 干净；
2. **更新日志**：把 `CHANGELOG.md` 的 `Unreleased` 内容整理为新版本段，
   标注日期与版本号；如涉及不兼容变更，显式写 `BREAKING`；
3. **同步文档**：README 中的版本示例、接口说明与规格若有变化，一并更新；
4. **提交准备**：提交信息用 `chore(release): 准备 vX.Y.Z`；
5. **打附注 tag**：`git tag -a vX.Y.Z -m "vX.Y.Z：<一句话摘要>"`，指向 `main` 的提交；
6. **推送 tag**：`git push origin vX.Y.Z`，不要推送本地未评审的其它分支；
7. **自动发布**：`Release` 工作流会先跑格式检查、`go vet`、`go test -race`，
   通过后用 tag 创建 Release 并生成变更说明；
8. **发布后核验**：确认 Release 页面存在、内容正确，并验证
   `go get github.com/acnxo/MetaExt@vX.Y.Z` 能拉取到该版本。

发布失败时：若 tag 还没推送，删除本地 tag 重新来；若已推送，不删除远端 tag，
修正问题后发新的 `PATCH` 版本。

## 五、兼容与弃用

- 导出标识符的删除、重命名、签名变化、语义反转都属于不兼容变更，必须体现在版本号上；
- 弃用先标注再移除：给标识符加 `// Deprecated: 说明替代方案与移除时间` 注释，
  至少保留一个次版本后再删除；
- 不兼容变更必须在 `CHANGELOG.md` 与 Release 说明中显著标注，让下游在升级前能看到；
- 结构体新增字段尽量放在末尾，且零值保持原有语义，减少下游的隐式破坏。

## 六、发版检查清单

- [ ] `main` 上 `CICD` 为 success，工作区干净
- [ ] `CHANGELOG.md` 已补新版本段，日期与版本号正确
- [ ] 不兼容变更已标注 `BREAKING`
- [ ] 导出 API 的注释、README、OpenAPI 规格与实现一致
- [ ] `gofmt`、`go vet`、`go test ./...` 本地通过
- [ ] tag 为附注 tag，指向 `main` 的提交，消息包含一句话摘要
- [ ] tag 推送后 `Release` 工作流成功，Release 页面内容正确
- [ ] `go get github.com/acnxo/MetaExt@vX.Y.Z` 可拉取

## 七、下游如何使用版本

```bash
# 固定到某个版本（推荐给生产环境）
go get github.com/acnxo/MetaExt@v0.1.0

# 跟随最新版本（仅适合尝鲜）
go get github.com/acnxo/MetaExt@latest
```

- 没有可用 tag 时，Go 会解析出伪版本（如 `v0.0.0-20261102170644-cca0669`），
  这种形式不适合第三方生产依赖，应推动尽快打正式 tag；
- 下游应锁定版本并在升级前查看 `CHANGELOG.md` 与 Release 说明。
