package push2

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newTestClient 构造只指向测试服务器的客户端（镜像列表只有一个），并跳过退避等待。
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	return New(Config{
		ListHosts:  []string{server.URL},
		KLineHosts: []string{server.URL},
		Timeout:    5 * time.Second,
		Sleep:      func(time.Duration) {},
	})
}

func TestGetQuoteListRequestAndParse(t *testing.T) {
	var gotFields, gotFS, gotReferer string
	var gotPage, gotSize, gotOrder, gotFid string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotFields, gotFS = q.Get("fields"), q.Get("fs")
		gotPage, gotSize, gotOrder, gotFid = q.Get("pn"), q.Get("pz"), q.Get("po"), q.Get("fid")
		gotReferer = r.Header.Get("referer")
		_, _ = w.Write([]byte(`{"rc":0,"data":{"total":3210,"diff":[
		  {"f2":11.76,"f5":2189429,"f6":2593456789,"f12":"000001","f13":0,
		   "f14":"平安银行","f15":12.10,"f16":11.70,"f17":11.76,
		   "f20":230987654321,"f21":220000000000,"f26":"1991-04-03","f124":1763006400},
		  {"f2":"-","f5":"-","f12":"000002","f14":"停牌股","f20":"-"}
		]}}`))
	}))
	defer server.Close()

	page, err := newTestClient(t, server).GetQuoteList(context.Background(), GetQuoteListRequest{Page: 2, PageSize: 50})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if gotPage != "2" || gotSize != "50" || gotOrder != "0" || gotFid != SortByTotalMV {
		t.Errorf("分页/排序参数不符：pn=%s pz=%s po=%s fid=%s", gotPage, gotSize, gotOrder, gotFid)
	}
	if !strings.HasPrefix(gotFields, "f2,f5,f6,f12") {
		t.Errorf("fields 不符：%s", gotFields)
	}
	if gotFS != UniverseMainBoard {
		t.Errorf("fs = %q，期望 %q", gotFS, UniverseMainBoard)
	}
	if gotReferer != DefaultReferer {
		t.Errorf("referer = %q，期望 %q", gotReferer, DefaultReferer)
	}
	if page.Total != 3210 || page.Page != 2 || page.PageSize != 50 || page.SortField != SortByTotalMV {
		t.Fatalf("分页信息不符：%+v", page)
	}
	// 第二条无行情（f2 与 f20 都是 "-"），应被剔除。
	if len(page.Items) != 1 {
		t.Fatalf("条目数 = %d，期望 1", len(page.Items))
	}
	item := page.Items[0]
	if item.Code != "000001" || item.Name != "平安银行" || item.Last != 11.76 {
		t.Fatalf("基本字段不符：%+v", item)
	}
	if item.TotalMV != 230987654321 || item.FloatMV != 220000000000 {
		t.Errorf("市值解析不符：%+v", item)
	}
	if item.ListDate != "1991-04-03" || item.Market != 0 || item.Timestamp != 1763006400 {
		t.Errorf("附加字段不符：%+v", item)
	}
	// Raw 必须保留原始文本，便于排障。
	if item.Raw["f12"] != "000001" || item.Raw["f2"] != "11.76" {
		t.Errorf("Raw 未保留原值：%v", item.Raw)
	}
}

func TestGetQuoteListDescAndUniverse(t *testing.T) {
	var gotOrder, gotFid, gotFS string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotOrder, gotFid, gotFS = q.Get("po"), q.Get("fid"), q.Get("fs")
		_, _ = w.Write([]byte(`{"rc":0,"data":{"total":0,"diff":[]}}`))
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetQuoteList(context.Background(), GetQuoteListRequest{
		Desc:      true,
		SortField: SortByListDate,
		Universe:  UniverseAllA,
	})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if gotOrder != "1" || gotFid != SortByListDate {
		t.Errorf("排序参数不符：po=%s fid=%s", gotOrder, gotFid)
	}
	if gotFS != UniverseAllA {
		t.Errorf("fs = %q，期望 %q", gotFS, UniverseAllA)
	}
}

