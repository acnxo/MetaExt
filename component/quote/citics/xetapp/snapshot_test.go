package xetapp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// snapshotSample 是文档里的真实响应片段（默认 20 列）。
const snapshotSample = `{
  "GRID0": [
    "名称|最新|幅度|涨跌|昨收|开盘|最高|最低|均价|总手|总额|现手|换手|振幅|委比|委差|量比|内盘|外盘|代码|",
    "沪深京全A|1488.03|-2.08%|-31.58|1519.61|1522.37|1526.09|1484.97|1504.97|10.37亿手|1.67万亿|1216.23万手|1.41%|2.71%|-2.04%|-125.33万手|1.02|--|--|994650|",
    "1.幸福蓝海|17.99|20.01%|3.00|14.99|14.90|17.99|14.90|17.05|60.75万手|10.36亿|8手|16.30%|20.61%|100.00%|7211.80手|1.92|27.02万|33.73万|300528|"
  ],
  "ACTION": "20199",
  "MAXCOUNT": "5579",
  "ERRORNO": "5581",
  "NEEDCHECK": "1|2|4|3|5|6|7|8|9|10|11|12|13|16|14|15|50|49|48|32|",
  "STOCKCODE": "994650",
  "STOCKNAME": "沪深京全A",
  "HQDATE": "20261113",
  "SERVERDATETIME": "2026-11-13 15:00:00"
}`

// newSnapshotClient 构造指向测试服务器的客户端，并记录最后一次表单与请求次数。
func newSnapshotClient(t *testing.T, handler func(w http.ResponseWriter, r *http.Request, attempt int)) (*Client, *http.Request, *int) {
	t.Helper()
	calls := 0
	var last *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		last = r
		handler(w, r, calls)
	}))
	t.Cleanup(server.Close)
	client := New(Config{
		GatewayURL: server.URL,
		// 测试里跳过退避等待，保持用例快速。
		Sleep: func(time.Duration) {},
	})
	return client, last, &calls
}

func writeBody(w http.ResponseWriter, body string) {
	w.Header().Set("content-type", "application/json")
	_, _ = w.Write([]byte(body))
}

func TestGetQuoteSnapshotParsesRows(t *testing.T) {
	calls := 0
	var form map[string][]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = r.ParseForm()
		form = r.PostForm
		if got := r.Header.Get("content-type"); !strings.Contains(got, "application/x-www-form-urlencoded") {
			t.Errorf("content-type = %q", got)
		}
		writeBody(w, snapshotSample)
	}))
	defer server.Close()
	client := New(Config{GatewayURL: server.URL, Sleep: func(time.Duration) {}})

	result, err := client.GetQuoteSnapshotCNAStock(context.Background(), SnapshotRequest{Pool: "all"})
	if err != nil {
		t.Fatalf("GetQuoteSnapshotCNAStock 返回错误：%v", err)
	}
	// 池自身那一行必须剔除。
	if len(result.Rows) != 1 {
		t.Fatalf("行数 = %d，期望 1（池自身应被剔除）", len(result.Rows))
	}
	row := result.Rows[0]
	if row.Code != "300528" {
		t.Errorf("code = %q，期望 300528", row.Code)
	}
	// 名称的 "1." 序号前缀要剥掉。
	if row.Name != "幸福蓝海" {
		t.Errorf("name = %q，期望 幸福蓝海", row.Name)
	}
	if row.Last != 17.99 || row.ChangeRatio != 20.01 || row.PreClose != 14.99 {
		t.Errorf("价格与涨跌幅解析错误：%+v", row)
	}
	// 带单位字段：万 → ×1e4，亿 → ×1e8。
	if row.Volume != 607500 {
		t.Errorf("volume = %d，期望 607500（60.75万手）", row.Volume)
	}
	if row.Amount != 1036000000 {
		t.Errorf("amount = %v，期望 1.036e9（10.36亿）", row.Amount)
	}
	if row.InnerVolume != 270200 || row.OuterVolume != 337300 {
		t.Errorf("内外盘解析错误：inner=%d outer=%d", row.InnerVolume, row.OuterVolume)
	}
	if row.TurnoverRate != 16.30 || row.VolumeRatio != 1.92 {
		t.Errorf("换手/量比解析错误：%+v", row)
	}
	// 元信息。
	if result.MaxCount != 5579 {
		t.Errorf("MaxCount = %d，期望 5579", result.MaxCount)
	}
	if result.PoolName != "沪深京全A" || result.HQDate != "20261113" {
		t.Errorf("元信息不符：%+v", result)
	}
	if len(result.NeedCheck) != 20 {
		t.Errorf("NeedCheck 列数 = %d，期望 20", len(result.NeedCheck))
	}

	// 表单参数：MaxCount 必须显式传，NeedCheck 以竖线结尾。
	if calls != 1 {
		t.Errorf("请求次数 = %d，期望 1", calls)
	}
	if form["action"][0] != ActionQuoteSnapshot || form["StockCode"][0] != DefaultPoolCode {
		t.Errorf("表单参数不符：%v", form)
	}
	if form["MaxCount"][0] != "60000" || form["ReqlinkType"][0] != "0" {
		t.Errorf("MaxCount / ReqlinkType 不符：%v", form)
	}
	need := form["NeedCheck"][0]
	if !strings.HasSuffix(need, "|") {
		t.Errorf("NeedCheck 应以竖线结尾：%q", need)
	}
	// 内部固定补上 1 与 32，默认集里本来就有。
	if !strings.Contains(need, "1|") || !strings.Contains(need, "32|") {
		t.Errorf("NeedCheck 缺少内部字段：%q", need)
	}
}

