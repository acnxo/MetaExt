package dataapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	return New(Config{BaseURL: server.URL, Timeout: 5 * time.Second, Sleep: func(time.Duration) {}})
}

func TestGetStockListRequestAndParse(t *testing.T) {
	var gotPath, gotReferer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotReferer = r.Header.Get("referer")
		_, _ = w.Write([]byte(`{"stockList": [
		  {"code": "000001", "zwjc": "平安银行", "orgId": "1"},
		  {"code": "000002", "zwjc": "万科A", "orgId": "2"},
		  {"code": "", "zwjc": "无代码", "orgId": "3"}
		]}`))
	}))
	defer server.Close()

	items, err := newTestClient(t, server).GetStockList(context.Background())
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if gotPath != DefaultStockListPath {
		t.Errorf("路径 = %q，期望 %q", gotPath, DefaultStockListPath)
	}
	if gotReferer != DefaultReferer {
		t.Errorf("referer = %q，期望 %q", gotReferer, DefaultReferer)
	}
	if len(items) != 2 {
		t.Fatalf("条目数 = %d，期望 2（空代码应丢弃）", len(items))
	}
	if items[0].Code != "000001" || items[0].Name != "平安银行" {
		t.Fatalf("首条解析不符：%+v", items[0])
	}
}

func TestGetStockListArrayShape(t *testing.T) {
	// 旧版或镜像直接返回数组。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"code":"000001","zwjc":"平安银行"}]`))
	}))
	defer server.Close()

	items, err := newTestClient(t, server).GetStockList(context.Background())
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if len(items) != 1 || items[0].Code != "000001" {
		t.Fatalf("数组形态解析不符：%+v", items)
	}
}

func TestGetStockListEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"stockList": []}`))
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetStockList(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("空列表应返回 *APIError：%v", err)
	}
	if apiErr.Message == "" {
		t.Error("错误信息不应为空")
	}
}

func TestGetStockListHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("unavailable"))
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetStockList(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusServiceUnavailable {
		t.Fatalf("应返回 HTTPStatus=503 的错误：%v", err)
	}
}

func TestGetStockListInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>blocked</html>"))
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetStockList(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Message == "" {
		t.Fatalf("非 JSON 应返回 *APIError：%v", err)
	}
}

func TestConfigDefaults(t *testing.T) {
	got := Config{}.withDefaults()
	if got.BaseURL != DefaultBaseURL || got.StockListPath != DefaultStockListPath ||
		got.Referer != DefaultReferer {
		t.Fatalf("默认值未补齐：%+v", got)
	}
}
