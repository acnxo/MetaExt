package quoteapiv2

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stockListSample 股票列表响应样本：行情类字段为带单位或百分号的字符串，
// leverage 为 null，stockId 与 instrumentType 为数值。
const stockListSample = `{
    "code": 0,
    "message": "成功",
    "data": {
        "pagination": {"page": 1, "pageSize": 50, "pageCount": 65, "total": 3228},
        "list": [
            {
                "stockId": 58531814313881,
                "name": "洛阳钼业",
                "stockCode": "03993",
                "marketLabel": "HK",
                "instrumentType": 3,
                "changeRatio": "-4.30%",
                "priceDirect": "down",
                "change": "-0.860",
                "priceNominal": "19.120",
                "tradeTrunover": "14.33亿",
                "tradeVolumn": "7408.21万",
                "marketVal": "4090.59亿",
                "circulationMarketValue": "752.08亿",
                "totalShares": "213.94亿",
                "circulationTotalShares": "39.33亿",
                "c_5Days": "+15.18%",
                "c_5Days_priceDirect": "up",
                "c_10Days": "-2.00%",
                "c_10Days_priceDirect": "down",
                "c_20Days": "+6.40%",
                "c_20Days_priceDirect": "up",
                "c_60Days": "+5.03%",
                "c_60Days_priceDirect": "up",
                "c_120Days": "+6.56%",
                "c_120Days_priceDirect": "up",
                "c_250Days": "+206.62%",
                "c_250Days_priceDirect": "up",
                "c_YearDays": "+1.09%",
                "c_YearDays_priceDirect": "up",
                "trustAum": "0.00",
                "leverage": null,
                "tradeChangeraio": "1.88%",
                "priceAmplitude": "5.26%",
                "volumnRatio": "1.09",
                "buysellRatio": "0.00%"
            }
        ]
    }
}`

// TestGetStockListRequestAndParse 校验路径、参数键序、签名、响应解析与字段映射。
func TestGetStockListRequestAndParse(t *testing.T) {
	var (
		gotOrder   []string
		gotQuery   map[string]string
		gotHeaders http.Header
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != DefaultStockListPath {
			t.Errorf("路径期望 %s，实际 %s", DefaultStockListPath, r.URL.Path)
		}
		gotHeaders = r.Header.Clone()
		gotOrder, gotQuery = splitQuery(r.URL.RawQuery)
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(stockListSample))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	page, err := client.GetStockList(context.Background(), GetStockListRequest{
		MarketType:      "1",
		PlateType:       "1",
		RankType:        "3",
		Page:            "1",
		PageSize:        "50",
		TimestampMillis: "1781611867445",
	})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}

	wantOrder := []string{"marketType", "plateType", "rankType", "page", "pageSize", "_"}
	if strings.Join(gotOrder, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("查询键序不符：期望 %v，实际 %v", wantOrder, gotOrder)
	}
	if gotQuery["marketType"] != "1" || gotQuery["plateType"] != "1" {
		t.Fatalf("必填参数不符：%v", gotQuery)
	}
	if gotQuery["rankType"] != "3" || gotQuery["page"] != "1" || gotQuery["pageSize"] != "50" {
		t.Fatalf("可选参数不符：%v", gotQuery)
	}
	if gotQuery["_"] != "1781611867445" {
		t.Fatalf("时间戳未透传：%s", gotQuery["_"])
	}
	// 全参数 + 该时间戳的签名结果固定，可作为回归基线。
	if got := gotHeaders.Get("quote-token"); got != "6481798cf4" {
		t.Fatalf("quote-token 期望 6481798cf4，实际 %s", got)
	}

	if page.Pagination.Page != 1 || page.Pagination.PageSize != 50 ||
		page.Pagination.PageCount != 65 || page.Pagination.Total != 3228 {
		t.Fatalf("分页解析不符：%+v", page.Pagination)
	}
	if len(page.Items) != 1 {
		t.Fatalf("条目数期望 1，实际 %d", len(page.Items))
	}
	item := page.Items[0]
	if item.StockID != 58531814313881 || item.Name != "洛阳钼业" ||
		item.StockCode != "03993" || item.MarketLabel != "HK" || item.InstrumentType != 3 {
		t.Fatalf("基本字段不符：%+v", item)
	}
	if item.ChangeRatio != "-4.30%" || item.PriceDirect != "down" ||
		item.Change != "-0.860" || item.PriceNominal != "19.120" {
		t.Fatalf("涨跌字段不符：%+v", item)
	}
	if item.TradeTurnover != "14.33亿" || item.TradeVolume != "7408.21万" ||
		item.MarketValue != "4090.59亿" || item.CirculationMarketValue != "752.08亿" ||
		item.TotalShares != "213.94亿" || item.CirculationTotalShares != "39.33亿" {
		t.Fatalf("成交与市值字段不符：%+v", item)
	}
	if item.Change5Days != "+15.18%" || item.Change5DaysPriceDirect != "up" ||
		item.Change10Days != "-2.00%" || item.Change10DaysPriceDirect != "down" ||
		item.Change20Days != "+6.40%" || item.Change60Days != "+5.03%" ||
		item.Change120Days != "+6.56%" || item.Change250Days != "+206.62%" ||
		item.ChangeYearDays != "+1.09%" || item.ChangeYearDaysPriceDirect != "up" {
		t.Fatalf("多周期涨跌幅不符：%+v", item)
	}
	if item.TradeChangeRatio != "1.88%" || item.PriceAmplitude != "5.26%" ||
		item.VolumeRatio != "1.09" || item.BuySellRatio != "0.00%" {
		t.Fatalf("换手率等字段不符：%+v", item)
	}
	if item.Leverage != nil {
		t.Fatalf("leverage 为 null 时应保持 nil，实际 %v", item.Leverage)
	}
}

