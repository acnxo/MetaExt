package webapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// pageBody 拼一页响应：顶层是数组，第 0 个元素带 metadata 与 data。
func pageBody(pageNo, pageCount int, rows ...string) string {
	return fmt.Sprintf(`[{"metadata":{"pageno":"%d","pagecount":"%d","recordcount":"%d"},"data":[%s]}]`,
		pageNo, pageCount, pageCount*2, strings.Join(rows, ","))
}

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	return New(Config{BaseURL: server.URL, Timeout: 5 * time.Second, Sleep: func(time.Duration) {}})
}

func TestGetStockListSinglePage(t *testing.T) {
	var gotQuery, gotReferer string
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		gotQuery = r.URL.RawQuery
		gotReferer = r.Header.Get("referer")
		_, _ = w.Write([]byte(pageBody(1, 1,
			`{"agdm":"000001","agjc":"<u>平安银行</u>","agssrq":"1991-04-03"}`,
			`{"agdm":"000002","agjc":"<u>万科A</u>","agssrq":"1991-01-29"}`)))
	}))
	defer server.Close()

	page, err := newTestClient(t, server).GetStockList(context.Background(), GetStockListRequest{})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if calls != 1 {
		t.Fatalf("单页只应请求 1 次，实际 %d", calls)
	}
	for _, want := range []string{"CATALOGID=" + DefaultCatalogID, "TABKEY=" + DefaultTabKey, "PAGENO=1"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("查询串缺少 %s：%s", want, gotQuery)
		}
	}
	if gotReferer != DefaultReferer {
		t.Errorf("referer = %q，期望 %q", gotReferer, DefaultReferer)
	}
	if page.PageCount != 1 || page.RecordCount != 2 || len(page.Items) != 2 {
		t.Fatalf("分页信息不符：%+v", page)
	}
	if page.Items[0].Code != "000001" || page.Items[0].Name != "平安银行" ||
		page.Items[0].ListDate != "1991-04-03" {
		t.Fatalf("首条解析不符：%+v", page.Items[0])
	}
}

func TestGetStockListAllPages(t *testing.T) {
	var calls int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&calls, 1)
		pageNo := r.URL.Query().Get("PAGENO")
		switch pageNo {
		case "1":
			_, _ = w.Write([]byte(pageBody(1, 3, `{"agdm":"000001","agjc":"甲","agssrq":"1991-01-01"}`)))
		case "2":
			_, _ = w.Write([]byte(pageBody(2, 3, `{"agdm":"000002","agjc":"乙","agssrq":"1991-01-02"}`)))
		case "3":
			_, _ = w.Write([]byte(pageBody(3, 3, `{"agdm":"000003","agjc":"丙","agssrq":"1991-01-03"}`)))
		default:
			t.Errorf("未预期的页码 %s", pageNo)
		}
	}))
	defer server.Close()

	page, err := newTestClient(t, server).GetStockList(context.Background(), GetStockListRequest{All: true, Workers: 2})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if got := atomic.LoadInt64(&calls); got != 3 {
		t.Fatalf("应请求 3 页，实际 %d", got)
	}
	if len(page.Items) != 3 || page.PageCount != 3 {
		t.Fatalf("合并结果不符：%+v", page)
	}
	// 必须按页序拼接，不能因并发而乱序。
	want := []string{"000001", "000002", "000003"}
	for i, code := range want {
		if page.Items[i].Code != code {
			t.Fatalf("第 %d 条 = %s，期望 %s", i, page.Items[i].Code, code)
		}
	}
}

func TestGetStockListPageError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("PAGENO") == "2" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(pageBody(1, 2, `{"agdm":"000001","agjc":"甲","agssrq":"1991-01-01"}`)))
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetStockList(context.Background(), GetStockListRequest{All: true})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("应返回 *APIError：%v", err)
	}
	if apiErr.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("HTTPStatus = %d，期望 500", apiErr.HTTPStatus)
	}
}

func TestCleanName(t *testing.T) {
	cases := map[string]string{
		"<u>平安银行</u>":                            "平安银行",
		"<a href='#' target='_blank'>XD平安银行</a>": "XD平安银行",
		"  万科A  ":                                "万科A",
		"<u>ST泰禾</u>":                            "ST泰禾",
	}
	for raw, want := range cases {
		if got := cleanName(raw); got != want {
			t.Errorf("cleanName(%q) = %q，期望 %q", raw, got, want)
		}
	}
}

func TestConfigDefaults(t *testing.T) {
	got := Config{}.withDefaults()
	if got.BaseURL != DefaultBaseURL || got.StockListPath != DefaultStockListPath ||
		got.Referer != DefaultReferer {
		t.Fatalf("默认值未补齐：%+v", got)
	}
}
