package quoteapiv2

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fxRateListSample 汇率列表响应样本，取自实测报文：价格为字符串且小数位随货币对变化。
const fxRateListSample = `{
    "code": 0,
    "message": "成功",
    "data": {
        "listId": "1",
        "list": [
            {
                "stockId": 72000027,
                "stockCode": "EURUSD",
                "marketLabel": "FX",
                "instrumentType": 10,
                "name": "EUR/USD",
                "priceDirect": "flat",
                "priceNominal": "1.12286",
                "changeAmount": "-0.00001",
                "changePercent": "0.00%",
                "priceBid": "1.12269",
                "priceBidDirect": "down",
                "priceAsk": "1.12303",
                "priceAskDirect": "up",
                "priceOpen": "1.12287",
                "priceLastClose": "1.12287",
                "priceHighest": "1.12291",
                "priceLowest": "1.12266",
                "amplitudePrice": "0.02%"
            }
        ],
        "pagination": {"page": 0, "pageSize": 50, "pageCount": 6, "total": 282}
    }
}`

// TestGetFxRateListRequestAndParse 校验路径、参数键序、默认值、签名、响应解析。
func TestGetFxRateListRequestAndParse(t *testing.T) {
	var (
		gotPath    string
		gotOrder   []string
		gotQuery   map[string]string
		gotHeaders http.Header
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotHeaders = r.Header.Clone()
		gotOrder, gotQuery = splitQuery(r.URL.RawQuery)
		_, _ = w.Write([]byte(fxRateListSample))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	page, err := client.GetFxRateList(context.Background(), GetFxRateListRequest{
		ListID:          "1",
		Page:            "0",
		PageSize:        "50",
		TimestampMillis: "1793831403000",
	})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}

	if gotPath != DefaultFxRateListPath {
		t.Fatalf("路径期望 %s，实际 %s", DefaultFxRateListPath, gotPath)
	}
	wantOrder := []string{"listId", "page", "pageSize", "_"}
	if strings.Join(gotOrder, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("查询键序不符：期望 %v，实际 %v", wantOrder, gotOrder)
	}
	if gotQuery["listId"] != "1" || gotQuery["page"] != "0" || gotQuery["pageSize"] != "50" {
		t.Fatalf("参数不符：%v", gotQuery)
	}
	if gotQuery["_"] != "1793831403000" {
		t.Fatalf("时间戳未透传：%s", gotQuery["_"])
	}
	// 实测签名向量，可作回归基线。
	if got := gotHeaders.Get("quote-token"); got != "71a7397b8b" {
		t.Fatalf("quote-token 期望 71a7397b8b，实际 %s", got)
	}

	if page.ListID != "1" {
		t.Fatalf("listId 应回显，实际 %q", page.ListID)
	}
	if page.Pagination.Page != 0 || page.Pagination.PageSize != 50 ||
		page.Pagination.PageCount != 6 || page.Pagination.Total != 282 {
		t.Fatalf("分页解析不符：%+v", page.Pagination)
	}
	if len(page.Items) != 1 {
		t.Fatalf("条目数期望 1，实际 %d", len(page.Items))
	}
	item := page.Items[0]
	if item.StockID != 72000027 || item.StockCode != "EURUSD" || item.MarketLabel != "FX" ||
		item.InstrumentType != 10 || item.Name != "EUR/USD" {
		t.Fatalf("基本字段不符：%+v", item)
	}
	if item.PriceDirect != "flat" || item.PriceNominal != "1.12286" ||
		item.ChangeAmount != "-0.00001" || item.ChangePercent != "0.00%" {
		t.Fatalf("涨跌字段不符：%+v", item)
	}
	if item.PriceBid != "1.12269" || item.PriceBidDirect != "down" ||
		item.PriceAsk != "1.12303" || item.PriceAskDirect != "up" {
		t.Fatalf("买卖盘字段不符：%+v", item)
	}
	if item.PriceOpen != "1.12287" || item.PriceLastClose != "1.12287" ||
		item.PriceHighest != "1.12291" || item.PriceLowest != "1.12266" ||
		item.AmplitudePrice != "0.02%" {
		t.Fatalf("开高低与振幅字段不符：%+v", item)
	}
}

