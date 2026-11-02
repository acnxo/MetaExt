package quoteapiv2

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// cashTrendSample 资金流水响应样本：金额与时间全为字符串，比率是百分数乘以 100。
const cashTrendSample = `{
    "code": 0,
    "message": "成功",
    "data": {
        "hasMore": true,
        "list": [
            {
                "time": "1786075200",
                "totalNetIn": "-700324120157",
                "superNetIn": "9752295125",
                "bigNetIn": "-293476434557",
                "midNetIn": "-248713964950",
                "smlNetIn": "-167886015775",
                "mainNetIn": "-283724139432",
                "priceChangeRatio": "117",
                "closePrice": "723.03",
                "mainDealRatio": "2897",
                "accMainNetIn": "-283724139432"
            }
        ]
    }
}`

// TestGetHistoryCashTrendRequestAndParse 校验路径、参数键序、签名与记录解析。
func TestGetHistoryCashTrendRequestAndParse(t *testing.T) {
	var (
		gotOrder   []string
		gotQuery   map[string]string
		gotHeaders http.Header
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != DefaultHistoryCashTrendPath {
			t.Errorf("路径期望 %s，实际 %s", DefaultHistoryCashTrendPath, r.URL.Path)
		}
		gotHeaders = r.Header.Clone()
		gotOrder, gotQuery = splitQuery(r.URL.RawQuery)
		_, _ = w.Write([]byte(cashTrendSample))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	page, err := client.GetHistoryCashTrend(context.Background(), GetHistoryCashTrendRequest{
		StockID:         "203290",
		Type:            CashTrendTypeDay,
		BaseTime:        "0",
		TimestampMillis: "1789943637356",
	})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}

	wantOrder := []string{"stockId", "type", "baseTime", "_"}
	if strings.Join(gotOrder, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("查询键序不符：期望 %v，实际 %v", wantOrder, gotOrder)
	}
	if gotQuery["stockId"] != "203290" || gotQuery["type"] != CashTrendTypeDay ||
		gotQuery["baseTime"] != "0" {
		t.Fatalf("参数值不符：%v", gotQuery)
	}
	if got := gotHeaders.Get("quote-token"); got != "44346db4fa" {
		t.Fatalf("quote-token 期望 44346db4fa，实际 %s", got)
	}

	if !page.HasMore {
		t.Fatal("hasMore 应为 true")
	}
	if len(page.Items) != 1 {
		t.Fatalf("记录数期望 1，实际 %d", len(page.Items))
	}
	item := page.Items[0]
	if item.Time != "1786075200" || item.TotalNetIn != "-700324120157" ||
		item.SuperNetIn != "9752295125" || item.BigNetIn != "-293476434557" ||
		item.MidNetIn != "-248713964950" || item.SmallNetIn != "-167886015775" ||
		item.MainNetIn != "-283724139432" || item.AccMainNetIn != "-283724139432" {
		t.Fatalf("金额字段不符：%+v", item)
	}
	if item.PriceChangeRatio != "117" || item.MainDealRatio != "2897" || item.ClosePrice != "723.03" {
		t.Fatalf("比率与收盘价不符（应保留原始字符串）：%+v", item)
	}
}

// TestGetHistoryCashTrendEmptyType type 为空时本地直接报错，不发请求。
func TestGetHistoryCashTrendEmptyType(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetHistoryCashTrend(context.Background(), GetHistoryCashTrendRequest{StockID: "203290"})
	if err == nil || !strings.Contains(err.Error(), "type") {
		t.Fatalf("type 为空应报错并提示原因，实际 %v", err)
	}
	if called {
		t.Fatal("type 为空时不应发起请求，避免服务端静默返回空列表")
	}
}

// TestGetHistoryCashTrendEmptyStockID stockId 为空时本地直接报错。
func TestGetHistoryCashTrendEmptyStockID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("stockId 为空时不应发起请求")
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	if _, err := client.GetHistoryCashTrend(context.Background(), GetHistoryCashTrendRequest{Type: CashTrendTypeDay}); err == nil {
		t.Fatal("stockId 为空应报错")
	}
}

// TestGetHistoryCashTrendDefaultBaseTime baseTime 为空时应补 0，并省略时间戳。
func TestGetHistoryCashTrendDefaultBaseTime(t *testing.T) {
	var gotQuery map[string]string
	var gotToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("quote-token")
		_, gotQuery = splitQuery(r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"code":0,"data":{"hasMore":false,"list":[]}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	page, err := client.GetHistoryCashTrend(context.Background(), GetHistoryCashTrendRequest{
		StockID: "203290",
		Type:    CashTrendTypeDay,
	})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
	if page.HasMore || len(page.Items) != 0 {
		t.Fatalf("空列表解析不符：%+v", page)
	}
	if gotQuery["baseTime"] != DefaultCashTrendBaseTime {
		t.Fatalf("baseTime 应补默认值 0，实际 %q", gotQuery["baseTime"])
	}
	if _, ok := gotQuery["_"]; ok {
		t.Fatal("时间戳为空时应省略")
	}
	if gotToken != "cca66938e3" {
		t.Fatalf("默认参数的 quote-token 期望 cca66938e3，实际 %s", gotToken)
	}
}

// TestGetHistoryCashTrendSilentEmpty 空列表属于正常响应，不应转成错误。
func TestGetHistoryCashTrendSilentEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"hasMore":false,"list":[]}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	page, err := client.GetHistoryCashTrend(context.Background(), GetHistoryCashTrendRequest{
		StockID: "1",
		Type:    CashTrendTypeDay,
	})
	if err != nil {
		t.Fatalf("空列表不应报错: %v", err)
	}
	if page.HasMore || page.Items == nil || len(page.Items) != 0 {
		t.Fatalf("应返回空列表：%+v", page)
	}
}

// TestGetHistoryCashTrendBusinessError 其它业务码应返回可判定的 APIError。
func TestGetHistoryCashTrendBusinessError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":500,"message":"Params Error","data":null}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetHistoryCashTrend(context.Background(), GetHistoryCashTrendRequest{
		StockID: "203290",
		Type:    "9",
	})
	if err == nil || !IsParamsError(err) {
		t.Fatalf("应识别为参数错误（500），实际 %v", err)
	}
}

// TestGetHistoryCashTrendHTTPError 非 200 应返回带状态码的错误。
func TestGetHistoryCashTrendHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limited"))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetHistoryCashTrend(context.Background(), GetHistoryCashTrendRequest{
		StockID: "203290",
		Type:    CashTrendTypeDay,
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusTooManyRequests {
		t.Fatalf("应返回 HTTPStatus=429 的错误，实际 %v", err)
	}
}
