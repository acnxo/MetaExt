package xetapp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// etfSample 是文档里的真实响应片段：没有「池自身」行，`STOCKCODE` / `STOCKNAME` 也不返回。
const etfSample = `{
  "GRID0": [
    "名称|最新|幅度|涨跌|代码|",
    "1.科创200ETF工银|1.194|4.01%|0.046|589200|",
    "2.标普油气ETF富国|1.338|3.08%|0.040|513350|"
  ],
  "ACTION": "20192",
  "MAXCOUNT": "1727",
  "ERRORNO": "1729",
  "NEEDCHECK": "1|2|4|3|32|",
  "HQDATE": "20261113",
  "SERVERDATETIME": "2026-11-13 15:00:00"
}`

func TestGetQuoteSnapshotETFForm(t *testing.T) {
	var form map[string][]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.PostForm
		writeBody(w, etfSample)
	}))
	defer server.Close()
	client := New(Config{GatewayURL: server.URL, Sleep: func(time.Duration) {}})

	result, err := client.GetQuoteSnapshotCNAETF(context.Background(), ETFSnapshotRequest{})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	wantForm := map[string]string{
		"action":      "20192",
		"ReqlinkType": "0",
		"StockCode":   DefaultETFBoard,
		"MaxCount":    "60000",
	}
	for key, want := range wantForm {
		if got := fieldValue(form, key); got != want {
			t.Errorf("表单 %s = %q，期望 %q", key, got, want)
		}
	}
	if _, ok := form["StartPos"]; ok {
		t.Errorf("StartPos 为 0 时不应传：%v", form)
	}

	// ETF 接口没有「池自身」行，两条记录都应保留。
	if len(result.Rows) != 2 {
		t.Fatalf("行数 = %d，期望 2", len(result.Rows))
	}
	if got := result.Rows[0].Name; got != "科创200ETF工银" {
		t.Errorf("名称未剥离序号前缀：%q", got)
	}
	if got := result.Rows[1].Code; got != "513350" {
		t.Errorf("代码 = %q，期望 513350", got)
	}
	if got := result.Rows[0].ChangeRatio; got != 4.01 {
		t.Errorf("涨跌幅 = %v，期望 4.01", got)
	}
	if result.MaxCount != 1727 || result.ErrorNo != 1729 {
		t.Errorf("分页/错误码不符：MaxCount=%d ErrorNo=%d", result.MaxCount, result.ErrorNo)
	}
	// 该接口不返回池名，PoolCode 回落为板块中文名。
	if result.PoolCode != DefaultETFBoard || result.PoolName != "" {
		t.Errorf("池信息不符：%q / %q", result.PoolCode, result.PoolName)
	}
}

func TestGetQuoteSnapshotETFCustomBoardAndPaging(t *testing.T) {
	var form map[string][]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.PostForm
		writeBody(w, `{"GRID0":["名称|代码|"],"ACTION":"20192","ERRORNO":"2","NEEDCHECK":"1|32|"}`)
	}))
	defer server.Close()
	client := New(Config{GatewayURL: server.URL, Sleep: func(time.Duration) {}})

	if _, err := client.GetQuoteSnapshotCNAETF(context.Background(), ETFSnapshotRequest{
		Board:    " 国证指数 ",
		StartPos: 2,
	}); err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if got := fieldValue(form, "StockCode"); got != "国证指数" {
		t.Errorf("StockCode = %q，期望 国证指数", got)
	}
	if got := fieldValue(form, "StartPos"); got != "2" {
		t.Errorf("StartPos = %q，期望 2", got)
	}
}