func TestGetQuoteListRetriesOnEmptyData(t *testing.T) {
	// 第一个镜像返回 data=null（限频），第二个镜像返回正常数据。
	var hit int64
	limited := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&hit, 1)
		_, _ = w.Write([]byte(`{"rc":0,"data":null}`))
	}))
	defer limited.Close()
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"rc":0,"data":{"total":1,"diff":[{"f2":1,"f12":"000001","f14":"甲","f20":100}]}}`))
	}))
	defer okServer.Close()

	client := New(Config{
		ListHosts: []string{limited.URL, okServer.URL},
		Timeout:   5 * time.Second,
		Sleep:     func(time.Duration) {},
	})
	page, err := client.GetQuoteList(context.Background(), GetQuoteListRequest{})
	if err != nil {
		t.Fatalf("应换镜像后成功：%v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("条目数 = %d，期望 1", len(page.Items))
	}
	if got := atomic.LoadInt64(&hit); got != 1 {
		t.Errorf("限频镜像应被请求 1 次，实际 %d", got)
	}
}

func TestGetQuoteListAllHostsFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetQuoteList(context.Background(), GetQuoteListRequest{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("应返回 *APIError：%v", err)
	}
	if apiErr.HTTPStatus != http.StatusBadGateway {
		t.Errorf("HTTPStatus = %d，期望 502", apiErr.HTTPStatus)
	}
	if apiErr.URL == "" {
		t.Error("错误信息应带最后一次请求地址")
	}
}

func TestGetKLineRequestAndParse(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"rc":0,"data":{"code":"000001","name":"平安银行","klines":[
		  "2004-12-31,6.580,6.590,6.660,6.580,17521,11568524.157",
		  "2026-11-13,11.760,12.070,12.100,11.700,2189429,2593456789.000"
		]}}`))
	}))
	defer server.Close()

	result, err := newTestClient(t, server).GetKLine(context.Background(), GetKLineRequest{
		Code:  "000001",
		Begin: "20041201",
		End:   "20261130",
		Limit: 1000,
	})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	for _, want := range []string{"secid=0.000001", "klt=101", "fqt=0", "beg=20041201", "end=20261130", "lmt=1000"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("查询串缺少 %s：%s", want, gotQuery)
		}
	}
	if result.Code != "000001" || result.Name != "平安银行" || len(result.Bars) != 2 {
		t.Fatalf("结果不符：%+v", result)
	}
	bar := result.Bars[1]
	if bar.Date != "2026-11-13" || bar.Open != 11.76 || bar.Close != 12.07 ||
		bar.High != 12.10 || bar.Low != 11.70 {
		t.Fatalf("OHLC 解析不符：%+v", bar)
	}
	// 成交量原始单位是手，不做换算。
	if bar.Volume != 2189429 || bar.Amount != 2593456789 {
		t.Errorf("量额解析不符：%+v", bar)
	}
}

func TestGetKLineSecIDInference(t *testing.T) {
	cases := map[string]string{
		"600000": "1.600000",
		"000001": "0.000001",
	}
	for code, want := range cases {
		got, err := resolveSecID(GetKLineRequest{Code: code})
		if err != nil || got != want {
			t.Errorf("resolveSecID(%s) = %q, %v；期望 %q", code, got, err, want)
		}
	}
	if _, err := resolveSecID(GetKLineRequest{}); err == nil {
		t.Fatal("缺少 SecID 与 Code 时应报错")
	}
	// SecID 优先。
	if got, _ := resolveSecID(GetKLineRequest{SecID: "1.000300", Code: "000001"}); got != "1.000300" {
		t.Errorf("SecID 应优先：%s", got)
	}
}

func TestGetTradingCalendar(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"rc":0,"data":{"code":"000001","name":"上证指数","klines":[
		  "2026-11-12,1,2,3,4,5,6","2026-11-13,1,2,3,4,5,6"]}}`))
	}))
	defer server.Close()

	dates, err := newTestClient(t, server).GetTradingCalendar(context.Background(), 260)
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if !strings.Contains(gotQuery, "secid=1.000001") || !strings.Contains(gotQuery, "lmt=260") {
		t.Errorf("询价参数不符：%s", gotQuery)
	}
	if len(dates) != 2 || dates[0] != "2026-11-12" || dates[1] != "2026-11-13" {
		t.Fatalf("日历不符：%v", dates)
	}
}

func TestGetKLineEmptyBars(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"rc":0,"data":{"code":"000001","name":"甲","klines":[]}}`))
	}))
	defer server.Close()

	result, err := newTestClient(t, server).GetKLine(context.Background(), GetKLineRequest{Code: "000001"})
	if err != nil {
		t.Fatalf("空 K 线不应报错：%v", err)
	}
	if len(result.Bars) != 0 {
		t.Fatalf("期望空切片：%+v", result.Bars)
	}
}

func TestRotateCyclesHosts(t *testing.T) {
	client := New(Config{ListHosts: []string{"a", "b", "c"}})
	got := []string{client.rotate(client.cfg.ListHosts), client.rotate(client.cfg.ListHosts),
		client.rotate(client.cfg.ListHosts), client.rotate(client.cfg.ListHosts)}
	want := []string{"a", "b", "c", "a"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("轮换结果不符：%v，期望 %v", got, want)
		}
	}
}

func TestConfigDefaults(t *testing.T) {
	got := Config{}.withDefaults()
	if len(got.ListHosts) != len(DefaultListHosts) || len(got.KLineHosts) != len(DefaultKLineHosts) {
		t.Fatalf("默认镜像未补齐：%+v", got)
	}
	if got.Referer != DefaultReferer {
		t.Fatalf("referer 默认值不符：%q", got.Referer)
	}
}

func TestParseNumTolerance(t *testing.T) {
	cases := map[string]float64{
		"11.76": 11.76, "-": 0, "": 0, "null": 0, "abc": 0, " 12.5 ": 12.5,
	}
	for text, want := range cases {
		if got := parseNum(text); got != want {
			t.Errorf("parseNum(%q) = %v，期望 %v", text, got, want)
		}
	}
}
