package finscope

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// fiveDaySample 是文档中的实测响应节选：marketData 只保留首日、每日只保留两条记录。
const fiveDaySample = `{
  "QueryID": "15325184726536934777",
  "ResultCode": 0,
  "ResultNum": 0,
  "Result": {
    "StdStg": "hundsun_fivedays",
    "code": "399001",
    "key": "399001",
    "update": {"timezone": "北京时间", "text": "2026-11-13 15:00:00", "time": 1794553200},
    "newMarketData": {
      "headers": ["时间戳","时间","价格","涨跌额","涨跌幅","成交量","成交额"],
      "keys": ["timestamp","time","price","range","ratio","volume","amount"],
      "marketData": [
        {"date": "2026-11-09",
         "p": "1794187860,11-09 09:31,11256.33,+80.55,+0.72,2784792751.00,41449256787.00;1794187920,11-09 09:32,11245.94,+70.16,+0.63,1252108048.00,18677359165.00;1794187980,11-10 09:33,残缺"}
      ],
      "maxPoints": "1201",
      "cx": ["11-09"]
    }
  }
}`

func TestGetQuoteFiveDayParsesPoints(t *testing.T) {
	client, rec := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, fiveDaySample)
	})
	result, err := client.GetQuoteFiveDay(context.Background(), GetQuoteFiveDayRequest{Code: "399001"})
	if err != nil {
		t.Fatalf("GetQuoteFiveDay 返回错误：%v", err)
	}
	if len(result.Days) != 1 {
		t.Fatalf("交易日数 = %d，期望 1", len(result.Days))
	}
	day := result.Days[0]
	if day.Date != "2026-11-09" {
		t.Errorf("date = %q", day.Date)
	}
	// 第三条记录字段不足，应跳过并计数。
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d，期望 1", result.Skipped)
	}
	if len(day.Points) != 2 {
		t.Fatalf("分钟点数 = %d，期望 2", len(day.Points))
	}
	first := day.Points[0]
	if first.Timestamp != 1794187860 {
		t.Errorf("timestamp = %d", first.Timestamp)
	}
	// time 不含年份，年份要从 date 取。
	if first.Time != "11-09 09:31" {
		t.Errorf("time = %q，期望 11-09 09:31", first.Time)
	}
	if first.Price != 11256.33 || first.ChangePrice != 80.55 || first.ChangeRatio != 0.72 {
		t.Errorf("价位解析错误：%+v", first)
	}
	if first.Volume != 2784792751 || first.Amount != 41449256787 {
		t.Errorf("量额解析错误：%+v", first)
	}
	if result.MaxPoints != "1201" || len(result.CX) != 1 {
		t.Errorf("元信息不符：maxPoints=%q cx=%v", result.MaxPoints, result.CX)
	}

	// 该接口没有 ktype / count 参数。
	if strings.Contains(rec.query, "ktype=") || strings.Contains(rec.query, "count=") {
		t.Errorf("五日分时不应带 ktype / count：%s", rec.query)
	}
	want := "srcid=5353&pointType=string&group=quotation_fiveday_ab&newFormat=1" +
		"&finClientType=pc&code=399001&query=399001&market_type=ab&is_kc=0"
	if rec.query != want {
		t.Errorf("查询串 = %q，期望 %q", rec.query, want)
	}
}

func TestGetQuoteFiveDayArrayForm(t *testing.T) {
	// 旧数据可能是数组形态，当前实现明确报错，不静默丢弃分时。
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, `{"ResultCode":0,"Result":{"newMarketData":{"keys":["timestamp","time"],
			"marketData":[{"date":"2026-11-09","p":[["0930,10.5"]]}]}}}`)
	})
	_, err := client.GetQuoteFiveDay(context.Background(), GetQuoteFiveDayRequest{Code: "399001"})
	if err == nil {
		t.Fatal("数组形态应返回明确错误")
	}
	if !strings.Contains(err.Error(), "不是字符串") {
		t.Errorf("错误信息未说明形态不支持：%v", err)
	}
}

func TestGetQuoteFiveDayRiskControl(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, `{"QueryID":"1","ResultCode":0,"Result":{"code":403,"isCaptchaEnabled":true,"msg":"hit risk"}}`)
	})
	if _, err := client.GetQuoteFiveDay(context.Background(), GetQuoteFiveDayRequest{Code: "399001"}); !IsRiskControl(err) {
		t.Fatalf("应识别为风控，实际 %v", err)
	}
}

func TestGetQuoteFiveDayValidates(t *testing.T) {
	client := New(Config{})
	if _, err := client.GetQuoteFiveDay(context.Background(), GetQuoteFiveDayRequest{}); err == nil {
		t.Fatal("code 为空应返回错误")
	}
	if _, err := client.GetQuoteFiveDay(context.Background(), GetQuoteFiveDayRequest{Code: "1", MarketType: "jp"}); err == nil {
		t.Fatal("market_type 非法应返回错误")
	}
}

// TestGetQuoteFiveDayTenColumns 覆盖真实响应里的 10 列形态：
// 比文档记录的 7 列多出 avgPrice、totalVolume、totalAmount。
func TestGetQuoteFiveDayTenColumns(t *testing.T) {
	const sample = `{"ResultCode":0,"Result":{"newMarketData":{
	  "headers":["时间戳","时间","价格","均价","涨跌额","涨跌幅","成交量","成交额","累积成交量","累积成交额"],
	  "keys":["timestamp","time","price","avgPrice","range","ratio","volume","amount","totalVolume","totalAmount"],
	  "marketData":[{"date":"2026-11-13","p":"1794553200,11-13 15:00,876.33,875.90,-31.80,-3.50,1234567,987654321.00,2345678,1876543210.00"}],
	  "maxPoints":"1201","cx":["11-13"]}}}`
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, sample)
	})
	result, err := client.GetQuoteFiveDay(context.Background(), GetQuoteFiveDayRequest{Code: "899050"})
	if err != nil {
		t.Fatalf("10 列形态解析失败：%v", err)
	}
	if len(result.Days) != 1 || len(result.Days[0].Points) != 1 {
		t.Fatalf("解析结果不足：%+v", result.Days)
	}
	point := result.Days[0].Points[0]
	if point.Price != 876.33 || point.AvgPrice != 875.90 {
		t.Errorf("价格/均价解析错误：%+v", point)
	}
	if point.Volume != 1234567 || point.Amount != 987654321 {
		t.Errorf("量额解析错误：%+v", point)
	}
	if point.TotalVolume != 2345678 || point.TotalAmount != 1876543210 {
		t.Errorf("累积量额解析错误：%+v", point)
	}
	if result.MaxPoints != "1201" || len(result.CX) != 1 {
		t.Errorf("元信息不符：%+v", result)
	}
}
