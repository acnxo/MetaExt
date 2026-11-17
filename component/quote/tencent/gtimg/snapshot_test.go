package gtimg

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	return New(Config{
		BaseURL:      server.URL,
		KLineBaseURL: server.URL,
		Timeout:      5 * time.Second,
		Sleep:        func(time.Duration) {},
	})
}

// writeGBK 把文本按 GBK 编码写入响应，模拟站点的真实编码。
func writeGBK(t *testing.T, w http.ResponseWriter, text string) {
	t.Helper()
	encoded, err := simplifiedchinese.GBK.NewEncoder().String(text)
	if err != nil {
		t.Fatalf("GBK 编码失败：%v", err)
	}
	_, _ = w.Write([]byte(encoded))
}

// snapshotLine 拼一行快照响应。
func snapshotLine(symbol, name, code, last, open, volume, ts, amount, floatMV, totalMV string) string {
	fields := make([]string, minSnapshotSeg)
	fields[idxName] = name
	fields[idxCode] = code
	fields[idxLast] = last
	fields[idxOpen] = open
	fields[idxVolume] = volume
	fields[idxTimestamp] = ts
	fields[idxAmount] = amount
	fields[idxFloatMV] = floatMV
	fields[idxTotalMV] = totalMV
	return `v_` + symbol + `="` + strings.Join(fields, "~") + `";`
}

func TestGetQuoteSnapshotParse(t *testing.T) {
	var gotPath, gotReferer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotReferer = r.Header.Get("referer")
		body := snapshotLine("sz000001", "平安银行", "000001", "12.07", "11.76", "2189429",
			"20261113150000", "259345.67", "2309.88", "2309.88") +
			snapshotLine("sh600000", "浦发银行", "600000", "10.11", "10.05", "1234567",
				"20261113150000", "120000.00", "3000.00", "3000.00")
		writeGBK(t, w, body)
	}))
	defer server.Close()

	snapshot, err := newTestClient(t, server).GetQuoteSnapshot(context.Background(), GetQuoteSnapshotRequest{
		Codes: []string{"000001", " 600000 ", "000001", ""},
	})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if gotPath != "/q=sz000001,sh600000" {
		t.Errorf("请求路径 = %q，期望 q=sz000001,sh600000（去重后）", gotPath)
	}
	if gotReferer != DefaultReferer {
		t.Errorf("referer = %q，期望 %q", gotReferer, DefaultReferer)
	}
	if len(snapshot.Items) != 2 {
		t.Fatalf("条目数 = %d，期望 2", len(snapshot.Items))
	}
	item := snapshot.Items[0]
	if item.Code != "000001" || item.Name != "平安银行" || item.Last != 12.07 || item.Open != 11.76 {
		t.Fatalf("基本字段不符：%+v", item)
	}
	// 成交额由万元换算成元，市值由亿元换算成元。
	if item.Amount != 2593456700 {
		t.Errorf("成交额 = %v，期望 2593456700", item.Amount)
	}
	if item.TotalMV != 230988000000 || item.FloatMV != 230988000000 {
		t.Errorf("市值换算不符：%+v", item)
	}
	if item.Volume != 2189429 {
		t.Errorf("成交量应保留「手」：%v", item.Volume)
	}
	if item.Timestamp != "20261113150000" {
		t.Errorf("时间戳不符：%q", item.Timestamp)
	}
}

func TestGetQuoteSnapshotSkipsDeadCodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		body := snapshotLine("sz000002", "停牌股", "000002", "0", "0", "0", "20261113150000", "0", "0", "0") +
			snapshotLine("sz000003", "正常股", "000003", "5.00", "4.90", "100", "20261113150000", "10", "1", "1")
		writeGBK(t, w, body)
	}))
	defer server.Close()

	snapshot, err := newTestClient(t, server).GetQuoteSnapshot(context.Background(), GetQuoteSnapshotRequest{
		Codes: []string{"000002", "000003"},
	})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if len(snapshot.Items) != 1 || snapshot.Items[0].Code != "000003" {
		t.Fatalf("应只保留有行情的记录：%+v", snapshot.Items)
	}
}

func TestGetQuoteSnapshotBinarySplitOn414(t *testing.T) {
	// 超过 2 只即返回 414，客户端应二分到 1 只后成功。
	var calls int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&calls, 1)
		symbols := strings.TrimPrefix(r.URL.Path, "/q=")
		if len(strings.Split(symbols, ",")) > 1 {
			w.WriteHeader(http.StatusRequestURITooLong)
			return
		}
		code := strings.TrimPrefix(strings.TrimPrefix(symbols, "sz"), "sh")
		writeGBK(t, w, snapshotLine(symbols, "股"+code, code, "1", "1", "1",
			"20261113150000", "1", "1", "1"))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:       server.URL,
		KLineBaseURL:  server.URL,
		Timeout:       5 * time.Second,
		Sleep:         func(time.Duration) {},
		MinSplitCodes: 1,
	})
	snapshot, err := client.GetQuoteSnapshot(context.Background(), GetQuoteSnapshotRequest{
		Codes: []string{"000001", "000002", "000003", "000004"},
	})
	if err != nil {
		t.Fatalf("二分后应成功：%v", err)
	}
	if len(snapshot.Items) != 4 {
		t.Fatalf("条目数 = %d，期望 4", len(snapshot.Items))
	}
	if got := atomic.LoadInt64(&calls); got < 4 {
		t.Errorf("应发生多次拆分请求，实际 %d 次", got)
	}
}

func TestGetQuoteSnapshotEmptyCodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("空代码不应发起请求")
	}))
	defer server.Close()

	if _, err := newTestClient(t, server).GetQuoteSnapshot(context.Background(), GetQuoteSnapshotRequest{}); err == nil {
		t.Fatal("空代码应报错")
	}
}

func TestGetQuoteSnapshotHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetQuoteSnapshot(context.Background(), GetQuoteSnapshotRequest{
		Codes: []string{"000001"},
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusServiceUnavailable {
		t.Fatalf("应返回 HTTPStatus=503 的错误：%v", err)
	}
}

func TestGetKLineRequestAndParse(t *testing.T) {
	var gotParam, gotReferer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotParam = r.URL.Query().Get("param")
		gotReferer = r.Header.Get("referer")
		_, _ = w.Write([]byte(`{"code":0,"msg":"","data":{"sz000001":{"qfqday":[
		  ["2026-11-12","11.70","11.99","12.00","11.65","1987654"],
		  ["2026-11-13","11.76","12.07","12.10","11.70","2189429"]
		],"day":[["2026-11-13","11.76","12.07","12.10","11.70","2189429"]]}}}`))
	}))
	defer server.Close()

	result, err := newTestClient(t, server).GetKLine(context.Background(), GetKLineRequest{
		Code:   "000001",
		Begin:  "2026-11-01",
		End:    "2026-11-30",
		Count:  100,
		Adjust: true,
	})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if gotParam != "sz000001,day,2026-11-01,2026-11-30,100,qfq" {
		t.Errorf("param = %q", gotParam)
	}
	if gotReferer != DefaultReferer {
		t.Errorf("referer = %q", gotReferer)
	}
	if result.Symbol != "sz000001" || len(result.Bars) != 2 {
		t.Fatalf("结果不符：%+v", result)
	}
	bar := result.Bars[1]
	if bar.Date != "2026-11-13" || bar.Open != 11.76 || bar.Close != 12.07 ||
		bar.High != 12.10 || bar.Low != 11.70 || bar.Volume != 2189429 {
		t.Fatalf("日线解析不符：%+v", bar)
	}
}

func TestGetKLineUnadjustedUsesDay(t *testing.T) {
	var gotParam string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotParam = r.URL.Query().Get("param")
		_, _ = w.Write([]byte(`{"code":0,"data":{"sh600000":{"day":[["2026-11-13","10.05","10.11","10.20","10.00","1234567"]]}}}`))
	}))
	defer server.Close()

	result, err := newTestClient(t, server).GetKLine(context.Background(), GetKLineRequest{Code: "600000"})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if !strings.HasSuffix(gotParam, ",1000,") {
		t.Errorf("未复权时最后一段应为空：%q", gotParam)
	}
	if len(result.Bars) != 1 || result.Bars[0].Close != 10.11 {
		t.Fatalf("未复权解析不符：%+v", result.Bars)
	}
}

func TestGetKLineUnknownSymbol(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"data":{}}`))
	}))
	defer server.Close()

	result, err := newTestClient(t, server).GetKLine(context.Background(), GetKLineRequest{Code: "000001"})
	if err != nil {
		t.Fatalf("无数据不应报错：%v", err)
	}
	if len(result.Bars) != 0 {
		t.Fatalf("期望空 Bars：%+v", result.Bars)
	}
}

func TestSplitCodes(t *testing.T) {
	codes := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		codes = append(codes, "00000"+strconv.Itoa(i))
	}
	chunks := splitCodes(codes, 2)
	if len(chunks) != 3 || len(chunks[0]) != 2 || len(chunks[2]) != 1 {
		t.Fatalf("切分不符：%v", chunks)
	}
	if got := splitCodes(codes, 10); len(got) != 1 || len(got[0]) != 5 {
		t.Fatalf("不足一批时应单批返回：%v", got)
	}
}

func TestSymbolOf(t *testing.T) {
	cases := map[string]string{
		"600000": "sh600000", "000001": "sz000001", "300750": "sz300750",
		// 北交所（4/8 开头）必须用 bj 前缀。
		"830799": "bj830799", "430047": "bj430047",
	}
	for code, want := range cases {
		if got := symbolOf(code); got != want {
			t.Errorf("symbolOf(%s) = %s，期望 %s", code, got, want)
		}
	}
}

func TestConfigDefaults(t *testing.T) {
	got := Config{}.withDefaults()
	if got.BaseURL != DefaultBaseURL || got.KLineBaseURL != DefaultKLineBaseURL ||
		got.Referer != DefaultReferer || got.Batch != DefaultBatch || got.Workers != DefaultWorkers {
		t.Fatalf("默认值未补齐：%+v", got)
	}
}
