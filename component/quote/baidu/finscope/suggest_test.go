package finscope

import (
	"context"
	"net/http"
	"testing"
)

// suggestSample 是 /vapi/v1/sug?wd=专精特新 的真实响应节选：同名的板块、两个指数与 ETF 并列。
const suggestSample = `{"QueryID":"4861967583676525358","Result":{"list":[
 {"code":"031425","name":"专精特新","market":"ab","type":"block","price":"1024.24","ratio":"-2.73%","status":"-1","subType":"--","exchange":"BK"},
 {"code":"399267","name":"专精特新","market":"ab","type":"index","price":"1931.19","ratio":"-5.06%","status":"-1","subType":"--","exchange":"SZ"},
 {"code":"899601","name":"专精特新","market":"ab","type":"index","price":"1565.73","ratio":"-3.35%","status":"-1","subType":"--","exchange":"NQ"},
 {"code":"563210","name":"专精特新ETF富国","market":"ab","type":"etf","price":"1.495","ratio":"-3.55%","status":"-1","subType":"--","exchange":"SH"}
]},"ResultCode":0}`

func TestSuggestParsesItems(t *testing.T) {
	client, rec := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, suggestSample)
	})
	items, err := client.Suggest(context.Background(), "专精特新")
	if err != nil {
		t.Fatalf("Suggest 返回错误：%v", err)
	}
	if len(items) != 4 {
		t.Fatalf("条目数 = %d，期望 4", len(items))
	}
	if rec.query != "wd=%E4%B8%93%E7%B2%BE%E7%89%B9%E6%96%B0" {
		t.Errorf("查询串 = %q", rec.query)
	}
	index := items[1]
	if index.Code != "399267" || index.Type != ItemTypeIndex || index.Exchange != "SZ" {
		t.Errorf("指数条目不符：%+v", index)
	}
	if price, ok := index.PriceValue(); !ok || price != 1931.19 {
		t.Errorf("PriceValue = %v, %v，期望 1931.19", price, ok)
	}
	if ratio, ok := index.RatioPercent(); !ok || ratio != -5.06 {
		t.Errorf("RatioPercent = %v, %v，期望 -5.06", ratio, ok)
	}
}

func TestFindSuggestFiltersByType(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, suggestSample)
	})
	items, err := client.FindSuggest(context.Background(), "专精特新", ItemTypeIndex)
	if err != nil {
		t.Fatalf("FindSuggest 返回错误：%v", err)
	}
	// 同名条目里有 1 个板块、2 个指数、1 个 ETF，过滤后应只剩指数。
	if len(items) != 2 {
		t.Fatalf("指数条目数 = %d，期望 2", len(items))
	}
	codes := []string{items[0].Code, items[1].Code}
	if codes[0] != "399267" || codes[1] != "899601" {
		t.Errorf("指数代码 = %v，期望 [399267 899601]", codes)
	}
}

func TestSuggestNullResult(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, `{"ResultCode":0,"ResultNum":0,"QueryID":"1","Result":null}`)
	})
	items, err := client.Suggest(context.Background(), "不存在的名字")
	if err != nil {
		t.Fatalf("无匹配不应返回错误：%v", err)
	}
	if len(items) != 0 {
		t.Errorf("条目数 = %d，期望 0", len(items))
	}
}

func TestSuggestValidatesKeyword(t *testing.T) {
	client := New(Config{})
	if _, err := client.Suggest(context.Background(), "  "); err == nil {
		t.Fatal("空关键词应返回错误")
	}
}

func TestSuggestBadResponse(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, `not json`)
	})
	if _, err := client.Suggest(context.Background(), "专精特新"); err == nil {
		t.Fatal("非 JSON 响应应返回错误")
	}
}