func TestGetQuoteSnapshotSkipsOwnRowByName(t *testing.T) {
	// 代码对不上但名称等于池名时，同样应剔除。
	const body = `{"GRID0":["名称|最新|代码|","沪深京全A|1488.03|999999|","1.测试股|10.00|300001|"],
	  "ERRORNO":"4","NEEDCHECK":"1|2|32|","STOCKCODE":"994650","STOCKNAME":"沪深京全A"}`
	client, _, _ := newSnapshotClient(t, func(w http.ResponseWriter, _ *http.Request, _ int) {
		writeBody(w, body)
	})
	result, err := client.GetQuoteSnapshotCNAStock(context.Background(), SnapshotRequest{})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	if len(result.Rows) != 1 || result.Rows[0].Code != "300001" {
		t.Fatalf("按名称剔除池自身失败：%+v", result.Rows)
	}
}

func TestGetQuoteSnapshotBusinessError(t *testing.T) {
	client, _, _ := newSnapshotClient(t, func(w http.ResponseWriter, _ *http.Request, _ int) {
		writeBody(w, `{"ERRORNO":"-206008","ERRORMESSAGE":"未找到匹配的证券代码!代码:999999|0"}`)
	})
	_, err := client.GetQuoteSnapshotCNAStock(context.Background(), SnapshotRequest{Pool: "999999"})
	if err == nil {
		t.Fatal("负值 ERRORNO 应返回错误")
	}
	if !IsBusinessError(err) {
		t.Errorf("IsBusinessError = false：%v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.ErrorNo != -206008 {
		t.Errorf("错误码不符：%v", err)
	}
	if !strings.Contains(err.Error(), "未找到匹配的证券代码") {
		t.Errorf("错误信息未携带服务端提示：%v", err)
	}
}

func TestGetQuoteSnapshotEmptyGrid(t *testing.T) {
	client, _, _ := newSnapshotClient(t, func(w http.ResponseWriter, _ *http.Request, _ int) {
		writeBody(w, `{"ERRORNO":"2","MAXCOUNT":"0"}`)
	})
	if _, err := client.GetQuoteSnapshotCNAStock(context.Background(), SnapshotRequest{}); err == nil {
		t.Fatal("没有 GRID0 时应返回错误")
	}
}

func TestGetQuoteSnapshotRetriesTruncatedBody(t *testing.T) {
	// 第一次返回被截断的 JSON，第二次完整：应自动重试并成功。
	client, _, calls := newSnapshotClient(t, func(w http.ResponseWriter, _ *http.Request, attempt int) {
		if attempt == 1 {
			writeBody(w, `{"GRID0":["名称|最新|代码|","1.测试股|10.0`)
			return
		}
		writeBody(w, snapshotSample)
	})
	result, err := client.GetQuoteSnapshotCNAStock(context.Background(), SnapshotRequest{})
	if err != nil {
		t.Fatalf("重试后应成功：%v", err)
	}
	if *calls != 2 {
		t.Errorf("请求次数 = %d，期望 2", *calls)
	}
	if len(result.Rows) != 1 {
		t.Errorf("行数 = %d，期望 1", len(result.Rows))
	}
}

func TestGetQuoteSnapshotRetryExhausted(t *testing.T) {
	client, _, calls := newSnapshotClient(t, func(w http.ResponseWriter, _ *http.Request, _ int) {
		writeBody(w, `{"GRID0":[`)
	})
	if _, err := client.GetQuoteSnapshotCNAStock(context.Background(), SnapshotRequest{}); err == nil {
		t.Fatal("始终截断时应返回错误")
	}
	if *calls != DefaultRetries {
		t.Errorf("请求次数 = %d，期望 %d", *calls, DefaultRetries)
	}
}

func TestGetQuoteSnapshotDepth(t *testing.T) {
	// 码 33–42 每码两段「价格|数量」，价格原始单位是厘。
	const body = `{"GRID0":["名称|卖一|买一|代码|",
	  "1.测试股|12.340|9900|12.330|8800|300001|"],
	  "ERRORNO":"4","NEEDCHECK":"1|37|38|32|","STOCKCODE":"994650","STOCKNAME":"沪深京全A"}`
	client, _, _ := newSnapshotClient(t, func(w http.ResponseWriter, _ *http.Request, _ int) {
		writeBody(w, body)
	})
	result, err := client.GetQuoteSnapshotCNAStock(context.Background(), SnapshotRequest{
		Fields: []int{1, 37, 38, 32},
	})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	depth, ok := result.Rows[0].Values["depth"].(map[string]DepthLevel)
	if !ok {
		t.Fatalf("depth 未解析：%+v", result.Rows[0].Values)
	}
	// 12.340 厘 = 0.01234 元，四舍五入到 4 位。
	if got := depth["ask1"].Price; got != 0.0123 {
		t.Errorf("ask1 价格 = %v，期望 0.0123", got)
	}
	if got := depth["ask1"].Volume; got != 9900 {
		t.Errorf("ask1 数量 = %d，期望 9900", got)
	}
	if got := depth["bid1"].Volume; got != 8800 {
		t.Errorf("bid1 数量 = %d，期望 8800", got)
	}
}

func TestGetQuoteSnapshotExtraFields(t *testing.T) {
	var need string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		need = r.PostForm.Get("NeedCheck")
		writeBody(w, `{"GRID0":["名称|本月涨幅|5日净流入|代码|","1.测试股|3.20|1.50亿|300001|"],
		  "ERRORNO":"4","NEEDCHECK":"1|341|372|32|","STOCKCODE":"994650","STOCKNAME":"沪深京全A"}`)
	}))
	defer server.Close()
	client := New(Config{GatewayURL: server.URL, Sleep: func(time.Duration) {}})

	result, err := client.GetQuoteSnapshotCNAStock(context.Background(), SnapshotRequest{
		Fields:      []int{1, 32},
		ExtraFields: []int{341, 372},
	})
	if err != nil {
		t.Fatalf("返回错误：%v", err)
	}
	// 追加字段应写进 NeedCheck。
	for _, want := range []string{"341|", "372|"} {
		if !strings.Contains(need, want) {
			t.Errorf("NeedCheck 缺少 %s：%q", want, need)
		}
	}
	values := result.Rows[0].Values
	if got := values["change_pct_month"]; got != 3.2 {
		t.Errorf("change_pct_month = %v，期望 3.2", got)
	}
	if got := values["net_inflow_5d"]; got != 1.5e8 {
		t.Errorf("net_inflow_5d = %v，期望 1.5e8", got)
	}
}

