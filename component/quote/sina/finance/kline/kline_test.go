package kline

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

func TestGetKLineRequestAndParse(t *testing.T) {
	var gotSymbol, gotScale, gotMA, gotDataLen, gotReferer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotSymbol, gotScale = q.Get("symbol"), q.Get("scale")
		gotMA, gotDataLen = q.Get("ma"), q.Get("datalen")
		gotReferer = r.Header.Get("referer")
		_, _ = w.Write([]byte(`[
		  {"day":"2026-11-12","open":"11.700","high":"12.000","low":"11.650","close":"11.990","volume":"198765432"},
		  {"day":"2026-11-13","open":"11.760","high":"12.100","low":"11.700","close":"12.070","volume":"218942998"},
		  {"day":"2026-11-14","open":"0","high":"0","low":"0","close":"0","volume":"0"}
		]`))
	}))
	defer server.Close()

	result, err := newTestClient(t, server).GetKLine(context.Background(), GetKLineRequest{
		Symbol:  "sz399303",
		DataLen: 6000,
	})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if gotSymbol != "sz399303" || gotScale != "240" || gotMA != "no" || gotDataLen != "6000" {
		t.Errorf("查询参数不符：symbol=%s scale=%s ma=%s datalen=%s",
			gotSymbol, gotScale, gotMA, gotDataLen)
	}
	if gotReferer != DefaultReferer {
		t.Errorf("referer = %q，期望 %q", gotReferer, DefaultReferer)
	}
	// 收盘为 0 的记录应被剔除。
	if len(result.Bars) != 2 {
		t.Fatalf("条目数 = %d，期望 2", len(result.Bars))
	}
	bar := result.Bars[1]
	if bar.Date != "2026-11-13" || bar.Open != 11.76 || bar.Close != 12.07 ||
		bar.High != 12.10 || bar.Low != 11.70 {
		t.Fatalf("OHLC 解析不符：%+v", bar)
	}
	// 成交量单位是股，不做换算。
	if bar.Volume != 218942998 {
		t.Errorf("成交量 = %v，期望 218942998", bar.Volume)
	}
}

func TestGetKLineEmptyArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	result, err := newTestClient(t, server).GetKLine(context.Background(), GetKLineRequest{Code: "000001"})
	if err != nil {
		t.Fatalf("空数组不应报错：%v", err)
	}
	if len(result.Bars) != 0 || result.Symbol != "sz000001" {
		t.Fatalf("结果不符：%+v", result)
	}
}

func TestGetKLineNonJSON(t *testing.T) {
	// 站点限频时会返回 null 或错误页。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("null"))
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetKLine(context.Background(), GetKLineRequest{Code: "000001"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.URL == "" {
		t.Fatalf("应返回带 URL 的 *APIError：%v", err)
	}
}

func TestGetKLineHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetKLine(context.Background(), GetKLineRequest{Code: "000001"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusForbidden {
		t.Fatalf("应返回 HTTPStatus=403 的错误：%v", err)
	}
}

func TestResolveSymbol(t *testing.T) {
	cases := map[string]string{
		"600000": "sh600000", "000001": "sz000001", "399303": "sz399303",
		// 北交所（4/8 开头）必须用 bj 前缀。
		"830799": "bj830799", "430047": "bj430047",
	}
	for code, want := range cases {
		got, err := resolveSymbol(GetKLineRequest{Code: code})
		if err != nil || got != want {
			t.Errorf("resolveSymbol(%s) = %q, %v；期望 %q", code, got, err, want)
		}
	}
	// Symbol 优先。
	if got, _ := resolveSymbol(GetKLineRequest{Symbol: "sz399303", Code: "600000"}); got != "sz399303" {
		t.Errorf("Symbol 应优先：%s", got)
	}
	if _, err := resolveSymbol(GetKLineRequest{}); err == nil {
		t.Fatal("缺少 Symbol 与 Code 时应报错")
	}
}

func TestConfigDefaults(t *testing.T) {
	got := Config{}.withDefaults()
	if got.BaseURL != DefaultBaseURL || got.KLinePath != DefaultKLinePath ||
		got.Referer != DefaultReferer {
		t.Fatalf("默认值未补齐：%+v", got)
	}
}
