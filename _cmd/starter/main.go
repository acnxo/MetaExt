package main

import (
	"fmt"
	"runtime"

	"github.com/acnxo/MetaExt/base/version"
)

func main() {
	// 版本号来自根目录 VERSION 文件，构建时由 CI 注入；拿不到时回退为 devel。
	fmt.Printf("MetaExt %s (%s, %s/%s)\n",
		version.Version(), runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
