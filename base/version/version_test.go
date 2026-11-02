package version

import (
	"runtime/debug"
	"testing"
)

// TestVersionNotEmpty 任何环境下都应返回非空版本号。
func TestVersionNotEmpty(t *testing.T) {
	if got := Version(); got == "" {
		t.Fatal("Version 不应返回空串")
	}
}

// TestVersionInjected 注入的版本号优先于构建信息。
func TestVersionInjected(t *testing.T) {
	original := injected
	t.Cleanup(func() { injected = original })

	injected = "v0.1.1"
	if got := Version(); got != "v0.1.1" {
		t.Fatalf("期望注入值 v0.1.1，实际 %s", got)
	}
}

// TestVersionFallback 未注入时回退到构建信息，本地构建通常是 devel。
func TestVersionFallback(t *testing.T) {
	original := injected
	t.Cleanup(func() { injected = original })

	injected = ""
	got := Version()
	if got == "" {
		t.Fatal("回退结果不应为空")
	}
	// 本地 go test 构建下主模块版本为 "(devel)"，应回退为 devel 而不是原样返回。
	if got == "(devel)" {
		t.Fatalf("不应把构建信息的占位版本直接暴露出去，实际 %s", got)
	}
	if got != develVersion {
		t.Logf("当前从构建信息取到版本：%s", got)
	}
}

// TestUsable 构建信息中的占位版本应判为不可用。
func TestUsable(t *testing.T) {
	cases := map[string]bool{
		"":        false,
		"(devel)": false,
		"v0.1.1":  true,
		"v1.0.0":  true,
	}
	for value, want := range cases {
		if got := usable(value); got != want {
			t.Errorf("usable(%q) 期望 %v，实际 %v", value, want, got)
		}
	}
}

// stubBuildInfo 临时替换构建信息来源，并在用例结束后恢复。
func stubBuildInfo(t *testing.T, info *debug.BuildInfo, ok bool) {
	t.Helper()
	original := readBuildInfo
	t.Cleanup(func() { readBuildInfo = original })
	readBuildInfo = func() (*debug.BuildInfo, bool) { return info, ok }
}

// TestVersionFromMainModule 主模块版本可用时直接采用。
func TestVersionFromMainModule(t *testing.T) {
	original := injected
	t.Cleanup(func() { injected = original })
	injected = ""

	stubBuildInfo(t, &debug.BuildInfo{
		Main: debug.Module{Path: modulePath, Version: "v0.3.0"},
	}, true)
	if got := Version(); got != "v0.3.0" {
		t.Fatalf("期望 v0.3.0，实际 %s", got)
	}
}

// TestVersionFromDependency 作为依赖被引入时取依赖版本。
func TestVersionFromDependency(t *testing.T) {
	original := injected
	t.Cleanup(func() { injected = original })
	injected = ""

	stubBuildInfo(t, &debug.BuildInfo{
		Main: debug.Module{Path: "example.com/app", Version: "v1.0.0"},
		Deps: []*debug.Module{
			{Path: "example.com/other", Version: "v2.0.0"},
			{Path: modulePath, Version: "v0.1.1"},
		},
	}, true)
	if got := Version(); got != "v0.1.1" {
		t.Fatalf("期望 v0.1.1，实际 %s", got)
	}
}

// TestVersionNoUsableInfo 构建信息里没有可用版本时返回 devel。
func TestVersionNoUsableInfo(t *testing.T) {
	original := injected
	t.Cleanup(func() { injected = original })
	injected = ""

	stubBuildInfo(t, &debug.BuildInfo{
		Main: debug.Module{Path: "example.com/app", Version: "(devel)"},
	}, true)
	if got := Version(); got != develVersion {
		t.Fatalf("期望 %s，实际 %s", develVersion, got)
	}
}

// TestVersionBuildInfoUnavailable 拿不到构建信息时返回 devel。
func TestVersionBuildInfoUnavailable(t *testing.T) {
	original := injected
	t.Cleanup(func() { injected = original })
	injected = ""

	stubBuildInfo(t, nil, false)
	if got := Version(); got != develVersion {
		t.Fatalf("期望 %s，实际 %s", develVersion, got)
	}
}
