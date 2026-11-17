package finscope

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestGetQuoteForexMinuteRiskControl(t *testing.T) {
	// 实测：该 group 在本机持续返回风控，响应用例即真实形态。
	client, rec := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, `{"QueryID":"7852813130838806383","ResultCode":0,"Result":{"code":403,"isCaptchaEnabled":true,"msg":"hit risk"}}`)
	})
	_, err := client.GetQuoteForexMinute(context.Background(), GetQuoteForexMinuteRequest{Code: "usdcny"})
	if !IsRiskControl(err) {
		t.Fatalf("应识别为风控，实际 %v", err)
	}
	// 外汇没有 market_type，也没有 srcid / pointType / newFormat。
	want := "group=huilv_minute&need_reverse_real=1&code=USDCNY&finClientType=pc"
	if rec.query != want {
		t.Errorf("查询串 = %q，期望 %q（货币对应转大写）", rec.query, want)
	}
}

func TestGetQuoteForexMinuteRawResult(t *testing.T) {
	// 结构未验证，正常响应也只应把 Result 原样交出。
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, `{"QueryID":"1","ResultCode":0,"Result":{"whatever":{"a":1}}}`)
	})
	result, err := client.GetQuoteForexMinute(context.Background(), GetQuoteForexMinuteRequest{Code: "BTCUSD"})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if !strings.Contains(string(result.Result), "whatever") {
		t.Errorf("Result 未原样返回：%s", result.Result)
	}
	if result.QueryID != "1" {
		t.Errorf("QueryID = %q", result.QueryID)
	}
}

func TestGetQuoteForexMinuteValidates(t *testing.T) {
	client := New(Config{})
	cases := map[string]GetQuoteForexMinuteRequest{
		"code 为空":               {},
		"code 长度不对":             {Code: "USD"},
		"need_reverse_real 非 1": {Code: "USDCNY", NeedReverseReal: 2},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := client.GetQuoteForexMinute(context.Background(), req); err == nil {
				t.Fatal("参数非法应返回错误")
			}
		})
	}
}
