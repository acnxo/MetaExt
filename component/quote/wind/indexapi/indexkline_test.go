package indexapi

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	return New(Config{BaseURL: server.URL, Timeout: 5 * time.Second, Sleep: func(time.Duration) {}})
}

const klineBody = `{"Result":[
  {"tradeDate":1794499200000,"open":"6201.33","hight":"6230.11","low":"6170.55",
   "close":"6188.07","volume":12345678,"amount":9876543210,"pctChange":"-0.21"},
  {"tradeDate":1794412800000,"open":"6123.45","hight":"6215.08","low":"6110.24",
   "close":"6201.33","volume":11223344,"amount":8765432109,"pctChange":"0.19"}
]}`

func TestGetIndexKLineRequestAndParse(t *testing.T) {
	var gotIndexID, gotLang string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotIndexID, gotLang = q.Get("indexId"), q.Get("lan")
		_, _ = w.Write([]byte(klineBody))
	}))
	defer server.Close()

	result, err := newTestClient(t, server).GetIndexKLine(context.Background(), GetIndexKLineRequest{
		IndexCode: "8841431.WI",
	})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	// indexId 必须按映射表填入，并原样（含 + / =）编码后传输。
	if gotIndexID != "2O5u8vTCa4KiLTLQ7DT/9w==" {
		t.Errorf("indexId = %q", gotIndexID)
	}
	if gotLang != "cn" {
		t.Errorf("lan = %q，期望 cn", gotLang)
	}
	if result.IndexCode != "8841431.WI" || result.IndexID != "2O5u8vTCa4KiLTLQ7DT/9w==" {
		t.Fatalf("结果标识不符：%+v", result)
	}
	if len(result.Bars) != 2 {
		t.Fatalf("条目数 = %d，期望 2", len(result.Bars))
	}
	// 必须按日期升序。
	if result.Bars[0].Date != "2026-11-12" || result.Bars[1].Date != "2026-11-13" {
		t.Fatalf("排序或日期换算不符：%+v", result.Bars)
	}
	last := result.Bars[1]
	if last.Close != 6188.07 || last.Open != 6201.33 || last.High != 6230.11 ||
		last.Low != 6170.55 || last.Volume != 12345678 || last.Amount != 9876543210 ||
		last.ChangeRatio != -0.21 {
		t.Fatalf("字段解析不符：%+v", last)
	}
}

func TestGetIndexKLineExplicitIndexID(t *testing.T) {
	var gotIndexID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIndexID = r.URL.Query().Get("indexId")
		_, _ = w.Write([]byte(`{"Result":[]}`))
	}))
	defer server.Close()

	// 显式 IndexID 优先，可绕过映射表。
	result, err := newTestClient(t, server).GetIndexKLine(context.Background(), GetIndexKLineRequest{
		IndexCode: "999999.WI",
		IndexID:   "abc+/=",
	})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if gotIndexID != "abc+/=" {
		t.Errorf("indexId = %q，期望 abc+/=", gotIndexID)
	}
	if len(result.Bars) != 0 {
		t.Fatalf("期望空 Bars：%+v", result.Bars)
	}
}

func TestGetIndexKLineUnknownCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("未知 windCode 不应发起请求")
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetIndexKLine(context.Background(), GetIndexKLineRequest{
		IndexCode: "000000.XX",
	})
	if err == nil || !strings.Contains(err.Error(), "未收录") {
		t.Fatalf("未收录的 windCode 应报错：%v", err)
	}

	if _, err := newTestClient(t, server).GetIndexKLine(context.Background(), GetIndexKLineRequest{}); err == nil {
		t.Fatal("IndexCode 与 IndexID 都为空时应报错")
	}
}

func TestGetIndexKLineDedupesByDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"Result":[
		  {"tradeDate":1794499200000,"close":"1.00"},
		  {"tradeDate":1794499200000,"close":"2.00"}
		]}`))
	}))
	defer server.Close()

	result, err := newTestClient(t, server).GetIndexKLine(context.Background(), GetIndexKLineRequest{
		IndexCode: "868008.WI",
	})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if len(result.Bars) != 1 || result.Bars[0].Close != 2.00 {
		t.Fatalf("同日期应保留最后一条：%+v", result.Bars)
	}
}

func TestGetIndexKLineHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetIndexKLine(context.Background(), GetIndexKLineRequest{
		IndexCode: "8841431.WI",
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusBadGateway {
		t.Fatalf("应返回 HTTPStatus=502 的错误：%v", err)
	}
}

func TestNewTLSConfigSkipsHostnameOnly(t *testing.T) {
	// 只跳过主机名校验：证书链校验仍然要执行，空链必须报错。
	cfg := newTLSConfig()
	if !cfg.InsecureSkipVerify {
		t.Error("应关闭默认的链+主机名校验（随后由 VerifyConnection 补回链校验）")
	}
	if cfg.VerifyConnection == nil {
		t.Fatal("必须提供 VerifyConnection 做链校验")
	}
	if err := cfg.VerifyConnection(tls.ConnectionState{}); err == nil {
		t.Fatal("空证书链应返回错误")
	}
}

func TestConfigDefaults(t *testing.T) {
	got := Config{}.withDefaults()
	if got.BaseURL != DefaultBaseURL || got.IndexKLinePath != DefaultIndexKLinePath {
		t.Fatalf("默认值未补齐：%+v", got)
	}
	if len(got.IndexIDs) != len(DefaultIndexIDs) {
		t.Fatalf("indexId 映射未补齐：%+v", got.IndexIDs)
	}
}
