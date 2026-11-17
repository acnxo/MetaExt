package stock

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newTestClient 构造指向测试服务器的客户端，预热地址也指向同一个服务器。
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	return New(Config{
		BaseURL: server.URL,
		WarmURL: server.URL + "/warm",
		Timeout: 5 * time.Second,
		Sleep:   func(time.Duration) {},
	})
}

const klineBody = `{"error_code":0,"data":{
  "symbol":"SZ000001","period":"day",
  "column":["timestamp","volume","open","high","low","close","amount"],
  "item":[
    [1794412800000, 198765432, 11.70, 12.00, 11.65, 11.99, 2380000000],
    [1794499200000, 218942998, 11.76, 12.10, 11.70, 12.07, 2593456789]
  ]}}`

func TestGetKLineWarmsOnceAndParses(t *testing.T) {
	var warmCalls, dataCalls int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/warm" {
			atomic.AddInt64(&warmCalls, 1)
			http.SetCookie(w, &http.Cookie{Name: "xq_a_token", Value: "t", Path: "/"})
			_, _ = w.Write([]byte("ok"))
			return
		}
		atomic.AddInt64(&dataCalls, 1)
		// 预热拿到的 cookie 必须带到数据请求上。
		if cookie, err := r.Cookie("xq_a_token"); err != nil || cookie.Value != "t" {
			t.Errorf("数据请求未携带预热 cookie：%v", err)
		}
		_, _ = w.Write([]byte(klineBody))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	for i := 0; i < 3; i++ {
		result, err := client.GetKLine(context.Background(), GetKLineRequest{Code: "000001"})
		if err != nil {
			t.Fatalf("第 %d 次调用返回错误：%v", i+1, err)
		}
		if result.Symbol != "SZ000001" || len(result.Bars) != 2 {
			t.Fatalf("结果不符：%+v", result)
		}
	}
	// 预热只做一次，数据请求每次一次。
	if got := atomic.LoadInt64(&warmCalls); got != 1 {
		t.Errorf("预热调用 = %d，期望 1", got)
	}
	if got := atomic.LoadInt64(&dataCalls); got != 3 {
		t.Errorf("数据请求 = %d，期望 3", got)
	}

	result, _ := client.GetKLine(context.Background(), GetKLineRequest{Code: "000001"})
	bar := result.Bars[1]
	if bar.Date != "2026-11-13" || bar.Open != 11.76 || bar.Close != 12.07 ||
		bar.High != 12.10 || bar.Low != 11.70 {
		t.Fatalf("OHLC 解析不符：%+v", bar)
	}
	// 雪球的量与额都是原生单位，不换算。
	if bar.Volume != 218942998 || bar.Amount != 2593456789 {
		t.Errorf("量额解析不符：%+v", bar)
	}
}

func TestGetKLineRebuildsSessionOnEmpty(t *testing.T) {
	var warmCalls, dataCalls int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/warm" {
			atomic.AddInt64(&warmCalls, 1)
			_, _ = w.Write([]byte("ok"))
			return
		}
		call := atomic.AddInt64(&dataCalls, 1)
		if call == 1 {
			// 首次模拟 cookie 失效：返回空 item。
			_, _ = w.Write([]byte(`{"error_code":0,"data":{"symbol":"SZ000001","column":["timestamp","close"],"item":[]}}`))
			return
		}
		_, _ = w.Write([]byte(klineBody))
	}))
	defer server.Close()

	result, err := newTestClient(t, server).GetKLine(context.Background(), GetKLineRequest{Code: "000001"})
	if err != nil {
		t.Fatalf("重建会话后应成功：%v", err)
	}
	if len(result.Bars) != 2 {
		t.Fatalf("条目数 = %d，期望 2", len(result.Bars))
	}
	if got := atomic.LoadInt64(&warmCalls); got != 2 {
		t.Errorf("预热应执行 2 次（失效后重建），实际 %d", got)
	}
}

func TestGetKLineEmptyPersistsAsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error_code":0,"data":{"symbol":"SZ000001","column":[],"item":[]}}`))
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetKLine(context.Background(), GetKLineRequest{Code: "000001"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("持续空数据应返回 *APIError：%v", err)
	}
	if !strings.Contains(apiErr.Message, "会话可能失效") {
		t.Errorf("错误信息应提示会话失效：%v", apiErr)
	}
}

func TestGetKLineBusinessError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error_code":400016,"error_description":"参数错误"}`))
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetKLine(context.Background(), GetKLineRequest{Code: "000001"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != 400016 {
		t.Fatalf("应返回带业务码的 *APIError：%v", err)
	}
}

func TestGetKLineColumnOrderIndependent(t *testing.T) {
	// 列顺序打乱后仍应按列名取值。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/warm" {
			_, _ = w.Write([]byte("ok"))
			return
		}
		_, _ = w.Write([]byte(`{"error_code":0,"data":{
		  "symbol":"SZ000001",
		  "column":["close","timestamp","volume","open","high","low","amount"],
		  "item":[[12.07,1794499200000,218942998,11.76,12.10,11.70,2593456789]]}}`))
	}))
	defer server.Close()

	result, err := newTestClient(t, server).GetKLine(context.Background(), GetKLineRequest{Code: "000001"})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	bar := result.Bars[0]
	if bar.Close != 12.07 || bar.Open != 11.76 || bar.High != 12.10 ||
		bar.Low != 11.70 || bar.Volume != 218942998 || bar.Date != "2026-11-13" {
		t.Fatalf("列名取值不符：%+v", bar)
	}
}

func TestResolveSymbol(t *testing.T) {
	cases := map[string]string{
		"600000": "SH600000", "000001": "SZ000001",
		// 北交所（4/8 开头）必须用 BJ 前缀。
		"830799": "BJ830799", "430047": "BJ430047",
	}
	for code, want := range cases {
		got, err := resolveSymbol(GetKLineRequest{Code: code})
		if err != nil || got != want {
			t.Errorf("resolveSymbol(%s) = %q, %v；期望 %q", code, got, err, want)
		}
	}
	if _, err := resolveSymbol(GetKLineRequest{}); err == nil {
		t.Fatal("缺少 Symbol 与 Code 时应报错")
	}
}

func TestConfigDefaults(t *testing.T) {
	got := Config{}.withDefaults()
	if got.BaseURL != DefaultBaseURL || got.WarmURL != DefaultWarmURL || got.Referer != DefaultReferer {
		t.Fatalf("默认值未补齐：%+v", got)
	}
}
