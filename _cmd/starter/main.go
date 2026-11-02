package main

import (
	"fmt"
	"runtime"
)

// version 为程序版本号，可在构建时通过 -ldflags 注入。
var version = "dev"

func main() {
	fmt.Printf("MetaExt %s (%s, %s/%s)\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
