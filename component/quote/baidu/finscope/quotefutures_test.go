package finscope

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestGetQuoteFuturesMinuteRiskControl(t *testing.T) {
	client, rec := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, `{"ResultCode":0,"Result":{"code":403,"isCaptchaEnabled":true,"msg":"hit risk"}}`)
	})
	_, err := client.GetQuoteFuturesMinute(context.Background(), GetQuoteFuturesMinuteRequest{Code: "au888"})
	if !IsRiskControl(err) {
		t.Fatalf("应识别为风控，实际 %v", err)
	}
	want := "srcid=5353&all=1&code=AU888&query=AU888&eprop=min&financeType=futures" +
		"&group=quotation_futures_minute&market_type=ab&finClientType=pc"
	if rec.query != want {
		t.Errorf("查询串 = %q，期望 %q", rec.query, want)
	}
}

func TestGetQuoteFuturesFiveDayRiskControl(t *testing.T) {
	client, rec := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		// 该接口风控时 QueryID 可能为 "0"。
		writeJSON(w, `{"QueryID":"0","ResultCode":0,"Result":{"code":403,"isCaptchaEnabled":true,"msg":"hit risk"}}`)
	})
	_, err := client.GetQuoteFuturesFiveDay(context.Background(), GetQuoteFuturesFiveDayRequest{Code: "AU888", MarketType: MarketUS})
	if !IsRiskControl(err) {
		t.Fatalf("应识别为风控，实际 %v", err)
	}
	// 5 个 is* 开关必须且只能 isFutures=true。
	if !strings.Contains(rec.query, "isFutures=true") ||
		strings.Contains(rec.query, "isStock=true") || strings.Contains(rec.query, "isIndex=true") {
		t.Errorf("品种开关构造错误：%s", rec.query)
	}
	if !strings.Contains(rec.query, "group=quotation_futures_fiveday") ||
		!strings.Contains(rec.query, "market_type=us") {
		t.Errorf("分组或市场不符：%s", rec.query)
	}
}

func TestGetQuoteFuturesFiveDayUsesOwnPath(t *testing.T) {
	var path string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		writeJSON(w, `{"ResultCode":0,"Result":null}`)
	})
	if _, err := client.GetQuoteFuturesFiveDay(context.Background(), GetQuoteFuturesFiveDayRequest{Code: "AU888"}); err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	// 期货五日走 selfselect 路径，不是 vapi/v1/getquotation。
	if path != DefaultFuturesFiveDayPath {
		t.Errorf("请求路径 = %q，期望 %q", path, DefaultFuturesFiveDayPath)
	}
}

func TestGetQuoteFuturesValidates(t *testing.T) {
	client := New(Config{})
	if _, err := client.GetQuoteFuturesMinute(context.Background(), GetQuoteFuturesMinuteRequest{}); err == nil {
		t.Fatal("code 为空应返回错误")
	}
	if _, err := client.GetQuoteFuturesMinute(context.Background(), GetQuoteFuturesMinuteRequest{Code: "AU888", MarketType: "hk"}); err == nil {
		t.Fatal("期货市场只能是 ab / us")
	}
	if _, err := client.GetQuoteFuturesFiveDay(context.Background(), GetQuoteFuturesFiveDayRequest{Code: "GC26Z", MarketType: "hk"}); err == nil {
		t.Fatal("期货市场只能是 ab / us")
	}
}
