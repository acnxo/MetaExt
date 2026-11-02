// Package version 提供本库版本号的读取能力。
//
// 版本号的唯一来源是仓库根目录的 VERSION 文件。构建时由 CI 通过
// `-ldflags "-X github.com/acnxo/MetaExt/base/version.injected=X.Y.Z"` 注入；
// 未注入时回退到 Go 构建信息中的模块版本，作为第三方依赖引入时通常就是实际 tag。
//
// 该包只读、无副作用，可安全地在任意进程与并发场景下调用。
package version

import (
	"runtime/debug"
	"strings"
)

// modulePath 是本库的模块路径，用于在构建信息中定位自身。
const modulePath = "github.com/acnxo/MetaExt"

// develVersion 是拿不到版本号时的返回值，避免调用方拿到空串。
const develVersion = "devel"

// injected 由构建参数写入，保持未导出，避免运行时被外部包修改。
var injected string

// readBuildInfo 指向 debug.ReadBuildInfo，独立成变量以便测试替换。
var readBuildInfo = debug.ReadBuildInfo

// Version 返回本库版本号，形如 0.1.1，不带 v 前缀。
//
// 取值顺序：
//
//  1. 构建时通过 -ldflags 注入的版本号（取自 VERSION 文件）；
//  2. Go 构建信息中本模块的版本（作为依赖被引入时为实际 tag）；
//  3. 都拿不到时返回 "devel"。
//
// 无论来源如何，返回值都不带 v 前缀，与 VERSION 文件保持一致；
// 对应的 git 标签则是 v 加该版本号（如 0.1.1 对应 v0.1.1）。
func Version() string {
	if injected != "" {
		return normalize(injected)
	}
	info, ok := readBuildInfo()
	if !ok {
		return develVersion
	}
	if info.Main.Path == modulePath && usable(info.Main.Version) {
		return normalize(info.Main.Version)
	}
	for _, dep := range info.Deps {
		if dep.Path == modulePath && usable(dep.Version) {
			return normalize(dep.Version)
		}
	}
	return develVersion
}

// normalize 去掉版本号的 v 前缀，保证不同来源的返回值格式一致。
//
// VERSION 文件与注入值不带前缀，而 Go 模块版本总是带 v，这里统一成前者。
func normalize(v string) string {
	return strings.TrimPrefix(v, "v")
}

// usable 判断构建信息里的版本号是否为可展示的真实版本。
//
// 主模块在本地构建时版本为 "(devel)"，此时应当回退。
func usable(v string) bool {
	return v != "" && v != "(devel)"
}
