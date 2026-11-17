package indexapi

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

func TestGetIndexPerfRequestAndParse(t *testing.T) {
	var gotCode, gotBegin, gotEnd, gotReferer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotCode, gotBegin, gotEnd = q.Get("indexCode"), q.Get("startDate"), q.Get("endDate")
		gotReferer = r.Header.Get("referer")
		_, _ = w.Write([]byte(`{"code":200,"data":[
		  {"tradeDate":"20261113","open":"6201.33","high":"6230.11","low":"6170.55",
		   "close":"6188.07","change":"-13.26","changeRatio":"-0.21"},
		  {"tradeDate":"20261112","open":"6123.45","high":"6215.08","low":"6110.24",
		   "close":"6201.33","change":"12.00","changeRatio":"0.19"}
		]}`))
	}))
	defer server.Close()

	result, err := newTestClient(t, server).GetIndexPerf(context.Background(), GetIndexPerfRequest{
		IndexCode: "932000",
		Begin:     "2023-08-11",
		End:       "2026-11-13",
	})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if gotCode != "932000" || gotBegin != "20230811" || gotEnd != "20261113" {
		t.Errorf("查询参数不符：code=%s start=%s end=%s", gotCode, gotBegin, gotEnd)
	}
	if gotReferer != DefaultReferer {
		t.Errorf("referer = %q，期望 %q", gotReferer, DefaultReferer)
	}
	if result.IndexCode != "932000" || len(result.Bars) != 2 {
		t.Fatalf("结果不符：%+v", result)
	}
	// 必须按日期升序。
	if result.Bars[0].Date != "2026-11-12" || result.Bars[1].Date != "2026-11-13" {
		t.Fatalf("排序不符：%+v", result.Bars)
	}
	last := result.Bars[1]
	if last.Close != 6188.07 || last.Open != 6201.33 || last.High != 6230.11 ||
		last.Low != 6170.55 || last.Change != -13.26 || last.ChangeRatio != -0.21 {
		t.Fatalf("字段解析不符：%+v", last)
	}
}

func TestGetIndexPerfDedupesByDate(t *testing.T) {
	// 官网偶有重复交易日，保留最后一条。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":200,"data":[
		  {"tradeDate":"2026-11-13","close":"1.00"},
		  {"tradeDate":"20261113","close":"2.00"}
		]}`))
	}))
	defer server.Close()

	result, err := newTestClient(t, server).GetIndexPerf(context.Background(), GetIndexPerfRequest{IndexCode: "000985"})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if len(result.Bars) != 1 || result.Bars[0].Close != 2.00 {
		t.Fatalf("去重结果不符：%+v", result.Bars)
	}
}

func TestGetIndexPerfEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
	}))
	defer server.Close()

	result, err := newTestClient(t, server).GetIndexPerf(context.Background(), GetIndexPerfRequest{IndexCode: "932000"})
	if err != nil {
		t.Fatalf("空数据不应报错：%v", err)
	}
	if len(result.Bars) != 0 {
		t.Fatalf("期望空 Bars：%+v", result.Bars)
	}
}

func TestGetIndexPerfMissingCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("缺少指数代码不应发起请求")
	}))
	defer server.Close()

	if _, err := newTestClient(t, server).GetIndexPerf(context.Background(), GetIndexPerfRequest{}); err == nil {
		t.Fatal("缺少 IndexCode 应报错")
	}
}

func TestGetIndexPerfInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>blocked</html>"))
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetIndexPerf(context.Background(), GetIndexPerfRequest{IndexCode: "932000"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.URL == "" {
		t.Fatalf("应返回带 URL 的 *APIError：%v", err)
	}
}

func TestGetIndexPerfHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetIndexPerf(context.Background(), GetIndexPerfRequest{IndexCode: "932000"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusServiceUnavailable {
		t.Fatalf("应返回 HTTPStatus=503 的错误：%v", err)
	}
}

func TestNormalizeAndCompactDate(t *testing.T) {
	cases := map[string]string{
		`"20261113"`:   "2026-11-13",
		`"2026-11-13"`: "2026-11-13",
		`"2026/11/13"`: "2026-11-13",
		`""`:           "",
		`"abc"`:        "",
	}
	for raw, want := range cases {
		if got := normalizeDate([]byte(raw)); got != want {
			t.Errorf("normalizeDate(%s) = %q，期望 %q", raw, got, want)
		}
	}
	if got := compactDate("2026-11-13"); got != "20261113" {
		t.Errorf("compactDate = %q", got)
	}
	if got := compactDate("20261113"); got != "20261113" {
		t.Errorf("compactDate 幂等性不符：%q", got)
	}
}

func TestConfigDefaults(t *testing.T) {
	got := Config{}.withDefaults()
	if got.BaseURL != DefaultBaseURL || got.IndexPerfPath != DefaultIndexPerfPath ||
		got.Referer != DefaultReferer {
		t.Fatalf("默认值未补齐：%+v", got)
	}
}
