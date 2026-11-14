package finscope

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient 构造指向测试服务器的客户端，并记录最后一次请求的查询串与请求头。
type recorded struct {
	query       string
	headers     http.Header
	requestedAt time.Time
}

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *recorded) {
	t.Helper()
	rec := &recorded{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.query = r.URL.RawQuery
		rec.headers = r.Header.Clone()
		rec.requestedAt = time.Now()
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	return New(Config{BaseURL: server.URL}), rec
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("content-type", "application/json")
	_, _ = w.Write([]byte(body))
}

const constituentsSample = `{
  "QueryID": "12457052143672835826",
  "ResultCode": 0,
  "Result": {
    "list": {
      "headers": [
        {"canSort": 1, "key": "lastPx", "name": "最新价"},
        {"canSort": 1, "key": "pxChangeRate", "name": "涨跌幅"}
      ],
      "body": [
        {
          "code": "601939",
          "name": "建设银行",
          "exchange": "SH",
          "market": "ab",
          "financeType": "stock",
          "hasQuote": 1,
          "lastPx": "12.11",
          "pxChange": "+0.10",
          "pxChangeRate": "+0.83%",
          "volume": "128万手",
          "amount": "15.42亿",
          "turnoverRatio": "1.33%",
          "marketValue": "3.17万亿",
          "rawData": {
            "lastPx": 12.11,
            "pxChange": 0.1,
            "pxChangeRate": 0.83,
            "volume": 1276070.86,
            "amount": 1542319057,
            "turnoverRatio": 1.33,
            "marketValue": 3167980619468
          },
          "block": {"code": "480200", "name": "国有大型银行Ⅱ", "market": "ab", "financeType": "block"},
          "logo": {"type": "normal", "logo": "https://baidu-finance.cdn.bcebos.com/imgs/logo/stocks/stock_ab_601939.svg"},
          "pcUrl": "https://finance.baidu.com/stock/ab-601939"
        }
      ]
    },
    "pcUrl": "https://finance.baidu.com/index/ab-000300?mainTab=成分股"
  }
}`

// defaultRequest 返回一份合法的请求参数，供各用例复用。
func defaultRequest() GetConstituentsRequest {
	return GetConstituentsRequest{
		FinanceType: FinanceTypeIndex,
		Market:      MarketA,
		Code:        "000300",
		SortKey:     SortByMarketValue,
		SortType:    SortDesc,
		Limit:       500,
	}
}

func TestGetConstituentsParsesItems(t *testing.T) {
	client, rec := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, constituentsSample)
	})

	page, err := client.GetConstituents(context.Background(), defaultRequest())
	if err != nil {
		t.Fatalf("GetConstituents 返回错误：%v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("条目数 = %d，期望 1", len(page.Items))
	}
	item := page.Items[0]
	if item.Code != "601939" || item.Name != "建设银行" || item.Exchange != "SH" {
		t.Errorf("条目基础字段不符：%+v", item)
	}
	// 展示字符串与原始数值两套口径都要保留。
	if item.Amount != "15.42亿" {
		t.Errorf("成交额展示值 = %q，期望 15.42亿", item.Amount)
	}
	if item.RawData.Amount != 1542319057 {
		t.Errorf("成交额原始值 = %v，期望 1542319057", item.RawData.Amount)
	}
	if item.RawData.ChangeRatio != 0.83 {
		t.Errorf("涨跌幅原始值 = %v，期望 0.83", item.RawData.ChangeRatio)
	}
	if item.Block.Name != "国有大型银行Ⅱ" {
		t.Errorf("板块名 = %q", item.Block.Name)
	}
	if page.PCURL == "" {
		t.Error("pcUrl 未解析")
	}
	// 单页返回 1 条、Limit 为 500，说明没有更多数据。
	if page.HasMore {
		t.Error("条目数小于 Limit 时 HasMore 应为 false")
	}
	if len(page.Headers) != 2 || page.Headers[0].Key != "lastPx" {
		t.Errorf("列定义不符：%+v", page.Headers)
	}

	// 参数顺序与文档一致，且 pn 是偏移量而非页码。
	want := "financeType=index&market=ab&code=000300&sortKey=marketValue" +
		"&sortType=desc&style=tablelist&pn=0&rn=500&finClientType=pc"
	if rec.query != want {
		t.Errorf("查询串 = %q，期望 %q", rec.query, want)
	}
	if got := rec.headers.Get("referer"); got != DefaultReferer {
		t.Errorf("referer = %q，期望 %q", got, DefaultReferer)
	}
	if got := rec.headers.Get("content-type"); got != "" {
		t.Errorf("GET 请求不应带 content-type，实际 %q", got)
	}
}

