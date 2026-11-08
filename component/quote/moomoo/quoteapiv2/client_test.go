package quoteapiv2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fixedNow 固定时钟，便于断言时间戳与签名。
func fixedNow() time.Time { return time.UnixMilli(1780507278122) }

// newTestClient 创建一个指向测试服务器的客户端。
func newTestClient(t *testing.T, server *httptest.Server, mutate func(*Config)) *Client {
	t.Helper()
	cfg := Config{BaseURL: server.URL, CSRF: "abcdef0123", Now: fixedNow}
	if mutate != nil {
		mutate(&cfg)
	}
	return New(cfg)
}

// splitQuery 按出现顺序拆分查询串，返回键序与键值表。
func splitQuery(raw string) ([]string, map[string]string) {
	order := make([]string, 0, 8)
	values := make(map[string]string, 8)
	for _, pair := range strings.Split(raw, "&") {
		kv := strings.SplitN(pair, "=", 2)
		order = append(order, kv[0])
		if len(kv) == 2 {
			values[kv[0]] = kv[1]
		}
	}
	return order, values
}

// TestConfigDefaults 零值配置应补齐为协议默认值，并规整 URL 与各接口路径。
func TestConfigDefaults(t *testing.T) {
	cfg := New(Config{}).Config()
	if cfg.BaseURL != DefaultBaseURL {
		t.Fatalf("BaseURL 期望 %s，实际 %s", DefaultBaseURL, cfg.BaseURL)
	}
	if cfg.KLinePath != DefaultKLinePath {
		t.Fatalf("KLinePath 期望 %s，实际 %s", DefaultKLinePath, cfg.KLinePath)
	}
	if cfg.StockListPath != DefaultStockListPath {
		t.Fatalf("StockListPath 期望 %s，实际 %s", DefaultStockListPath, cfg.StockListPath)
	}
	if cfg.Locale != DefaultLocale {
		t.Fatalf("Locale 期望 %s，实际 %s", DefaultLocale, cfg.Locale)
	}
	if cfg.Timeout != DefaultTimeout {
		t.Fatalf("Timeout 期望 %s，实际 %s", DefaultTimeout, cfg.Timeout)
	}
	if cfg.CSRF == "" {
		t.Fatal("CSRF 应有占位值")
	}
	if cfg.Now == nil {
		t.Fatal("Now 应补默认时钟")
	}

	custom := New(Config{
		BaseURL:       "https://example.com/",
		KLinePath:     "quote/get-kline",
		StockListPath: "quote/get-stock-list",
	}).Config()
	if custom.BaseURL != "https://example.com" {
		t.Fatalf("BaseURL 结尾斜杠应被去掉，实际 %s", custom.BaseURL)
	}
	if custom.KLinePath != "/quote/get-kline" {
		t.Fatalf("KLinePath 应补前导斜杠，实际 %s", custom.KLinePath)
	}
	if custom.StockListPath != "/quote/get-stock-list" {
		t.Fatalf("StockListPath 应补前导斜杠，实际 %s", custom.StockListPath)
	}
}

// TestAPIErrorError 错误信息应按「业务码 → HTTP 状态 → 原因」的优先级输出。
func TestAPIErrorError(t *testing.T) {
	cases := []struct {
		name string
		err  *APIError
		want string
	}{
		{
			name: "业务码优先",
			err:  &APIError{Path: "/p", URL: "http://x/p", Code: CodeTokenInvalid, Message: "非法请求", HTTPStatus: http.StatusBadRequest},
			want: "code=-403 message=非法请求",
		},
		{
			name: "退到 HTTP 状态",
			err:  &APIError{Path: "/p", URL: "http://x/p", HTTPStatus: http.StatusBadGateway},
			want: "HTTP 错误：502",
		},
		{
			name: "退到原因",
			err:  &APIError{Path: "/p", URL: "http://x/p", Message: "响应解析失败"},
			want: "响应解析失败",
		},
		{
			name: "无附加信息",
			err:  &APIError{Path: "/p", URL: "http://x/p"},
			want: "请求失败",
		},
	}
	for _, tc := range cases {
		if got := tc.err.Error(); !strings.Contains(got, tc.want) {
			t.Errorf("%s：错误信息 %q 应包含 %q", tc.name, got, tc.want)
		}
	}
}

// TestFlexNum JSON 数字、字符串、null 三种写法都应能解析。
func TestFlexNum(t *testing.T) {
	var payload klinePayload
	body := `{"code":0,"data":{"list":[{"k":1,"o":187.5,"c":"178.2","v":"12.5","r":0.03,"lc":187.5,"cp":-9.3}]}}`
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	bars := payload.Data.Bars()
	if len(bars) != 1 {
		t.Fatalf("期望 1 根，实际 %d", len(bars))
	}
	bar := bars[0]
	if bar.Open != "187.5" || bar.Close != "178.2" || bar.Volume != 12.5 ||
		bar.ChangePrice != "-9.3" || bar.LastClose != 187.5 {
		t.Fatalf("混用类型解析不符：%+v", bar)
	}
	if bar.ChangeRatio == nil || *bar.ChangeRatio != 0.03 {
		t.Fatalf("涨跌幅解析不符：%+v", bar.ChangeRatio)
	}

	var empty FlexNum
	if err := json.Unmarshal([]byte("null"), &empty); err != nil {
		t.Fatalf("null 解析失败: %v", err)
	}
	if empty.String() != "" || empty.Float() != 0 {
		t.Fatalf("null 应解析为空串且数值为 0，实际 %q / %v", empty.String(), empty.Float())
	}
	if got := FlexNum("not-a-number").Float(); got != 0 {
		t.Fatalf("非法数值应退回 0，实际 %v", got)
	}
}

