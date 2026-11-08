package quoteapiv2

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fxOverviewSample 汇率概览响应样本：data 直接是以货币对代码为键的对象。
const fxOverviewSample = `{
    "code": 0,
    "message": "成功",
    "data": {
        "USDCNH": {"time": 1793859536104, "priceNominal": "6.66623", "price": "6.66623"},
        "EURUSD": {"time": 1793859536488, "priceNominal": "1.12614", "price": "1.12614"},
        "USDJPY": {"time": 1793859536470, "priceNominal": "157.971", "price": "157.971"},
        "JPYUSD": {"time": 1793859509942, "priceNominal": "0.006330", "price": "0.006330"}
    }
}`

// TestGetFxOverviewWithTimestamp 带时间戳时请求携带 _，签名取实测向量。
func TestGetFxOverviewWithTimestamp(t *testing.T) {
	var (
		gotPath  string
		gotQuery map[string]string
		gotOrder []string
		gotToken string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotOrder, gotQuery = splitQuery(r.URL.RawQuery)
		gotToken = r.Header.Get("quote-token")
		_, _ = w.Write([]byte(fxOverviewSample))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	quotes, err := client.GetFxOverview(context.Background(), GetFxOverviewRequest{
		TimestampMillis: "1793859303524",
	})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}

	if gotPath != DefaultFxOverviewPath {
		t.Fatalf("路径期望 %s，实际 %s", DefaultFxOverviewPath, gotPath)
	}
	if len(gotOrder) != 1 || gotOrder[0] != "_" || gotQuery["_"] != "1793859303524" {
		t.Fatalf("查询参数应只有时间戳，实际 %v / %v", gotOrder, gotQuery)
	}
	if gotToken != "d1bea85060" {
		t.Fatalf("quote-token 期望 d1bea85060，实际 %s", gotToken)
	}

	if len(quotes) != 4 {
		t.Fatalf("货币对数量期望 4，实际 %d", len(quotes))
	}
	usdcnh, ok := quotes["USDCNH"]
	if !ok {
		t.Fatalf("应含 USDCNH 键，实际 %v", keysOf(quotes))
	}
	if usdcnh.Time != 1793859536104 || usdcnh.PriceNominal != "6.66623" || usdcnh.Price != "6.66623" {
		t.Fatalf("USDCNH 字段不符：%+v", usdcnh)
	}
	if jpyusd := quotes["JPYUSD"]; jpyusd.PriceNominal != "0.006330" {
		t.Fatalf("JPYUSD 精度应保留原始字符串：%+v", jpyusd)
	}
}

// TestGetFxOverviewWithoutParams 省略时间戳时不带查询串，
// 签名输入为空对象 {}（不是空串），token 取实测向量。
func TestGetFxOverviewWithoutParams(t *testing.T) {
	var (
		gotRawQuery string
		gotToken    string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		gotToken = r.Header.Get("quote-token")
		_, _ = w.Write([]byte(fxOverviewSample))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	if _, err := client.GetFxOverview(context.Background(), GetFxOverviewRequest{}); err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
	if gotRawQuery != "" {
		t.Fatalf("省略参数时不应带查询串，实际 %q", gotRawQuery)
	}
	// 空参数列表的签名输入是 {}，若走 Sign 的空串降级会得到 58af97268c（不匹配）。
	if gotToken != "279c67896d" {
		t.Fatalf("quote-token 期望 279c67896d，实际 %s", gotToken)
	}
}

// TestGetFxOverviewServerError 业务码非 0 时应返回 APIError。
func TestGetFxOverviewServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":-403,"message":"非法请求","data":null}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetFxOverview(context.Background(), GetFxOverviewRequest{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != CodeTokenInvalid {
		t.Fatalf("应返回 -403 业务错误，实际 %v", err)
	}
}

// TestGetFxOverviewEmptyData code=0 但 data 为空时返回空 map 而非 nil 报错。
func TestGetFxOverviewEmptyData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	quotes, err := client.GetFxOverview(context.Background(), GetFxOverviewRequest{})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
	if len(quotes) != 0 {
		t.Fatalf("空 data 应返回空 map，实际 %v", quotes)
	}
}

// TestFxOverviewQuoteJSON 条目结构按接口原名序列化，便于在日志中复用。
func TestFxOverviewQuoteJSON(t *testing.T) {
	raw, err := json.Marshal(FxOverviewQuote{Time: 1793859536104, PriceNominal: "6.66623", Price: "6.66623"})
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	want := `{"time":1793859536104,"priceNominal":"6.66623","price":"6.66623"}`
	if string(raw) != want {
		t.Fatalf("JSON 不符：期望 %s，实际 %s", want, raw)
	}
}

// keysOf 返回 map 的键，用于断言失败时输出可读内容。
func keysOf(m map[string]FxOverviewQuote) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