func TestGetQuoteSnapshotETFExtraFields(t *testing.T) {
	const body = `{
	  "GRID0": ["名称|单位净值|累计净值|日期|基金净值|基金规模|折价率|代码|",
	    "1.沪深300ETF华泰柏瑞|4.092|1.844|20261113|4.0927|1874.81亿|-0.24%|510300|"],
	  "ACTION": "20192",
	  "MAXCOUNT": "1727",
	  "ERRORNO": "2",
	  "NEEDCHECK": "1|19|20|23|84|326|347|32|"
	}`
	var need string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		need = r.PostForm.Get("NeedCheck")
		writeBody(w, body)
	}))
	defer server.Close()
	client := New(Config{GatewayURL: server.URL, Sleep: func(time.Duration) {}})

	result, err := client.GetQuoteSnapshotCNAETF(context.Background(), ETFSnapshotRequest{
		Fields:      []int{1, 32},
		ExtraFields: []int{19, 20, 23, 84, 326, 347},
	})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	for _, want := range []string{"19|", "20|", "23|", "84|", "326|", "347|"} {
		if !strings.Contains(need, want) {
			t.Errorf("NeedCheck 缺少 %s：%q", want, need)
		}
	}
	row := result.Rows[0]
	if row.UnitNetValue != 4.092 || row.AccNetValue != 1.844 {
		t.Errorf("净值解析不符：%+v", row)
	}
	if row.Date != "20261113" {
		t.Errorf("净值日期 = %q，期望 20261113", row.Date)
	}
	if row.FundNav != 4.0927 {
		t.Errorf("基金净值 = %v，期望 4.0927", row.FundNav)
	}
	if row.FundScale != 1.87481e11 {
		t.Errorf("基金规模 = %v，期望 1.87481e11", row.FundScale)
	}
	if row.DiscountRate != -0.24 {
		t.Errorf("折价率 = %v，期望 -0.24", row.DiscountRate)
	}
	// 多取的内部字段只在调用方没要时才剔除；这里 1 与 32 是显式要的。
	for _, key := range []string{"name", "code", "unit_net_value", "fund_scale"} {
		if _, ok := row.Values[key]; !ok {
			t.Errorf("Values 缺少 %s", key)
		}
	}
}

func TestGetQuoteSnapshotETFBusinessError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeBody(w, `{"ERRORNO":"-206021","ERRORMESSAGE":"查无相应的排名信息!"}`)
	}))
	defer server.Close()
	client := New(Config{GatewayURL: server.URL, Sleep: func(time.Duration) {}})

	_, err := client.GetQuoteSnapshotCNAETF(context.Background(), ETFSnapshotRequest{Board: "沪市ETF基金"})
	if err == nil || !IsBusinessError(err) {
		t.Fatalf("应识别为业务错误：%v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.ErrorNo != -206021 {
		t.Fatalf("ERRORNO 不符：%+v", apiErr)
	}
}

func TestGetQuoteSnapshotETFFiltersUnrequestedInternals(t *testing.T) {
	// 只请求「最新」一个字段时，内部多取的名称与代码要从事后结果中剔除，
	// 但 Row.Name / Row.Code 仍然可用（便于调用方定位）。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeBody(w, `{"GRID0":["最新|名称|代码|","1.194|科创200ETF工银|589200|"],
		  "ACTION":"20192","ERRORNO":"2","NEEDCHECK":"2|1|32|"}`)
	}))
	defer server.Close()
	client := New(Config{GatewayURL: server.URL, Sleep: func(time.Duration) {}})

	result, err := client.GetQuoteSnapshotCNAETF(context.Background(), ETFSnapshotRequest{Fields: []int{2}})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	row := result.Rows[0]
	if row.Last != 1.194 || row.Name != "科创200ETF工银" || row.Code != "589200" {
		t.Fatalf("强类型字段不符：%+v", row)
	}
	if _, ok := row.Values["name"]; ok {
		t.Error("未请求的 name 不应出现在 Values")
	}
	if _, ok := row.Values["code"]; ok {
		t.Error("未请求的 code 不应出现在 Values")
	}
	if _, ok := row.Values["last"]; !ok {
		t.Error("Values 应包含 last")
	}
}

// fieldValue 取表单首个值。
func fieldValue(form map[string][]string, key string) string {
	if values := form[key]; len(values) > 0 {
		return values[0]
	}
	return ""
}