func TestGetConstituentsRiskControl(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		// 外层 ResultCode 为 0，错误码在 Result 层。
		writeJSON(w, `{"QueryID":"7852813130838806383","ResultCode":0,
			"Result":{"code":403,"isCaptchaEnabled":true,"msg":"hit risk"}}`)
	})

	_, err := client.GetConstituents(context.Background(), defaultRequest())
	if err == nil {
		t.Fatal("风控响应应返回错误")
	}
	if !IsRiskControl(err) {
		t.Errorf("IsRiskControl = false，err = %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("错误类型 = %T，期望 *APIError", err)
	}
	if apiErr.Code != CodeRiskControl {
		t.Errorf("code = %d，期望 %d", apiErr.Code, CodeRiskControl)
	}
	if !strings.Contains(apiErr.Error(), "hit risk") {
		t.Errorf("错误信息未携带服务端描述：%s", apiErr.Error())
	}
}

func TestGetConstituentsEmptyPages(t *testing.T) {
	cases := map[string]string{
		"list 为空数组": `{"QueryID":"1","ResultCode":0,"Result":{"list":[]}}`,
		"list 为空对象": `{"QueryID":"1","ResultCode":0,"Result":{"list":{}}}`,
		"list 缺失":   `{"QueryID":"1","ResultCode":0,"Result":{"pcUrl":"https://x"}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, body)
			})
			page, err := client.GetConstituents(context.Background(), defaultRequest())
			if err != nil {
				t.Fatalf("空页不应返回错误：%v", err)
			}
			if len(page.Items) != 0 {
				t.Errorf("条目数 = %d，期望 0", len(page.Items))
			}
			if page.HasMore {
				t.Error("空页时 HasMore 应为 false")
			}
		})
	}
}

func TestGetConstituentsBadResponses(t *testing.T) {
	cases := map[string]string{
		"非 JSON":     `not json`,
		"缺少 Result":  `{"QueryID":"1","ResultCode":0}`,
		"Result 是数组": `{"QueryID":"1","ResultCode":0,"Result":[]}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, body)
			})
			if _, err := client.GetConstituents(context.Background(), defaultRequest()); err == nil {
				t.Fatal("异常响应应返回错误")
			}
		})
	}
}

func TestGetConstituentsHTTPError(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("too many requests"))
	})
	_, err := client.GetConstituents(context.Background(), defaultRequest())
	if err == nil {
		t.Fatal("HTTP 429 应返回错误")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("错误类型 = %T，期望 *APIError", err)
	}
	if apiErr.HTTPStatus != http.StatusTooManyRequests {
		t.Errorf("HTTPStatus = %d，期望 429", apiErr.HTTPStatus)
	}
	if !strings.Contains(apiErr.Body, "too many requests") {
		t.Errorf("未保留响应体：%q", apiErr.Body)
	}
	if IsRiskControl(err) {
		t.Error("HTTP 429 不应被判定为风控业务码")
	}
}

func TestGetConstituentsValidatesRequest(t *testing.T) {
	client := New(Config{})
	cases := map[string]func(*GetConstituentsRequest){
		"financeType 为空": func(r *GetConstituentsRequest) { r.FinanceType = "" },
		"financeType 非法": func(r *GetConstituentsRequest) { r.FinanceType = "plate" },
		"market 为空":      func(r *GetConstituentsRequest) { r.Market = "" },
		"market 非法":      func(r *GetConstituentsRequest) { r.Market = "jp" },
		"code 为空":        func(r *GetConstituentsRequest) { r.Code = "  " },
		"sortKey 非法":     func(r *GetConstituentsRequest) { r.SortKey = "price" },
		"sortType 非法":    func(r *GetConstituentsRequest) { r.SortType = "up" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			req := defaultRequest()
			mutate(&req)
			if _, err := client.GetConstituents(context.Background(), req); err == nil {
				t.Fatal("参数非法应返回错误")
			}
		})
	}
}

