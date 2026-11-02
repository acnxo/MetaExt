package cnatradecalendar

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient 创建指向测试服务器的客户端。
func newTestClient(t *testing.T, server *httptest.Server, cfg Config) *Client {
	t.Helper()
	cfg.BaseURL = server.URL
	return New(cfg)
}

// TestGetTradeCalendar 校验请求路径、请求头与解析结果。
func TestGetTradeCalendar(t *testing.T) {
	var gotPath, gotUA, gotReferer, gotAccept string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotUA = r.Header.Get("user-agent")
		gotReferer = r.Header.Get("referer")
		gotAccept = r.Header.Get("accept")
		_, _ = w.Write([]byte(calendarBody))
	}))
	defer server.Close()

	client := newTestClient(t, server, Config{})
	cal, err := client.GetTradeCalendar(context.Background())
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
	if gotPath != DefaultCalendarPath {
		t.Errorf("路径期望 %s，实际 %s", DefaultCalendarPath, gotPath)
	}
	if gotUA != DefaultUserAgent || gotReferer != DefaultReferer {
		t.Errorf("默认请求头不符：UA=%q Referer=%q", gotUA, gotReferer)
	}
	if gotAccept != "text/plain,*/*" {
		t.Errorf("accept 期望 text/plain,*/*，实际 %q", gotAccept)
	}
	if cal.Len() != 8796 {
		t.Fatalf("交易日数量期望 8796，实际 %d", cal.Len())
	}
}

// TestGetTradeCalendarConfigDefaults 零值配置补默认值，自定义路径自动补前导斜杠。
func TestGetTradeCalendarConfigDefaults(t *testing.T) {
	cfg := Config{CalendarPath: "realstock/company/klc_td_sh.txt"}.withDefaults()
	if cfg.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL 期望 %s，实际 %s", DefaultBaseURL, cfg.BaseURL)
	}
	if cfg.CalendarPath != DefaultCalendarPath {
		t.Errorf("CalendarPath 期望 %s，实际 %s", DefaultCalendarPath, cfg.CalendarPath)
	}
	if cfg.HTTPClient == nil || cfg.HTTPClient.Timeout != defaultTimeout {
		t.Errorf("HTTPClient 期望带 %v 超时", defaultTimeout)
	}
	if cfg.UserAgent != DefaultUserAgent || cfg.Referer != DefaultReferer {
		t.Errorf("默认请求头未补齐：%+v", cfg)
	}

	custom := Config{BaseURL: "https://example.com/", CalendarPath: "/x.txt"}.withDefaults()
	if custom.BaseURL != "https://example.com" {
		t.Errorf("BaseURL 应去掉末尾斜杠，实际 %s", custom.BaseURL)
	}
	if custom.CalendarPath != "/x.txt" {
		t.Errorf("CalendarPath 不应被改写，实际 %s", custom.CalendarPath)
	}
}

// TestGetTradeCalendarHTTPError 非 200 返回带状态码的错误。
func TestGetTradeCalendarHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden"))
	}))
	defer server.Close()

	_, err := newTestClient(t, server, Config{}).GetTradeCalendar(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusForbidden {
		t.Fatalf("期望 HTTPStatus=403 的错误，实际 %v", err)
	}
	if !strings.Contains(apiErr.Error(), "403") {
		t.Errorf("错误信息应包含状态码，实际 %q", apiErr.Error())
	}
}

// TestGetTradeCalendarNetworkError 网络错误包装为 APIError 且可解包。
func TestGetTradeCalendarNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	client := New(Config{BaseURL: url})
	_, err := client.GetTradeCalendar(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("期望 APIError，实际 %v", err)
	}
	if apiErr.HTTPStatus != 0 || apiErr.Unwrap() == nil {
		t.Fatalf("网络错误应带底层错误且状态码为 0，实际 %+v", apiErr)
	}
}

// TestGetTradeCalendarCanceled 上下文取消应中止请求。
func TestGetTradeCalendarCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := newTestClient(t, server, Config{}).GetTradeCalendar(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("期望 context.Canceled，实际 %v", err)
	}
}

// TestGetTradeCalendarNoData 响应格式变化时不能当成空日历返回。
func TestGetTradeCalendarNoData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<!DOCTYPE html><html>maintenance</html>"))
	}))
	defer server.Close()

	_, err := newTestClient(t, server, Config{}).GetTradeCalendar(context.Background())
	if !errors.Is(err, ErrNoData) {
		t.Fatalf("期望 ErrNoData，实际 %v", err)
	}
}

// TestGetTradeCalendarClientConfig 可注入自定义客户端与超时。
func TestGetTradeCalendarClientConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(calendarBody))
	}))
	defer server.Close()

	custom := &http.Client{Timeout: 3 * time.Second}
	client := newTestClient(t, server, Config{HTTPClient: custom})
	if client.Config().HTTPClient != custom {
		t.Fatal("自定义 HTTPClient 未生效")
	}
	if _, err := client.GetTradeCalendar(context.Background()); err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
}

// TestTruncate 过长文本按上限截断并追加省略号。
func TestTruncate(t *testing.T) {
	if got := truncate("  short  ", 20); got != "short" {
		t.Errorf("应去掉首尾空白，实际 %q", got)
	}
	long := strings.Repeat("a", 30)
	if got := truncate(long, 10); got != strings.Repeat("a", 10)+"..." {
		t.Errorf("截断结果不符：%q", got)
	}
}

// errReader 在读取响应体时报错，用于验证读取失败的路径。
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("模拟读取失败") }

// errTransport 返回状态 200 但响应体读取会失败。
type errTransport struct{}

func (errTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(errReader{}),
		Header:     make(http.Header),
	}, nil
}

// TestGetTradeCalendarReadError 读取响应体失败时包装为 APIError。
func TestGetTradeCalendarReadError(t *testing.T) {
	client := New(Config{HTTPClient: &http.Client{Transport: errTransport{}}})
	_, err := client.GetTradeCalendar(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusOK {
		t.Fatalf("期望带 200 的读取失败错误，实际 %v", err)
	}
	if apiErr.Unwrap() == nil {
		t.Fatal("应包装底层错误")
	}
}

// TestGetTradeCalendarBadURL 站点地址非法时构造请求失败。
func TestGetTradeCalendarBadURL(t *testing.T) {
	client := New(Config{BaseURL: "://bad"})
	_, err := client.GetTradeCalendar(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !strings.Contains(apiErr.Error(), "构造请求失败") {
		t.Fatalf("期望构造请求失败，实际 %v", err)
	}
	if apiErr.HTTPStatus != 0 {
		t.Fatalf("构造请求失败不应带状态码，实际 %d", apiErr.HTTPStatus)
	}
}