// TestGetFxRateListDefaults 缺省时补 listId=1、page=0、pageSize=50，并省略时间戳。
func TestGetFxRateListDefaults(t *testing.T) {
	var gotOrder []string
	var gotQuery map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrder, gotQuery = splitQuery(r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"code":0,"data":{"pagination":{},"list":[]}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	if _, err := client.GetFxRateList(context.Background(), GetFxRateListRequest{}); err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
	want := "listId,page,pageSize"
	if strings.Join(gotOrder, ",") != want {
		t.Fatalf("默认参数键序期望 %s，实际 %v", want, gotOrder)
	}
	if gotQuery["listId"] != DefaultFxListID || gotQuery["page"] != DefaultFxRatePage ||
		gotQuery["pageSize"] != DefaultFxRatePageSize {
		t.Fatalf("默认值不符：%v", gotQuery)
	}
	if _, ok := gotQuery["_"]; ok {
		t.Fatal("时间戳缺省时应省略该键")
	}
}

// TestGetFxRateListSignatureVector 第二个实测签名向量，覆盖非默认参数组合。
func TestGetFxRateListSignatureVector(t *testing.T) {
	var gotToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("quote-token")
		_, _ = w.Write([]byte(`{"code":0,"data":{"pagination":{},"list":[]}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	if _, err := client.GetFxRateList(context.Background(), GetFxRateListRequest{
		ListID:          "11",
		Page:            "2",
		PageSize:        "20",
		TimestampMillis: "1793859759000",
	}); err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
	if gotToken != "116547b41a" {
		t.Fatalf("quote-token 期望 116547b41a，实际 %s", gotToken)
	}
}

// TestGetFxRateListOutOfRange 页码越界时服务端返回 code=0、空列表且不回显 listId。
func TestGetFxRateListOutOfRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"data":{"list":[],"pagination":{"page":99,"pageSize":50,"pageCount":6,"total":282}}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	page, err := client.GetFxRateList(context.Background(), GetFxRateListRequest{Page: "99"})
	if err != nil {
		t.Fatalf("越界页不应报错: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("越界页应为空列表，实际 %d 条", len(page.Items))
	}
	if page.ListID != "" {
		t.Fatalf("越界页 listId 应缺失（空串），实际 %q", page.ListID)
	}
	if page.Pagination.Page != 99 || page.Pagination.Total != 282 {
		t.Fatalf("越界页分页应回显请求页码：%+v", page.Pagination)
	}
}

// TestGetFxRateListInvalidPaging 页码须不小于 0，每页数量须不小于 1。
func TestGetFxRateListInvalidPaging(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("非法分页参数不应发起请求")
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	cases := []GetFxRateListRequest{
		{Page: "-1"},
		{Page: "abc"},
		{PageSize: "0"},
		{PageSize: "-5"},
	}
	for _, req := range cases {
		if _, err := client.GetFxRateList(context.Background(), req); err == nil {
			t.Fatalf("应拒绝非法分页参数：%+v", req)
		}
	}
}

// TestGetFxRateListServerError listId=0 时服务端返回 code=3，应作为业务错误抛出。
func TestGetFxRateListServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":3,"message":"[srpc异常] 获取行情列表服务异常","data":null}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetFxRateList(context.Background(), GetFxRateListRequest{ListID: "0"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != 3 {
		t.Fatalf("应返回 code=3 的业务错误，实际 %v", err)
	}
	if apiErr.Path != DefaultFxRateListPath {
		t.Fatalf("错误应带接口路径，实际 %s", apiErr.Path)
	}
}
