package query

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestClient 构造指向测试服务器的客户端，并跳过退避等待。
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	return New(Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
		Sleep:   func(time.Duration) {},
	})
}

func TestGetStockListRequestAndParse(t *testing.T) {
	var gotPath, gotQuery, gotReferer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotReferer = r.Header.Get("referer")
		_, _ = w.Write([]byte(`{
		  "pageHelp": {"pageNo": 1, "pageSize": 3000, "total": 3},
		  "result": [
		    {"A_STOCK_CODE": "600000", "SEC_NAME_CN": "浦发银行", "LIST_DATE": "19991110"},
		    {"A_STOCK_CODE": "601398", "SEC_NAME_CN": "工商银行", "LIST_DATE": "20061027"},
		    {"A_STOCK_CODE": "688981", "SEC_NAME_CN": "中芯国际", "LIST_DATE": "20200716"}
		  ]
		}`))
	}))
	defer server.Close()

	items, err := newTestClient(t, server).GetStockList(context.Background(), GetStockListRequest{})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if gotPath != DefaultStockListPath {
		t.Errorf("路径 = %q，期望 %q", gotPath, DefaultStockListPath)
	}
	for _, want := range []string{"sqlId=" + DefaultSQLID, "STOCK_TYPE=1", "pageHelp.pageSize=3000"} {
		if !containsParam(gotQuery, want) {
			t.Errorf("查询串缺少 %s：%s", want, gotQuery)
		}
	}
	if gotReferer != DefaultReferer {
		t.Errorf("referer = %q，期望 %q", gotReferer, DefaultReferer)
	}
	if len(items) != 3 {
		t.Fatalf("条目数 = %d，期望 3", len(items))
	}
	if items[0].Code != "600000" || items[0].Name != "浦发银行" || items[0].ListDate != "19991110" {
		t.Fatalf("首条解析不符：%+v", items[0])
	}
}

func TestGetStockListSkipsInvalidRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// 上市日期不是八位数字、代码为空、名称为空的行都应被丢弃。
		_, _ = w.Write([]byte(`{"result": [
		  {"A_STOCK_CODE": "600000", "SEC_NAME_CN": "浦发银行", "LIST_DATE": "19991110"},
		  {"A_STOCK_CODE": "600001", "SEC_NAME_CN": "无日期", "LIST_DATE": ""},
		  {"A_STOCK_CODE": "600002", "SEC_NAME_CN": "短日期", "LIST_DATE": "1999-1"},
		  {"A_STOCK_CODE": "", "SEC_NAME_CN": "无代码", "LIST_DATE": "19991110"},
		  {"A_STOCK_CODE": "600003", "SEC_NAME_CN": "", "LIST_DATE": "19991110"}
		]}`))
	}))
	defer server.Close()

	items, err := newTestClient(t, server).GetStockList(context.Background(), GetStockListRequest{PageSize: 100})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if len(items) != 1 || items[0].Code != "600000" {
		t.Fatalf("应只保留 1 条合法记录：%+v", items)
	}
}

func TestGetStockListHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetStockList(context.Background(), GetStockListRequest{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("应返回 *APIError：%v", err)
	}
	if apiErr.HTTPStatus != http.StatusBadGateway {
		t.Errorf("HTTPStatus = %d，期望 502", apiErr.HTTPStatus)
	}
}

func TestGetStockListInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>blocked</html>"))
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetStockList(context.Background(), GetStockListRequest{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.URL == "" {
		t.Fatalf("应返回带 URL 的 *APIError：%v", err)
	}
	if apiErr.Message == "" {
		t.Error("错误信息不应为空")
	}
}

func TestGetStockListContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"result": []}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := newTestClient(t, server).GetStockList(ctx, GetStockListRequest{}); err == nil {
		t.Fatal("ctx 取消应返回错误")
	}
}

func TestConfigDefaults(t *testing.T) {
	got := Config{}.withDefaults()
	if got.BaseURL != DefaultBaseURL || got.StockListPath != DefaultStockListPath ||
		got.Referer != DefaultReferer {
		t.Fatalf("默认值未补齐：%+v", got)
	}
	// 结尾斜杠与前导斜杠都要规整。
	got = Config{BaseURL: "http://x/", StockListPath: "p"}.withDefaults()
	if got.BaseURL != "http://x" || got.StockListPath != "/p" {
		t.Fatalf("URL 规整不符：%+v", got)
	}
}

// containsParam 判断查询串里是否包含指定键值对片段。
func containsParam(rawQuery, want string) bool {
	for _, pair := range splitParams(rawQuery) {
		if pair == want {
			return true
		}
	}
	return false
}

func splitParams(rawQuery string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(rawQuery); i++ {
		if i == len(rawQuery) || rawQuery[i] == '&' {
			if i > start {
				out = append(out, rawQuery[start:i])
			}
			start = i + 1
		}
	}
	return out
}
