package quoteapiv2

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newsListSample 新闻列表响应样本：server_time、time 为数值，has_more 为布尔，
// 其余字段为字符串。
const newsListSample = `{
    "code": 0,
    "message": "成功",
    "data": {
        "server_time": 1781592387840,
        "has_more": true,
        "seq_mark": "1_1781537154",
        "list": [
            {
                "id": "news-1",
                "title": "小米集团发布新品",
                "time": 1781537154,
                "source": "新浪财经",
                "url": "https://example.com/news/1",
                "summary": "摘要内容"
            }
        ]
    }
}`

// TestGetNewsListRequestAndParse 校验路径、参数键序（含 snake_case 与 camelCase 混用）、
// 签名与响应解析。
func TestGetNewsListRequestAndParse(t *testing.T) {
	var (
		gotOrder   []string
		gotQuery   map[string]string
		gotHeaders http.Header
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != DefaultNewsListPath {
			t.Errorf("路径期望 %s，实际 %s", DefaultNewsListPath, r.URL.Path)
		}
		gotHeaders = r.Header.Clone()
		gotOrder, gotQuery = splitQuery(r.URL.RawQuery)
		_, _ = w.Write([]byte(newsListSample))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	page, err := client.GetNewsList(context.Background(), GetNewsListRequest{
		StockID:         "76033806042898",
		MarketType:      "1",
		Type:            "0",
		SubType:         "0",
		TimestampMillis: "1781592384789",
	})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}

	wantOrder := []string{"stock_id", "market_type", "type", "subType", "_"}
	if strings.Join(gotOrder, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("查询键序不符：期望 %v，实际 %v", wantOrder, gotOrder)
	}
	if gotQuery["stock_id"] != "76033806042898" || gotQuery["market_type"] != "1" ||
		gotQuery["type"] != "0" || gotQuery["subType"] != "0" {
		t.Fatalf("参数值不符：%v", gotQuery)
	}
	if gotQuery["_"] != "1781592384789" {
		t.Fatalf("时间戳未透传：%s", gotQuery["_"])
	}
	if got := gotHeaders.Get("quote-token"); got != "055b0960ec" {
		t.Fatalf("quote-token 期望 055b0960ec，实际 %s", got)
	}

	if page.ServerTime != 1781592387840 || !page.HasMore || page.SeqMark != "1_1781537154" {
		t.Fatalf("外层字段不符：%+v", page)
	}
	if len(page.Items) != 1 {
		t.Fatalf("条目数期望 1，实际 %d", len(page.Items))
	}
	item := page.Items[0]
	if item.ID != "news-1" || item.Title != "小米集团发布新品" || item.Time != 1781537154 ||
		item.Source != "新浪财经" || item.URL != "https://example.com/news/1" ||
		item.Summary != "摘要内容" {
		t.Fatalf("条目字段不符：%+v", item)
	}
}

// TestGetNewsListDefaults 未传可选参数时应补默认值；时间戳为空则整键省略。
func TestGetNewsListDefaults(t *testing.T) {
	var gotOrder []string
	var gotQuery map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrder, gotQuery = splitQuery(r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"code":0,"data":{"has_more":false,"list":[]}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	page, err := client.GetNewsList(context.Background(), GetNewsListRequest{StockID: "76033806042898"})
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
	if page.HasMore || len(page.Items) != 0 {
		t.Fatalf("空列表解析不符：%+v", page)
	}
	want := []string{"stock_id", "market_type", "type", "subType"}
	if strings.Join(gotOrder, ",") != strings.Join(want, ",") {
		t.Fatalf("查询键序不符：期望 %v，实际 %v", want, gotOrder)
	}
	if gotQuery["market_type"] != DefaultNewsMarketType ||
		gotQuery["type"] != DefaultNewsType || gotQuery["subType"] != DefaultNewsSubType {
		t.Fatalf("默认值不符：%v", gotQuery)
	}
	if _, ok := gotQuery["_"]; ok {
		t.Fatal("时间戳为空时应整键省略")
	}
}

// TestGetNewsListParamsMissing 服务端以 404 表示必填参数缺失。
func TestGetNewsListParamsMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":404,"message":"参数缺失","data":null}`))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetNewsList(context.Background(), GetNewsListRequest{StockID: "76033806042898"})
	if err == nil || !IsParamsMissing(err) {
		t.Fatalf("应识别为参数缺失（404），实际 %v", err)
	}
	if IsParamsError(err) || IsTokenRejected(err) {
		t.Fatal("404 不应被判定为其它错误类型")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Path != DefaultNewsListPath {
		t.Fatalf("错误应带接口路径，实际 %v", err)
	}
}

// TestGetNewsListEmptyStockID stock_id 为空时本地直接报错，不发请求。
func TestGetNewsListEmptyStockID(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	if _, err := client.GetNewsList(context.Background(), GetNewsListRequest{StockID: "  "}); err == nil {
		t.Fatal("stock_id 为空应报错")
	}
	if called {
		t.Fatal("stock_id 为空时不应发起请求")
	}
}

// TestGetNewsListParseError 响应不是合法 JSON 时应给出解析失败错误。
func TestGetNewsListParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetNewsList(context.Background(), GetNewsListRequest{StockID: "1"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !strings.Contains(apiErr.Message, "响应解析失败") {
		t.Fatalf("应返回解析失败错误，实际 %v", err)
	}
}

// TestGetNewsListHTTPError 非 200 应返回带状态码的错误。
func TestGetNewsListHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	}))
	defer server.Close()

	client := newTestClient(t, server, nil)
	_, err := client.GetNewsList(context.Background(), GetNewsListRequest{StockID: "1"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusBadGateway {
		t.Fatalf("应返回 HTTPStatus=502 的错误，实际 %v", err)
	}
}
