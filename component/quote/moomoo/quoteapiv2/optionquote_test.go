package quoteapiv2

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// optionQuoteSample 期权报价响应样本：data 是以合约 ID 为键的对象，字段全为字符串。
const optionQuoteSample = `{
    "code": 0,
    "message": "成功",
    "data": {
        "505895381": {
            "priceBid": "8.600",
            "volumeBid": "29",
            "priceAsk": "9.900",
            "volumeAsk": "13",
            "price": "6.500",
            "changeRatio": "-65.68%",
            "change": "-12.440",
            "volume": "15",
            "turnover": "9705.00",
            "priceDirect": "down"
        }
    }
}`

// TestBuildOptionQuoteBody 请求体必须是紧凑 JSON，键序固定，id 为字符串、价差表代码为整数。
func TestBuildOptionQuoteBody(t *testing.T) {
	body, err := buildOptionQuoteBody([]OptionQuoteID{
		{ID: "505895381", SpreadTableCode: 81},
		{ID: " 505928056 ", SpreadTableCode: 81},
	})
	if err != nil {
		t.Fatalf("构造失败: %v", err)
	}
	want := `{"ids":[{"id":"505895381","spreadTableCode":81},{"id":"505928056","spreadTableCode":81}]}`
	if string(body) != want {
		t.Fatalf("请求体不符：\n期望 %s\n实际 %s", want, body)
	}

	// id 中的特殊字符按 JSON 规则转义，不破坏请求体结构。
	escaped, err := buildOptionQuoteBody([]OptionQuoteID{{ID: `a"b`, SpreadTableCode: 0}})
	if err != nil {
		t.Fatalf("构造失败: %v", err)
	}
	if got, want := string(escaped), `{"ids":[{"id":"a\"b","spreadTableCode":0}]}`; got != want {
		t.Fatalf("转义不符：期望 %s，实际 %s", want, got)
	}
}

// TestBuildOptionQuoteBodyInvalid 空列表与空 ID 应本地报错。
func TestBuildOptionQuoteBodyInvalid(t *testing.T) {
	if _, err := buildOptionQuoteBody(nil); err == nil {
		t.Fatal("ids 为空应报错")
	}
	if _, err := buildOptionQuoteBody([]OptionQuoteID{{ID: "  "}}); err == nil {
		t.Fatal("id 为空应报错")
	}
}

// TestGetOptionQuoteRequestAndParse 校验 POST 装配（路径、无查询串、请求头、body 签名）与解析。
func TestGetOptionQuoteRequestAndParse(t *testing.T) {
	var (
		gotMethod  string
		gotPath    string
		gotQuery   string
		gotBody    string
		gotHeaders http.Header
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotHeaders = r.Header.Clone()
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("读取请求体失败: %v", err)
		}
		gotBody = string(raw)
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(optionQuoteSample))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	quotes, err := client.GetOptionQuote(context.Background(), GetOptionQuoteRequest{
		IDs: []OptionQuoteID{
			{ID: "505895381", SpreadTableCode: 81},
			{ID: "505928056", SpreadTableCode: 81},
		},
	})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("方法期望 POST，实际 %s", gotMethod)
	}
	if gotPath != DefaultOptionQuotePath {
		t.Fatalf("路径期望 %s，实际 %s", DefaultOptionQuotePath, gotPath)
	}
	if gotQuery != "" {
		t.Fatalf("该接口不应带查询串，实际 %q", gotQuery)
	}
	wantBody := `{"ids":[{"id":"505895381","spreadTableCode":81},{"id":"505928056","spreadTableCode":81}]}`
	if gotBody != wantBody {
		t.Fatalf("请求体不符：\n期望 %s\n实际 %s", wantBody, gotBody)
	}
	if got := gotHeaders.Get("quote-token"); got != "d28d1397f7" {
		t.Fatalf("quote-token 期望 d28d1397f7，实际 %s", got)
	}
	if got := gotHeaders.Get("content-type"); got != "application/json" {
		t.Fatalf("content-type 不符：%s", got)
	}
	if got := gotHeaders.Get("origin"); got != DefaultOrigin {
		t.Fatalf("origin 不符：%s", got)
	}
	if got := gotHeaders.Get("futu-x-csrf-token"); got != "abcdef0123" {
		t.Fatalf("csrf 头不符：%s", got)
	}

	if len(quotes) != 1 {
		t.Fatalf("报价条数期望 1，实际 %d", len(quotes))
	}
	quote, ok := quotes["505895381"]
	if !ok {
		t.Fatalf("结果应包含键 505895381，实际 %v", quotes)
	}
	if quote.PriceBid != "8.600" || quote.VolumeBid != "29" ||
		quote.PriceAsk != "9.900" || quote.VolumeAsk != "13" {
		t.Fatalf("买卖盘字段不符：%+v", quote)
	}
	if quote.Price != "6.500" || quote.ChangeRatio != "-65.68%" ||
		quote.Change != "-12.440" || quote.Volume != "15" ||
		quote.Turnover != "9705.00" || quote.PriceDirect != "down" {
		t.Fatalf("行情字段不符：%+v", quote)
	}
}

// TestGetOptionQuoteEmptyData 已过期或已摘牌的合约返回空对象，不应视为错误。
func TestGetOptionQuoteEmptyData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	quotes, err := client.GetOptionQuote(context.Background(), GetOptionQuoteRequest{
		IDs: []OptionQuoteID{{ID: "505895381", SpreadTableCode: 81}},
	})
	if err != nil {
		t.Fatalf("空对象不应报错: %v", err)
	}
	if quotes == nil || len(quotes) != 0 {
		t.Fatalf("应返回空表，实际 %v", quotes)
	}
}

// TestGetOptionQuoteParamsError 价差表代码与市场不匹配时服务端返回 500。
func TestGetOptionQuoteParamsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":500,"message":"Params Error","data":null}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetOptionQuote(context.Background(), GetOptionQuoteRequest{
		IDs: []OptionQuoteID{{ID: "505895381", SpreadTableCode: 1}},
	})
	if err == nil || !IsParamsError(err) {
		t.Fatalf("应识别为参数错误，实际 %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Path != DefaultOptionQuotePath {
		t.Fatalf("错误应带接口路径，实际 %v", err)
	}
}

// TestGetOptionQuoteEmptyIDs 没有合约时本地直接报错，不发请求。
func TestGetOptionQuoteEmptyIDs(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	if _, err := client.GetOptionQuote(context.Background(), GetOptionQuoteRequest{}); err == nil {
		t.Fatal("ids 为空应报错")
	}
	if called {
		t.Fatal("ids 为空时不应发起请求")
	}
}

// TestGetOptionQuoteParseError 响应不是合法 JSON 时应给出解析失败错误。
func TestGetOptionQuoteParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>not json</html>"))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetOptionQuote(context.Background(), GetOptionQuoteRequest{
		IDs: []OptionQuoteID{{ID: "1", SpreadTableCode: 81}},
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !strings.Contains(apiErr.Message, "响应解析失败") {
		t.Fatalf("应返回解析失败错误，实际 %v", err)
	}
	if IsTokenRejected(err) {
		t.Fatal("解析失败不应被判定为 token 被拒")
	}
}

// TestGetOptionQuoteHTTPError 非 200 应返回带状态码的错误。
func TestGetOptionQuoteHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetOptionQuote(context.Background(), GetOptionQuoteRequest{
		IDs: []OptionQuoteID{{ID: "1", SpreadTableCode: 81}},
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusInternalServerError {
		t.Fatalf("应返回 HTTPStatus=500 的错误，实际 %v", err)
	}
}