func TestGetConstituentsDefaults(t *testing.T) {
	req := defaultRequest()
	req.Style = ""
	req.ClientType = ""
	req.Limit = 0
	req.Offset = -5
	normalized, err := req.withDefaults()
	if err != nil {
		t.Fatalf("withDefaults 返回错误：%v", err)
	}
	if normalized.Style != DefaultStyle {
		t.Errorf("style = %q，期望 %q", normalized.Style, DefaultStyle)
	}
	if normalized.ClientType != DefaultClientType {
		t.Errorf("finClientType = %q，期望 %q", normalized.ClientType, DefaultClientType)
	}
	if normalized.Limit != DefaultPageSize {
		t.Errorf("limit = %d，期望 %d", normalized.Limit, DefaultPageSize)
	}
	if normalized.Offset != 0 {
		t.Errorf("负偏移量应归零，实际 %d", normalized.Offset)
	}
}

func TestGetAllConstituentsPaging(t *testing.T) {
	// 服务端共 5 条，每页 2 条：offset 0 → 2 条，2 → 2 条，4 → 1 条（末页）。
	var offsets []string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		offset := r.URL.Query().Get("pn")
		offsets = append(offsets, offset)
		var body string
		switch offset {
		case "0":
			body = `{"ResultCode":0,"Result":{"list":{"headers":[{"key":"lastPx","name":"最新价"}],
				"body":[{"code":"600000","name":"浦发银行"},{"code":"600036","name":"招商银行"}]}}}`
		case "2":
			body = `{"ResultCode":0,"Result":{"list":{"body":[
				{"code":"600036","name":"招商银行"},{"code":"601939","name":"建设银行"}]}}}`
		case "4":
			body = `{"ResultCode":0,"Result":{"list":{"body":[{"code":"601398","name":"工商银行"}]}}}`
		default:
			body = `{"ResultCode":0,"Result":{"list":[]}}`
		}
		writeJSON(w, body)
	})

	req := defaultRequest()
	req.Limit = 2
	result, err := client.GetAllConstituents(context.Background(), req)
	if err != nil {
		t.Fatalf("GetAllConstituents 返回错误：%v", err)
	}
	if result.Pages != 3 {
		t.Errorf("页数 = %d，期望 3", result.Pages)
	}
	// 600036 在第二页重复出现，应被去重。
	if len(result.Items) != 4 {
		t.Fatalf("条目数 = %d，期望 4：%+v", len(result.Items), result.Items)
	}
	codes := []string{"600000", "600036", "601939", "601398"}
	for i, want := range codes {
		if result.Items[i].Code != want {
			t.Errorf("第 %d 条 code = %q，期望 %q", i, result.Items[i].Code, want)
		}
	}
	if len(result.Headers) != 1 {
		t.Errorf("列定义应取首页，实际 %+v", result.Headers)
	}
	// 偏移量按 Limit 递进，而不是按页码自增。
	wantOffsets := []string{"0", "2", "4"}
	if strings.Join(offsets, ",") != strings.Join(wantOffsets, ",") {
		t.Errorf("请求偏移量 = %v，期望 %v", offsets, wantOffsets)
	}
}

func TestGetAllConstituentsStopsWithoutProgress(t *testing.T) {
	// 服务端忽略偏移量、每次都返回同一页：首页取到 2 条新条目后，第二页应因无新增而结束，
	// 不能一直翻下去。
	calls := 0
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		writeJSON(w, `{"ResultCode":0,"Result":{"list":{"body":[
			{"code":"600000"},{"code":"600036"}]}}}`)
	})
	req := defaultRequest()
	req.Limit = 2
	result, err := client.GetAllConstituents(context.Background(), req)
	if err != nil {
		t.Fatalf("GetAllConstituents 返回错误：%v", err)
	}
	if calls != 2 {
		t.Errorf("请求次数 = %d，期望 2：第二页无新增条目时应立即结束", calls)
	}
	if len(result.Items) != 2 {
		t.Errorf("条目数 = %d，期望 2", len(result.Items))
	}
}

func TestGetAllConstituentsStopsOnRiskControl(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, `{"ResultCode":0,"Result":{"code":403,"isCaptchaEnabled":true,"msg":"hit risk"}}`)
	})
	if _, err := client.GetAllConstituents(context.Background(), defaultRequest()); !IsRiskControl(err) {
		t.Fatalf("风控应中断翻页并返回风控错误，实际 %v", err)
	}
}

func TestGetAllConstituentsRespectsContext(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, constituentsSample)
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.GetAllConstituents(ctx, defaultRequest()); !errors.Is(err, context.Canceled) {
		t.Fatalf("ctx 取消应返回 context.Canceled，实际 %v", err)
	}
}
