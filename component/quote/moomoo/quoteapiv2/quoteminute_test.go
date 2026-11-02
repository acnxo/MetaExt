package quoteapiv2

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// quoteMinuteSample 分钟级分时响应样本：时间戳、价格、量额为数值，ratio 为带百分号的字符串。
const quoteMinuteSample = `{
    "code": 0,
    "message": "成功",
    "data": {
        "stockId": 70000294,
        "server_time": 1780507278500,
        "last_close_price": 45.5,
        "highest_price": 46.8,
        "lowest_price": 45.2,
        "list": [
            {
                "time": 1780480800,
                "price": 45.6,
                "cc_price": 45.6,
                "volume": 12500,
                "turnover": 570000.0,
                "ratio": "0.22%",
                "change_price": 0.1
            }
        ],
        "time_section": [
            {"begin": 1780480800, "end": 1780491600}
        ]
    }
}`

// quoteMinuteRequest 返回一组完整的定位参数。
func quoteMinuteRequest() GetQuoteMinuteRequest {
	return GetQuoteMinuteRequest{
		StockID:           "70000294",
		MarketType:        "2",
		Type:              "2",
		MarketCode:        "70",
		InstrumentType:    "9",
		SubInstrumentType: "9008",
	}
}

// TestGetQuoteMinuteRequestAndParse 校验路径、参数键序、签名与响应解析。
func TestGetQuoteMinuteRequestAndParse(t *testing.T) {
	var (
		gotOrder   []string
		gotQuery   map[string]string
		gotHeaders http.Header
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != DefaultQuoteMinutePath {
			t.Errorf("路径期望 %s，实际 %s", DefaultQuoteMinutePath, r.URL.Path)
		}
		gotHeaders = r.Header.Clone()
		gotOrder, gotQuery = splitQuery(r.URL.RawQuery)
		_, _ = w.Write([]byte(quoteMinuteSample))
	}))
	defer server.Close()

	req := quoteMinuteRequest()
	req.TimestampMillis = "1780507278122"

	client := newTestClient(t, server, nil)
	minute, err := client.GetQuoteMinute(context.Background(), req)
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}

	wantOrder := []string{"stockId", "marketType", "type", "marketCode", "instrumentType", "subInstrumentType", "_"}
	if strings.Join(gotOrder, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("查询键序不符：期望 %v，实际 %v", wantOrder, gotOrder)
	}
	if _, ok := gotQuery["req_section"]; ok {
		t.Fatal("req_section 为空时应整键省略")
	}
	// 与 K 线接口参数集合相同，因此签名向量一致。
	if got := gotHeaders.Get("quote-token"); got != "45671b44c1" {
		t.Fatalf("quote-token 期望 45671b44c1，实际 %s", got)
	}

	if minute.StockID != 70000294 || minute.ServerTime != 1780507278500 {
		t.Fatalf("概览字段不符：%+v", minute)
	}
	if minute.LastClosePrice != 45.5 || minute.HighestPrice != 46.8 || minute.LowestPrice != 45.2 {
		t.Fatalf("价格字段不符：%+v", minute)
	}
	if len(minute.Ticks) != 1 {
		t.Fatalf("分时条数期望 1，实际 %d", len(minute.Ticks))
	}
	tick := minute.Ticks[0]
	if tick.Time != 1780480800 || tick.Price != 45.6 || tick.CCPrice != 45.6 ||
		tick.Volume != 12500 || tick.Turnover != 570000 || tick.ChangePrice != 0.1 {
		t.Fatalf("分时字段不符：%+v", tick)
	}
	if tick.Ratio != "0.22%" {
		t.Fatalf("涨跌幅应保留原始字符串，实际 %q", tick.Ratio)
	}
	if len(minute.Sections) != 1 {
		t.Fatalf("交易时段数期望 1，实际 %d", len(minute.Sections))
	}
	if minute.Sections[0].Begin != 1780480800 || minute.Sections[0].End != 1780491600 {
		t.Fatalf("交易时段不符：%+v", minute.Sections[0])
	}
}

// TestGetQuoteMinuteReqSection req_section 必须插在 subInstrumentType 之后、时间戳之前。
func TestGetQuoteMinuteReqSection(t *testing.T) {
	var gotOrder []string
	var gotToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("quote-token")
		gotOrder, _ = splitQuery(r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"code":0,"data":{"list":[],"time_section":[]}}`))
	}))
	defer server.Close()

	req := quoteMinuteRequest()
	req.ReqSection = "1"
	req.TimestampMillis = "1780507278122"

	client := newTestClient(t, server, nil)
	if _, err := client.GetQuoteMinute(context.Background(), req); err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
	want := []string{"stockId", "marketType", "type", "marketCode", "instrumentType", "subInstrumentType", "req_section", "_"}
	if strings.Join(gotOrder, ",") != strings.Join(want, ",") {
		t.Fatalf("查询键序不符：期望 %v，实际 %v", want, gotOrder)
	}
	if gotToken != "53de4453c4" {
		t.Fatalf("quote-token 期望 53de4453c4，实际 %s", gotToken)
	}
}

// TestGetQuoteMinuteMissingRequired 六个定位参数都必填，缺失时本地报错且不发请求。
func TestGetQuoteMinuteMissingRequired(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	cases := map[string]GetQuoteMinuteRequest{
		"stockId":           {MarketType: "2", Type: "2", MarketCode: "70", InstrumentType: "9", SubInstrumentType: "9008"},
		"marketType":        {StockID: "70000294", Type: "2", MarketCode: "70", InstrumentType: "9", SubInstrumentType: "9008"},
		"type":              {StockID: "70000294", MarketType: "2", MarketCode: "70", InstrumentType: "9", SubInstrumentType: "9008"},
		"marketCode":        {StockID: "70000294", MarketType: "2", Type: "2", InstrumentType: "9", SubInstrumentType: "9008"},
		"instrumentType":    {StockID: "70000294", MarketType: "2", Type: "2", MarketCode: "70", SubInstrumentType: "9008"},
		"subInstrumentType": {StockID: "70000294", MarketType: "2", Type: "2", MarketCode: "70", InstrumentType: "9"},
	}
	for field, req := range cases {
		_, err := client.GetQuoteMinute(context.Background(), req)
		if err == nil || !strings.Contains(err.Error(), field) {
			t.Errorf("缺少 %s 应报错并指出字段，实际 %v", field, err)
		}
	}
	if called {
		t.Fatal("缺少必填参数时不应发起请求")
	}
}

// TestGetQuoteMinuteParamsError 定位参数与 stockId 不匹配时服务端返回 500。
func TestGetQuoteMinuteParamsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":500,"message":"Params Error","data":null}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetQuoteMinute(context.Background(), quoteMinuteRequest())
	if err == nil || !IsParamsError(err) {
		t.Fatalf("应识别为参数错误（500），实际 %v", err)
	}
}

// TestGetQuoteMinuteParseError 响应不是合法 JSON 时应给出解析失败错误。
func TestGetQuoteMinuteParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>not json</html>"))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetQuoteMinute(context.Background(), quoteMinuteRequest())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !strings.Contains(apiErr.Message, "响应解析失败") {
		t.Fatalf("应返回解析失败错误，实际 %v", err)
	}
}

// TestGetQuoteMinuteHTTPError 非 200 应返回带状态码的错误。
func TestGetQuoteMinuteHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetQuoteMinute(context.Background(), quoteMinuteRequest())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusBadGateway {
		t.Fatalf("应返回 HTTPStatus=502 的错误，实际 %v", err)
	}
}
