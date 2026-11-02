package quoteapiv2

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// etfListSample ETF 列表响应样本：条目结构与股票列表一致，instrumentType 为 4，
// leverage 为字符串，trustAum 带单位。
const etfListSample = `{
    "code": 0,
    "message": "成功",
    "data": {
        "pagination": {"page": 0, "pageSize": 50, "pageCount": 122, "total": 6097},
        "list": [
            {
                "stockId": 202861,
                "name": "整体股市指数ETF-Vanguard",
                "stockCode": "VTI",
                "marketLabel": "US",
                "instrumentType": 4,
                "changeRatio": "+0.02%",
                "priceDirect": "up",
                "change": "+0.090",
                "priceNominal": "375.430",
                "tradeTrunover": "8.59亿",
                "tradeVolumn": "229.14万",
                "marketVal": "6917.55亿",
                "circulationMarketValue": "6917.55亿",
                "totalShares": "18.43亿",
                "circulationTotalShares": "18.43亿",
                "c_5Days": "-0.23%",
                "c_5Days_priceDirect": "down",
                "c_10Days": "-1.44%",
                "c_10Days_priceDirect": "down",
                "c_20Days": "-0.31%",
                "c_20Days_priceDirect": "down",
                "c_60Days": "+3.54%",
                "c_60Days_priceDirect": "up",
                "c_120Days": "+20.26%",
                "c_120Days_priceDirect": "up",
                "c_250Days": "+15.64%",
                "c_250Days_priceDirect": "up",
                "c_YearDays": "+12.65%",
                "c_YearDays_priceDirect": "up",
                "trustAum": "2.34万亿",
                "leverage": "1",
                "tradeChangeraio": "0.12%",
                "priceAmplitude": "0.58%",
                "volumnRatio": "0.81",
                "buysellRatio": "-50.00%"
            }
        ]
    }
}`

// TestGetEtfListRequestAndParse 校验路径、参数键序（含空 leverage）、签名与响应解析。
func TestGetEtfListRequestAndParse(t *testing.T) {
	var (
		gotOrder   []string
		gotQuery   map[string]string
		gotHeaders http.Header
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != DefaultEtfListPath {
			t.Errorf("路径期望 %s，实际 %s", DefaultEtfListPath, r.URL.Path)
		}
		gotHeaders = r.Header.Clone()
		gotOrder, gotQuery = splitQuery(r.URL.RawQuery)
		_, _ = w.Write([]byte(etfListSample))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	page, err := client.GetEtfList(context.Background(), GetEtfListRequest{
		RouteType:       "1",
		PlateID:         "10002799",
		RankType:        "6",
		Leverage:        "",
		Page:            "0",
		PageSize:        "50",
		TimestampMillis: "1789943966621",
	})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}

	wantOrder := []string{"routeType", "plateId", "rankType", "leverage", "page", "pageSize", "_"}
	if strings.Join(gotOrder, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("查询键序不符：期望 %v，实际 %v", wantOrder, gotOrder)
	}
	if value, ok := gotQuery["leverage"]; !ok || value != "" {
		t.Fatalf("leverage 必须以空串出现，实际 %v（存在=%v）", value, ok)
	}
	if gotQuery["page"] != "0" || gotQuery["pageSize"] != "50" {
		t.Fatalf("分页参数不符：%v", gotQuery)
	}
	if got := gotHeaders.Get("quote-token"); got != "217d908009" {
		t.Fatalf("quote-token 期望 217d908009，实际 %s", got)
	}

	if page.Pagination.Page != 0 || page.Pagination.PageSize != 50 ||
		page.Pagination.PageCount != 122 || page.Pagination.Total != 6097 {
		t.Fatalf("分页解析不符：%+v", page.Pagination)
	}
	if len(page.Items) != 1 {
		t.Fatalf("条目数期望 1，实际 %d", len(page.Items))
	}
	item := page.Items[0]
	if item.StockID != 202861 || item.StockCode != "VTI" || item.InstrumentType != 4 {
		t.Fatalf("基本字段不符：%+v", item)
	}
	if item.TrustAUM != "2.34万亿" || item.Leverage == nil || item.Leverage.String() != "1" {
		t.Fatalf("规模与杠杆字段不符：%+v", item)
	}
}

// TestGetEtfListDefaults 全空请求应补齐默认值，且 leverage 仍以空串参与签名。
func TestGetEtfListDefaults(t *testing.T) {
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
	page, err := client.GetEtfList(context.Background(), GetEtfListRequest{})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("空列表应返回 0 条，实际 %d", len(page.Items))
	}
	wantOrder := []string{"routeType", "plateId", "rankType", "leverage", "page", "pageSize"}
	if strings.Join(gotOrder, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("查询键序不符：期望 %v，实际 %v", wantOrder, gotOrder)
	}
	if gotQuery["routeType"] != DefaultEtfRouteType || gotQuery["plateId"] != DefaultEtfPlateID {
		t.Fatalf("默认值不符：%v", gotQuery)
	}
	if gotQuery["rankType"] != DefaultEtfRankType || gotQuery["page"] != DefaultEtfPage ||
		gotQuery["pageSize"] != DefaultEtfPageSize {
		t.Fatalf("默认值不符：%v", gotQuery)
	}
	if _, ok := gotQuery["_"]; ok {
		t.Fatal("时间戳为空时应省略")
	}
	if gotToken != "88cf5ca696" {
		t.Fatalf("默认参数的 quote-token 期望 88cf5ca696，实际 %s", gotToken)
	}
}

// TestGetEtfListInvalidPaging 页码不能为负，每页数量不能小于 1。
func TestGetEtfListInvalidPaging(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("非法分页参数不应发起请求")
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	cases := []GetEtfListRequest{
		{Page: "-1"},
		{Page: "abc"},
		{PageSize: "0"},
		{PageSize: "abc"},
	}
	for _, req := range cases {
		if _, err := client.GetEtfList(context.Background(), req); err == nil {
			t.Fatalf("应拒绝非法分页参数：%+v", req)
		}
	}
}

// TestGetEtfListTokenRejected 省略 leverage 键会导致签名不匹配，服务端返回 -403。
func TestGetEtfListTokenRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":-403,"message":"非法请求","data":null}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetEtfList(context.Background(), GetEtfListRequest{})
	if err == nil || !IsTokenRejected(err) {
		t.Fatalf("应识别为 token 被拒（-403），实际 %v", err)
	}
}

// TestGetEtfListParamsError 参数非法时服务端返回 500。
func TestGetEtfListParamsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":500,"message":"Params Error","data":null}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetEtfList(context.Background(), GetEtfListRequest{PlateID: "1"})
	if err == nil || !IsParamsError(err) {
		t.Fatalf("应识别为参数错误（500），实际 %v", err)
	}
}

// TestGetEtfListHTTPError 非 200 应返回带状态码的错误。
func TestGetEtfListHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetEtfList(context.Background(), GetEtfListRequest{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusBadGateway {
		t.Fatalf("应返回 HTTPStatus=502 的错误，实际 %v", err)
	}
}
