package finscope

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// kLineSample 是文档中的真实响应节选（marketData 保留两条，均线列全为 --）。
const kLineSample = `{
  "QueryID": "1034999479576860943",
  "ResultNum": 0,
  "ResultCode": 0,
  "Result": {
    "newMarketData": {
      "headers": ["时间戳","时间","开盘","收盘","成交量","最高","最低","成交额","涨跌额","涨跌幅","换手率","昨收","ma5均价","ma5成交量"],
      "keys": ["timestamp","time","open","close","volume","high","low","amount","range","ratio","turnoverratio","preClose","ma5avgprice","ma5volume"],
      "marketData": "1794449220,2026-11-12 10:07,23.94,23.94,26300,23.95,23.92,629596.00,+0.01,+0.04,--,23.93,--,--;1794449280,2026-11-12 10:08,23.95,23.94,16300,23.96,23.94,390346.00,0.00,0.00,--,23.94,--,--;1794449340,2026-11-12 10:09,残缺记录"
    }
  }
}`

func TestGetQuoteKLineParsesBars(t *testing.T) {
	client, rec := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, kLineSample)
	})
	req := GetQuoteKLineRequest{Code: "600009", MarketType: MarketA, KType: KTypeMin1, Count: 241}
	result, err := client.GetQuoteKLine(context.Background(), req)
	if err != nil {
		t.Fatalf("GetQuoteKLine 返回错误：%v", err)
	}
	// 第三条是残缺记录，应跳过而不是报错。
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d，期望 1", result.Skipped)
	}
	if len(result.Bars) != 2 {
		t.Fatalf("K 线条数 = %d，期望 2", len(result.Bars))
	}
	first := result.Bars[0]
	if first.Timestamp != 1794449220 {
		t.Errorf("Timestamp = %d", first.Timestamp)
	}
	if first.Time != "2026-11-12 10:07" {
		t.Errorf("Time = %q", first.Time)
	}
	if first.Open != 23.94 || first.Close != 23.94 || first.High != 23.95 || first.Low != 23.92 {
		t.Errorf("OHLC 解析错误：%+v", first)
	}
	if first.Volume != 26300 || first.Amount != 629596 {
		t.Errorf("量额解析错误：%+v", first)
	}
	// 涨跌额带 + 号，涨跌幅是百分数。
	if first.ChangePrice != 0.01 || first.ChangeRatio != 0.04 {
		t.Errorf("涨跌解析错误：%+v", first)
	}
	// -- 应转成 nil，区分「无值」与 0。
	if first.TurnoverRatio != nil {
		t.Errorf("换手率为 -- 时应为 nil，实际 %v", *first.TurnoverRatio)
	}
	if first.LastClose == nil || *first.LastClose != 23.93 {
		t.Errorf("昨收解析错误：%v", first.LastClose)
	}
	// 均线列未识别，应保留在 Extra。
	if first.Extra["ma5avgprice"] != "--" {
		t.Errorf("Extra 未保留未知列：%+v", first.Extra)
	}

	want := "srcid=5353&pointType=string&group=quotation_kline_ab&newFormat=1" +
		"&finClientType=pc&code=600009&query=600009&market_type=ab&is_kc=0&ktype=min1&count=241"
	if rec.query != want {
		t.Errorf("查询串 = %q，期望 %q", rec.query, want)
	}
}

func TestGetQuoteKLineEmptyMarketData(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, `{"ResultCode":0,"Result":{"newMarketData":{"keys":["timestamp","time"],"marketData":""}}}`)
	})
	result, err := client.GetQuoteKLine(context.Background(), GetQuoteKLineRequest{Code: "899050"})
	if err != nil {
		t.Fatalf("空行情不应返回错误：%v", err)
	}
	if len(result.Bars) != 0 {
		t.Errorf("K 线条数 = %d，期望 0", len(result.Bars))
	}
}

func TestGetQuoteKLineMissingKeys(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, `{"ResultCode":0,"Result":{"newMarketData":{"marketData":"1,2,3"}}}`)
	})
	if _, err := client.GetQuoteKLine(context.Background(), GetQuoteKLineRequest{Code: "899050"}); err == nil {
		t.Fatal("缺少 keys 时应返回错误")
	}
}

func TestGetQuoteKLineRiskControl(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		// 北证50 与个股在真实环境中都返回这一结构。
		writeJSON(w, `{"QueryID":"1964182802104643411","Result":{"code":403,"isCaptchaEnabled":true,"msg":"hit risk"},"ResultCode":0}`)
	})
	_, err := client.GetQuoteKLine(context.Background(), GetQuoteKLineRequest{Code: "899050"})
	if !IsRiskControl(err) {
		t.Fatalf("应识别为风控，实际 %v", err)
	}
}

func TestGetQuoteKLineValidatesRequest(t *testing.T) {
	client := New(Config{})
	cases := map[string]GetQuoteKLineRequest{
		"code 为空":        {Code: " "},
		"market_type 非法": {Code: "899050", MarketType: "jp"},
		"ktype 非法":       {Code: "899050", KType: "week"},
		"is_kc 非法":       {Code: "899050", IsKC: 2},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := client.GetQuoteKLine(context.Background(), req); err == nil {
				t.Fatal("参数非法应返回错误")
			}
		})
	}
}

func TestGetQuoteKLineDefaults(t *testing.T) {
	normalized, err := GetQuoteKLineRequest{Code: " 899050 "}.withDefaults()
	if err != nil {
		t.Fatalf("withDefaults 返回错误：%v", err)
	}
	if normalized.MarketType != MarketA {
		t.Errorf("market_type 默认值 = %q，期望 %q", normalized.MarketType, MarketA)
	}
	if normalized.KType != KTypeMin1 {
		t.Errorf("ktype 默认值 = %q，期望 %q", normalized.KType, KTypeMin1)
	}
	if normalized.Count != DefaultKLineCount {
		t.Errorf("count 默认值 = %d，期望 %d", normalized.Count, DefaultKLineCount)
	}
	if normalized.Code != "899050" {
		t.Errorf("code 未去空白：%q", normalized.Code)
	}
	// 可选的 name 为空时不应出现在查询串里。
	if params := normalized.params(); strings.Contains(encodeQuery(params), "name=") {
		t.Errorf("name 为空时不应带上该参数：%s", encodeQuery(params))
	}
}
