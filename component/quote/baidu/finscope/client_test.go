package finscope

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewAppliesDefaults(t *testing.T) {
	client := New(Config{})
	cfg := client.Config()
	if cfg.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q，期望 %q", cfg.BaseURL, DefaultBaseURL)
	}
	if cfg.ConstituentsPath != DefaultConstituentsPath {
		t.Errorf("ConstituentsPath = %q，期望 %q", cfg.ConstituentsPath, DefaultConstituentsPath)
	}
	if cfg.Referer != DefaultReferer {
		t.Errorf("Referer = %q，期望 %q", cfg.Referer, DefaultReferer)
	}
	if cfg.UserAgent != DefaultUserAgent {
		t.Errorf("UserAgent 未取默认值：%q", cfg.UserAgent)
	}
	if cfg.Timeout != DefaultTimeout {
		t.Errorf("Timeout = %v，期望 %v", cfg.Timeout, DefaultTimeout)
	}
}

func TestNewNormalizesBaseURLAndPath(t *testing.T) {
	client := New(Config{BaseURL: "https://example.com/", ConstituentsPath: "sapi/v1/constituents"})
	cfg := client.Config()
	if cfg.BaseURL != "https://example.com" {
		t.Errorf("结尾斜杠未去掉：%q", cfg.BaseURL)
	}
	if cfg.ConstituentsPath != "/sapi/v1/constituents" {
		t.Errorf("路径未补前导斜杠：%q", cfg.ConstituentsPath)
	}
}

func TestClientHeaderOverride(t *testing.T) {
	var gotReferer, gotCustom string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReferer = r.Header.Get("referer")
		gotCustom = r.Header.Get("x-trace-id")
		writeJSON(w, constituentsSample)
	}))
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL,
		// 个股页地址比通用首页更贴近真实浏览，允许覆盖默认来源页。
		Referer: "https://finance.baidu.com/stock/ab-600009",
		Header:  map[string]string{"x-trace-id": "abc"},
	})
	if _, err := client.GetConstituents(context.Background(), defaultRequest()); err != nil {
		t.Fatalf("GetConstituents 返回错误：%v", err)
	}
	if gotReferer != "https://finance.baidu.com/stock/ab-600009" {
		t.Errorf("referer = %q，期望使用自定义来源页", gotReferer)
	}
	if gotCustom != "abc" {
		t.Errorf("扩展请求头未生效：%q", gotCustom)
	}
}

func TestClientNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := server.URL
	server.Close() // 关闭后连接被拒，模拟网络错误。

	client := New(Config{BaseURL: url, Timeout: 2 * time.Second})
	_, err := client.GetConstituents(context.Background(), defaultRequest())
	if err == nil {
		t.Fatal("网络错误应返回错误")
	}
	if IsRiskControl(err) {
		t.Error("网络错误不应被判定为风控")
	}
	if !strings.Contains(err.Error(), "请求失败") {
		t.Errorf("错误信息未说明请求失败：%v", err)
	}
}

func TestClientContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, constituentsSample)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := New(Config{BaseURL: server.URL})
	if _, err := client.GetConstituents(ctx, defaultRequest()); err == nil {
		t.Fatal("ctx 取消应返回错误")
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("  abc  ", 10); got != "abc" {
		t.Errorf("truncate 未去空白：%q", got)
	}
	if got := truncate("abcdef", 3); got != "abc..." {
		t.Errorf("truncate = %q，期望 abc...", got)
	}
}

func TestEncodeQueryKeepsOrder(t *testing.T) {
	params := []Param{{Name: "market", Value: "ab"}, {Name: "code", Value: "000300"}, {Name: "name", Value: "沪深300"}}
	got := encodeQuery(params)
	want := "market=ab&code=000300&name=%E6%B2%AA%E6%B7%B1300"
	if got != want {
		t.Errorf("查询串 = %q，期望 %q", got, want)
	}
}