// TestTruncate 超长响应体应被截断，避免错误信息膨胀。
func TestTruncate(t *testing.T) {
	if got := truncate("abcdef", 3); got != "abc..." {
		t.Fatalf("截断结果不符：%s", got)
	}
	if got := truncate("abc", 3); got != "abc" {
		t.Fatalf("未超长时不应改变内容：%s", got)
	}
}

// TestConfigHeader 自定义请求头应在内置头之后生效：同名覆盖、空白键跳过；
// 并且 New 时浅拷贝，调用方后续改 map 不影响已建客户端。
func TestConfigHeader(t *testing.T) {
	var got http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		_, _ = w.Write([]byte(`{"code":0,"data":{"pagination":{},"list":[]}}`))
	}))
	defer server.Close()

	header := map[string]string{
		"accept-language": "zh-CN,zh;q=0.9,en;q=0.8",
		"accept":          "application/vnd.moomoo+json",
		"   ":             "空白键应被跳过",
	}
	client := newTestClient(t, server, func(cfg *Config) { cfg.Header = header })
	if _, err := client.GetStockList(context.Background(), GetStockListRequest{MarketType: "1"}); err != nil {
		t.Fatalf("拉取失败: %v", err)
	}

	if v := got.Get("accept-language"); v != "zh-CN,zh;q=0.9,en;q=0.8" {
		t.Fatalf("accept-language 未生效：%q", v)
	}
	if v := got.Get("accept"); v != "application/vnd.moomoo+json" {
		t.Fatalf("accept 应被调用方覆盖：%q", v)
	}
	if v := got.Get("quote-token"); v == "" {
		t.Fatal("内置头不应被自定义头清除")
	}

	// 浅拷贝：调用方后续修改不应影响客户端已生效的配置。
	header["accept-language"] = "en-US"
	if v := client.Config().Header["accept-language"]; v != "zh-CN,zh;q=0.9,en;q=0.8" {
		t.Fatalf("Header 应被浅拷贝，实际 %q", v)
	}
	if len(header) != 3 {
		t.Fatalf("调用方的 map 不应被增删，实际 %d 项", len(header))
	}
}

// TestConfigHeaderNil 不传 Header 时行为与之前一致（内置头原样生效）。
func TestConfigHeaderNil(t *testing.T) {
	var got http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		_, _ = w.Write([]byte(`{"code":0,"data":{"pagination":{},"list":[]}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	if _, err := client.GetStockList(context.Background(), GetStockListRequest{MarketType: "1"}); err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
	if got.Get("accept-language") != "" {
		t.Fatalf("未配置时不应出现 accept-language，实际 %q", got.Get("accept-language"))
	}
	if v := got.Get("accept"); v != "application/json" {
		t.Fatalf("内置 accept 期望 application/json，实际 %q", v)
	}
	if v := got.Get("user-agent"); v != DefaultUserAgent {
		t.Fatalf("内置 user-agent 不符：%q", v)
	}
}

// TestRefererSelection 来源页优先级：
// Referers（市场覆盖） → Referer（全局覆盖） → DefaultReferers（市场默认） → DefaultReferer。
func TestRefererSelection(t *testing.T) {
	var gotReferer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReferer = r.Header.Get("referer")
		_, _ = w.Write([]byte(`{"code":0,"data":{"pagination":{},"list":[]}}`))
	}))
	defer server.Close()

	// list=true 走列表接口（按市场选 referer）；false 走普通接口（只用全局默认）。
	call := func(cfg Config, list bool, marketType string) string {
		t.Helper()
		// 只覆盖 referer / header 相关字段，保留测试服务器地址与固定时钟。
		client := newTestClient(t, server, func(c *Config) {
			c.Referer = cfg.Referer
			c.Referers = cfg.Referers
			c.Header = cfg.Header
		})
		if list {
			if _, err := client.GetStockList(context.Background(), GetStockListRequest{MarketType: marketType}); err != nil {
				t.Fatalf("拉取失败: %v", err)
			}
		} else if _, err := client.get(context.Background(), "/x", Params("marketType", marketType)); err != nil {
			t.Fatalf("拉取失败: %v", err)
		}
		return gotReferer
	}

	cases := []struct {
		name       string
		cfg        Config
		list       bool
		marketType string
		want       string
	}{
		{"零值配置默认取简体中文站首页", Config{}, false, "", DefaultReferer},
		{"列表接口按市场取中文列表页", Config{}, true, "1", DefaultReferers["1"]},
		{"未列出的市场回落默认值", Config{}, true, "9", DefaultReferer},
		{"Referers 覆盖市场默认值", Config{Referers: map[string]string{"1": "https://example.com/m1"}}, true, "1", "https://example.com/m1"},
		{"Referer 优先于市场默认值", Config{Referer: "https://example.com/global"}, true, "1", "https://example.com/global"},
		{"Referers 优先于 Referer", Config{Referer: "https://example.com/global", Referers: map[string]string{"1": "https://example.com/m1"}}, true, "1", "https://example.com/m1"},
		{"Referers 未命中的市场回落 Referer", Config{Referer: "https://example.com/global", Referers: map[string]string{"1": "https://example.com/m1"}}, true, "2", "https://example.com/global"},
		{"Header 的 referer 优先级最高", Config{Header: map[string]string{"referer": "https://example.com/h"}}, true, "1", "https://example.com/h"},
	}
	for _, tc := range cases {
		if got := call(tc.cfg, tc.list, tc.marketType); got != tc.want {
			t.Errorf("%s：referer 期望 %q，实际 %q", tc.name, tc.want, got)
		}
	}
}
