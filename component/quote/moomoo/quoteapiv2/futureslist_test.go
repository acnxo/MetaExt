package quoteapiv2

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// futuresListSample 期货列表响应样本（节选）：位置化字段结构，fieldList 为字段定义与原始值，
// fields 为按字段编号的展示值，其中 "19" 只出现在 fields 中。
const futuresListSample = `{
    "code": 0,
    "message": "成功",
    "data": {
        "pagination": {"page": 0, "pageCount": 1, "pageSize": 40, "total": 14},
        "list": [
            {
                "stockId": 70000294,
                "priceDirect": "down",
                "fieldList": [
                    {"fieldId": 1, "fieldName": "合约代码", "text": "GCmain", "contentType": 4},
                    {"fieldId": 3, "fieldName": "价格", "value": "4413800000000", "precision": 1, "contentType": 2},
                    {"fieldId": 4, "fieldName": "涨跌幅", "value": "-2508531", "precision": 2, "contentType": 1}
                ],
                "fields": {
                    "1": "GCmain", "2": "黄金", "3": "4413.8", "4": "-0.25%",
                    "11": "315327.0", "18": "4424.9", "19": "US"
                }
            }
        ]
    }
}`

// TestGetFuturesListRequestAndParse 校验路径、参数键序、签名与位置化字段解析。
func TestGetFuturesListRequestAndParse(t *testing.T) {
	var (
		gotOrder   []string
		gotQuery   map[string]string
		gotHeaders http.Header
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != DefaultFuturesListPath {
			t.Errorf("路径期望 %s，实际 %s", DefaultFuturesListPath, r.URL.Path)
		}
		gotHeaders = r.Header.Clone()
		gotOrder, gotQuery = splitQuery(r.URL.RawQuery)
		_, _ = w.Write([]byte(futuresListSample))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	page, err := client.GetFuturesList(context.Background(), GetFuturesListRequest{
		MarketType:      "2",
		FuturesType:     "4",
		Page:            "0",
		PageSize:        "40",
		TimestampMillis: "1789944039632",
	})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}

	wantOrder := []string{"marketType", "futuresType", "page", "pageSize", "_"}
	if strings.Join(gotOrder, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("查询键序不符：期望 %v，实际 %v", wantOrder, gotOrder)
	}
	if gotQuery["marketType"] != "2" || gotQuery["futuresType"] != "4" {
		t.Fatalf("必填参数不符：%v", gotQuery)
	}
	if gotQuery["page"] != "0" || gotQuery["pageSize"] != "40" {
		t.Fatalf("分页参数不符：%v", gotQuery)
	}
	if got := gotHeaders.Get("quote-token"); got != "d928a3b3e1" {
		t.Fatalf("quote-token 期望 d928a3b3e1，实际 %s", got)
	}

	if page.Pagination.Page != 0 || page.Pagination.PageSize != 40 ||
		page.Pagination.PageCount != 1 || page.Pagination.Total != 14 {
		t.Fatalf("分页解析不符：%+v", page.Pagination)
	}
	if len(page.Items) != 1 {
		t.Fatalf("条目数期望 1，实际 %d", len(page.Items))
	}
	item := page.Items[0]
	if item.StockID != 70000294 || item.PriceDirect != "down" {
		t.Fatalf("基本字段不符：%+v", item)
	}
	if len(item.Fields) != 7 {
		t.Fatalf("fields 键数期望 7，实际 %d", len(item.Fields))
	}
	if item.Fields["1"] != "GCmain" || item.Fields["3"] != "4413.8" ||
		item.Fields["4"] != "-0.25%" || item.Fields["19"] != "US" {
		t.Fatalf("fields 展示值不符：%v", item.Fields)
	}
	if len(item.FieldList) != 3 {
		t.Fatalf("fieldList 项数期望 3，实际 %d", len(item.FieldList))
	}
	price := item.FieldList[1]
	if price.FieldID != 3 || price.FieldName != "价格" || price.Value != "4413800000000" ||
		price.Precision != 1 || price.ContentType != 2 {
		t.Fatalf("数值类字段定义不符：%+v", price)
	}
	code := item.FieldList[0]
	if code.FieldID != 1 || code.Text != "GCmain" || code.ContentType != 4 || code.Value != "" {
		t.Fatalf("文本类字段定义不符：%+v", code)
	}
}

// TestGetFuturesListDefaults 未传分页与时间戳时应补默认值并省略时间戳。
func TestGetFuturesListDefaults(t *testing.T) {
	var gotOrder []string
	var gotQuery map[string]string
	var gotToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("quote-token")
		gotOrder, gotQuery = splitQuery(r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"code":0,"data":{"pagination":{},"list":[]}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	page, err := client.GetFuturesList(context.Background(), GetFuturesListRequest{
		MarketType:  "2",
		FuturesType: "4",
	})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("空列表应返回 0 条，实际 %d", len(page.Items))
	}
	want := []string{"marketType", "futuresType", "page", "pageSize"}
	if strings.Join(gotOrder, ",") != strings.Join(want, ",") {
		t.Fatalf("查询键序不符：期望 %v，实际 %v", want, gotOrder)
	}
	if gotQuery["page"] != DefaultFuturesPage || gotQuery["pageSize"] != DefaultFuturesPageSize {
		t.Fatalf("默认值不符：%v", gotQuery)
	}
	if _, ok := gotQuery["_"]; ok {
		t.Fatal("时间戳为空时应省略")
	}
	if gotToken != "f9c3474442" {
		t.Fatalf("默认参数的 quote-token 期望 f9c3474442，实际 %s", gotToken)
	}
}

// TestGetFuturesListMissingRequired marketType 与 futuresType 都是必填，缺失时本地报错。
func TestGetFuturesListMissingRequired(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	cases := []GetFuturesListRequest{
		{FuturesType: "4"},
		{MarketType: "2"},
		{MarketType: "  ", FuturesType: "4"},
	}
	for _, req := range cases {
		if _, err := client.GetFuturesList(context.Background(), req); err == nil {
			t.Fatalf("缺少必填参数应报错：%+v", req)
		}
	}
	if called {
		t.Fatal("缺少必填参数时不应发起请求")
	}
}

// TestGetFuturesListInvalidPaging 页码不能为负，每页数量不能小于 1。
func TestGetFuturesListInvalidPaging(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("非法分页参数不应发起请求")
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	cases := []GetFuturesListRequest{
		{MarketType: "2", FuturesType: "4", Page: "-1"},
		{MarketType: "2", FuturesType: "4", Page: "x"},
		{MarketType: "2", FuturesType: "4", PageSize: "0"},
	}
	for _, req := range cases {
		if _, err := client.GetFuturesList(context.Background(), req); err == nil {
			t.Fatalf("应拒绝非法分页参数：%+v", req)
		}
	}
}

// TestGetFuturesListBusinessError 参数组合不匹配时服务端返回 500。
func TestGetFuturesListBusinessError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":500,"message":"Params Error","data":null}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetFuturesList(context.Background(), GetFuturesListRequest{
		MarketType:  "9",
		FuturesType: "9",
	})
	if err == nil || !IsParamsError(err) {
		t.Fatalf("应识别为参数错误（500），实际 %v", err)
	}
}

// TestGetFuturesListHTTPError 非 200 应返回带状态码的错误。
func TestGetFuturesListHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("unavailable"))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetFuturesList(context.Background(), GetFuturesListRequest{
		MarketType:  "2",
		FuturesType: "4",
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusServiceUnavailable {
		t.Fatalf("应返回 HTTPStatus=503 的错误，实际 %v", err)
	}
}
