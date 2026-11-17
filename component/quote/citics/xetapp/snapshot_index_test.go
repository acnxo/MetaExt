package xetapp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// indexBody 拼一个指数池的响应：records 是「名称|最新|代码|」形式的记录行。
func indexBody(action string, records ...string) string {
	parts := make([]string, 0, len(records)+1)
	parts = append(parts, `"名称|最新|代码|"`)
	for _, record := range records {
		parts = append(parts, `"`+record+`"`)
	}
	return `{"GRID0":[` + strings.Join(parts, ",") +
		`],"ACTION":"` + action +
		`","MAXCOUNT":"` + strconv.Itoa(len(records)) +
		`","ERRORNO":"` + strconv.Itoa(len(records)+2) +
		`","NEEDCHECK":"1|2|32|","HQDATE":"20261113",` +
		`"SERVERDATETIME":"2026-11-13 15:00:00"}`
}

// indexServer 按「action|StockCode」返回四个池的样本，并记录请求顺序。
func indexServer(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.PostForm.Get("action")
		stock := r.PostForm.Get("StockCode")
		calls = append(calls, action+"|"+stock)
		if got := r.PostForm.Get("MaxCount"); got != "60000" {
			t.Errorf("%s 的 MaxCount = %q，期望 60000", action, got)
		}
		switch action + "|" + stock {
		case "20191|1100":
			writeBody(w, indexBody("20191", "1.上证指数|3634.35|1A0001|", "2.沪深300|4210.10|1A0300|"))
		case "20192|国证指数":
			writeBody(w, indexBody("20192", "1.沪深300|4210.10|1A0300|", "2.龙头家电|11150.85|980028|"))
		case "20192|股转指数":
			writeBody(w, indexBody("20192", "1.三板做市|1234.56|899304|"))
		case "20198|":
			writeBody(w, indexBody("20198", "1.上证指数|3634.35|1A0001|", "2.创业板指|2380.00|399006|"))
		default:
			t.Errorf("未预期的请求：action=%s stock=%q", action, stock)
			writeBody(w, `{"ERRORNO":"-206021","ERRORMESSAGE":"查无相应的排名信息!"}`)
		}
	}))
	return server, &calls
}

func TestGetQuoteSnapshotIndexMergesAndDedupes(t *testing.T) {
	server, calls := indexServer(t)
	defer server.Close()
	client := New(Config{GatewayURL: server.URL, Sleep: func(time.Duration) {}})

	result, err := client.GetQuoteSnapshotCNAIndex(context.Background(), IndexSnapshotRequest{})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}

	// 4 个池按 IndexBoards 的顺序请求，大盘池不带 StockCode。
	wantCalls := []string{"20191|1100", "20192|国证指数", "20192|股转指数", "20198|"}
	if strings.Join(*calls, ",") != strings.Join(wantCalls, ",") {
		t.Fatalf("请求顺序不符：期望 %v，实际 %v", wantCalls, *calls)
	}

	// 去重后按首次出现顺序：中证 2 + 国证 1（1A0300 重复）+ 股转 1 + 大盘 1（1A0001 重复）。
	wantCodes := []string{"1A0001", "1A0300", "980028", "899304", "399006"}
	gotCodes := make([]string, 0, len(result.Rows))
	for _, row := range result.Rows {
		gotCodes = append(gotCodes, row.Code)
	}
	if strings.Join(gotCodes, ",") != strings.Join(wantCodes, ",") {
		t.Fatalf("合并结果不符：期望 %v，实际 %v", wantCodes, gotCodes)
	}

	wantPools := []IndexPool{
		{Name: "中证指数", Action: "20191", Board: "1100", MaxCount: 2, Fetched: 2, Added: 2, ErrorNo: 4},
		{Name: "国证指数", Action: "20192", Board: "国证指数", MaxCount: 2, Fetched: 2, Added: 1, ErrorNo: 4},
		{Name: "股转指数", Action: "20192", Board: "股转指数", MaxCount: 1, Fetched: 1, Added: 1, ErrorNo: 3},
		{Name: "大盘指数", Action: "20198", Board: "", MaxCount: 2, Fetched: 2, Added: 1, ErrorNo: 4},
	}
	if len(result.Pools) != len(wantPools) {
		t.Fatalf("池数 = %d，期望 %d", len(result.Pools), len(wantPools))
	}
	for i, want := range wantPools {
		if result.Pools[i] != want {
			t.Errorf("池 %d 统计不符：期望 %+v，实际 %+v", i, want, result.Pools[i])
		}
	}

	// 名称序号前缀已剥离，元信息取首个池的回显。
	if result.Rows[0].Name != "上证指数" || result.Rows[0].Last != 3634.35 {
		t.Errorf("首条解析不符：%+v", result.Rows[0])
	}
	if result.HQDate != "20261113" || result.ServerDateTime != "2026-11-13 15:00:00" {
		t.Errorf("元信息不符：%q / %q", result.HQDate, result.ServerDateTime)
	}
	if len(result.NeedCheck) != 3 || result.NeedCheck[2] != 32 {
		t.Errorf("NeedCheck 不符：%v", result.NeedCheck)
	}
}

func TestGetQuoteSnapshotIndexPoolError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostForm.Get("action") == ActionQuoteSnapshotMarketIndex {
			writeBody(w, `{"ERRORNO":"-989905","ERRORMESSAGE":"功能被服务端禁用"}`)
			return
		}
		writeBody(w, indexBody(r.PostForm.Get("action"), "1.上证指数|3634.35|1A0001|"))
	}))
	defer server.Close()
	client := New(Config{GatewayURL: server.URL, Sleep: func(time.Duration) {}})

	_, err := client.GetQuoteSnapshotCNAIndex(context.Background(), IndexSnapshotRequest{})
	if err == nil {
		t.Fatal("任一池失败应返回错误")
	}
	// 错误信息要能定位到具体池，且底层仍是 *APIError。
	if !strings.Contains(err.Error(), "大盘指数") {
		t.Errorf("错误信息缺少池名：%v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.ErrorNo != -989905 {
		t.Errorf("应包装 APIError：%+v", apiErr)
	}
}

func TestGetQuoteSnapshotIndexCustomBoards(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		calls = append(calls, r.PostForm.Get("action")+"|"+r.PostForm.Get("StockCode"))
		writeBody(w, indexBody("20192", "1.龙头家电|11150.85|980028|"))
	}))
	defer server.Close()
	client := New(Config{GatewayURL: server.URL, Sleep: func(time.Duration) {}})

	result, err := client.GetQuoteSnapshotCNAIndex(context.Background(), IndexSnapshotRequest{
		Boards: []IndexBoard{{Name: "自定义", Action: ActionQuoteSnapshotETF, Board: "国证指数"}},
		Fields: []int{1, 2, 32},
	})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if len(calls) != 1 || calls[0] != "20192|国证指数" {
		t.Fatalf("自定义池应只请求一次：%v", calls)
	}
	if len(result.Rows) != 1 || result.Rows[0].Code != "980028" {
		t.Fatalf("结果不符：%+v", result.Rows)
	}
}