// TestGetStockListOmitsOptionalParams 只传必填参数时，其余键必须整键省略，
// plateType 按默认值补 1。
func TestGetStockListOmitsOptionalParams(t *testing.T) {
	var gotOrder []string
	var gotQuery map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrder, gotQuery = splitQuery(r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"code":0,"data":{"pagination":{},"list":[]}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	page, err := client.GetStockList(context.Background(), GetStockListRequest{MarketType: "1"})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("空列表应返回 0 条，实际 %d", len(page.Items))
	}
	want := "marketType,plateType"
	if strings.Join(gotOrder, ",") != want {
		t.Fatalf("查询参数应只含必填项与默认板块：期望 %s，实际 %v", want, gotOrder)
	}
	if gotQuery["plateType"] != DefaultPlateType {
		t.Fatalf("plateType 应取默认值 %s，实际 %s", DefaultPlateType, gotQuery["plateType"])
	}
	for _, key := range []string{"rankType", "page", "pageSize", "_"} {
		if _, ok := gotQuery[key]; ok {
			t.Fatalf("%s 缺省时应整键省略，实际出现在查询串中", key)
		}
	}
}

// TestGetStockListParamsError 服务端返回 500 时应可判定为参数错误。
func TestGetStockListParamsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":500,"message":"Params Error","data":null}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetStockList(context.Background(), GetStockListRequest{MarketType: "1"})
	if err == nil {
		t.Fatal("应返回错误")
	}
	if !IsParamsError(err) {
		t.Fatalf("应识别为参数错误（500），实际 %v", err)
	}
	if IsTokenRejected(err) {
		t.Fatal("参数错误不应被判定为 token 被拒")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != CodeParamsError || apiErr.Message != "Params Error" {
		t.Fatalf("错误内容不符：%v", err)
	}
	if apiErr.Path != DefaultStockListPath {
		t.Fatalf("错误应带接口路径，实际 %s", apiErr.Path)
	}
}

// TestGetStockListEmptyMarketType marketType 为空时本地直接报错，不发请求。
func TestGetStockListEmptyMarketType(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	if _, err := client.GetStockList(context.Background(), GetStockListRequest{MarketType: "  "}); err == nil {
		t.Fatal("marketType 为空应报错")
	}
	if called {
		t.Fatal("marketType 为空时不应发起请求")
	}
}

// TestGetStockListInvalidPaging 页码与每页数量必须是不小于 1 的整数。
func TestGetStockListInvalidPaging(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("非法分页参数不应发起请求")
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	cases := []GetStockListRequest{
		{MarketType: "1", Page: "0"},
		{MarketType: "1", Page: "abc"},
		{MarketType: "1", PageSize: "0"},
		{MarketType: "1", PageSize: "-5"},
	}
	for _, req := range cases {
		if _, err := client.GetStockList(context.Background(), req); err == nil {
			t.Fatalf("应拒绝非法分页参数：%+v", req)
		}
	}
}

// TestGetStockListLeverageNumber leverage 为数值时也能解析。
func TestGetStockListLeverageNumber(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"data":{"list":[{"stockId":1,"leverage":2.5}]}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	page, err := client.GetStockList(context.Background(), GetStockListRequest{MarketType: "1"})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Leverage == nil {
		t.Fatalf("leverage 应解析为非空，实际 %+v", page.Items)
	}
	if got := page.Items[0].Leverage.Float(); got != 2.5 {
		t.Fatalf("leverage 期望 2.5，实际 %v", got)
	}
}

// TestGetStockListHTTPError 非 200 应返回带状态码的错误。
func TestGetStockListHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("unavailable"))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetStockList(context.Background(), GetStockListRequest{MarketType: "1"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusServiceUnavailable {
		t.Fatalf("应返回 HTTPStatus=503 的错误，实际 %v", err)
	}
}
