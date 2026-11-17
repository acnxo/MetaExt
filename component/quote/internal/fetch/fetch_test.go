package fetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newFetcher 构造指向测试服务器的请求层，并跳过退避等待。
func newFetcher(t *testing.T) *Fetcher {
	t.Helper()
	return New(Config{
		Timeout: 5 * time.Second,
		Sleep:   func(time.Duration) {},
	})
}

func TestGetReturnsBodyAndHeaders(t *testing.T) {
	var gotUA, gotReferer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("user-agent")
		gotReferer = r.Header.Get("referer")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	body, err := newFetcher(t).Get(context.Background(), server.URL, map[string]string{
		"referer": "https://quote.eastmoney.com/",
	})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("响应体不符：%q", body)
	}
	if gotUA != DefaultUserAgent {
		t.Errorf("UA 未使用默认值：%q", gotUA)
	}
	if gotReferer != "https://quote.eastmoney.com/" {
		t.Errorf("Referer 未透传：%q", gotReferer)
	}
}

func TestGetRetriesOnServerError(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("bad gateway"))
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	body, err := newFetcher(t).Get(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("应重试后成功：%v", err)
	}
	if string(body) != "ok" || calls != 3 {
		t.Fatalf("重试次数不符：calls=%d body=%q", calls, body)
	}
}

func TestGetDoesNotRetryOnClientError(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	defer server.Close()

	_, err := newFetcher(t).Get(context.Background(), server.URL, nil)
	if err == nil {
		t.Fatal("404 应返回错误")
	}
	if calls != 1 {
		t.Fatalf("4xx 不应重试，实际请求 %d 次", calls)
	}
	var statusErr *StatusError
	if !errors.As(err, &statusErr) || statusErr.Status != http.StatusNotFound {
		t.Fatalf("应返回 *StatusError(404)：%v", err)
	}
	if !strings.Contains(statusErr.Body, "not found") {
		t.Errorf("错误体未保留：%q", statusErr.Body)
	}
}

func TestGetRetriesOnTooManyRequests(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	if _, err := newFetcher(t).Get(context.Background(), server.URL, nil); err != nil {
		t.Fatalf("429 应重试后成功：%v", err)
	}
	if calls != 2 {
		t.Fatalf("请求次数不符：%d", calls)
	}
}

func TestGetExhaustsRetries(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := newFetcher(t).Get(context.Background(), server.URL, nil)
	if err == nil {
		t.Fatal("持续 5xx 应返回错误")
	}
	if calls != DefaultRetries {
		t.Fatalf("请求次数 = %d，期望 %d", calls, DefaultRetries)
	}
	if !strings.Contains(err.Error(), "已尝试") {
		t.Errorf("错误信息缺少尝试次数：%v", err)
	}
}

func TestGetContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := newFetcher(t).Get(ctx, server.URL, nil); err == nil {
		t.Fatal("ctx 取消应返回错误")
	}
}

func TestGetTextDecodesGBK(t *testing.T) {
	// "平安银行" 的 GBK 编码。
	gbk := []byte{0xC6, 0xBD, 0xB0, 0xB2, 0xD2, 0xF8, 0xD0, 0xD0}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "text/plain; charset=GBK")
		_, _ = w.Write(gbk)
	}))
	defer server.Close()

	text, err := newFetcher(t).GetText(context.Background(), server.URL, nil, GBK)
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if text != "平安银行" {
		t.Fatalf("GBK 解码不符：%q", text)
	}

	// UTF-8 路径保持原样。
	text, err = newFetcher(t).GetText(context.Background(), server.URL, nil, UTF8)
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if text == "平安银行" {
		t.Fatal("按 UTF-8 解码 GBK 字节不应得到正确中文")
	}
}

func TestGetJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"data":{"total":2}}`))
	}))
	defer server.Close()

	var out struct {
		Code int `json:"code"`
		Data struct {
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := newFetcher(t).GetJSON(context.Background(), server.URL, nil, &out); err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if out.Code != 0 || out.Data.Total != 2 {
		t.Fatalf("解析不符：%+v", out)
	}

	// 非 JSON 应报解析错误。
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>blocked</html>"))
	}))
	defer bad.Close()
	if err := newFetcher(t).GetJSON(context.Background(), bad.URL, nil, &out); err == nil ||
		!strings.Contains(err.Error(), "响应解析失败") {
		t.Fatalf("非 JSON 应报解析失败：%v", err)
	}
}

func TestConfigDefaults(t *testing.T) {
	got := Config{}.withDefaults()
	if got.UserAgent != DefaultUserAgent || got.Timeout != DefaultTimeout ||
		got.Retries != DefaultRetries || got.MaxBytes != DefaultMaxBytes || got.Sleep == nil {
		t.Fatalf("默认值未补齐：%+v", got)
	}
}

func TestMaxBytesLimitsBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("a", 100)))
	}))
	defer server.Close()

	f := New(Config{MaxBytes: 10, Sleep: func(time.Duration) {}})
	body, err := f.Get(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if len(body) != 10 {
		t.Fatalf("响应体应被截断到 10 字节，实际 %d", len(body))
	}
}