func TestGetQuoteSnapshotPools(t *testing.T) {
	if resolvePool("") != DefaultPoolCode {
		t.Errorf("空池应取默认值：%q", resolvePool(""))
	}
	if resolvePool("hs300") != "000300" {
		t.Errorf("别名 hs300 未解析：%q", resolvePool("hs300"))
	}
	// 非别名按内部代码原样使用。
	if resolvePool("994649") != "994649" {
		t.Errorf("内部代码应原样使用：%q", resolvePool("994649"))
	}
}

func TestGetQuoteSnapshotContextCanceled(t *testing.T) {
	client, _, _ := newSnapshotClient(t, func(w http.ResponseWriter, _ *http.Request, _ int) {
		writeBody(w, snapshotSample)
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.GetQuoteSnapshotCNAStock(ctx, SnapshotRequest{}); err == nil {
		t.Fatal("ctx 取消应返回错误")
	}
}

func TestConvertValue(t *testing.T) {
	cases := []struct {
		raw  string
		kind FieldKind
		want any
	}{
		{"17.99", KindPrice, 17.99},
		{"20.01%", KindPct, 20.01},
		{"10.36亿", KindAmount, 1.036e9},
		{"1.67万亿", KindAmount, 1.67e12},
		{"60.75万手", KindVol, int64(607500)},
		{"--", KindPrice, nil},
		{"", KindStr, nil},
		{"正常交易", KindStr, "正常交易"},
		// 无法解析时保留原文，避免吞掉说明性文本。
		{"未知", KindPrice, "未知"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s/%s", tc.raw, tc.kind), func(t *testing.T) {
			if got := convertValue(tc.raw, tc.kind); got != tc.want {
				t.Errorf("convertValue(%q, %q) = %#v，期望 %#v", tc.raw, tc.kind, got, tc.want)
			}
		})
	}
}

func TestDefaultFieldsMatchSpec(t *testing.T) {
	// 默认字段集必须全部有命名规格，否则强类型字段会拿不到值。
	for _, code := range DefaultFields {
		if _, ok := fieldSpecs[code]; !ok {
			t.Errorf("默认字段 %d 没有命名规格", code)
		}
	}
	if len(DefaultFields) != 45 {
		t.Errorf("默认字段数 = %d，期望 45", len(DefaultFields))
	}
	if len(fieldSpecs) != 207 {
		t.Errorf("字段规格数 = %d，期望 207", len(fieldSpecs))
	}
}
