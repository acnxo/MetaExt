package quoteapiv2

import (
	"strings"
	"testing"
)

// TestSignVectors 固定输入到固定 token 的回归向量，用于防止签名实现漂移。
//
// 这些值是协议侧的既定结果，改动签名逻辑时必须先确认它们仍然成立。
func TestSignVectors(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"空输入降级为 quote", "", "58af97268c"},
		{"字面量 quote", "quote", "58af97268c"},
		{"普通 JSON 输入", `{"a":"1","b":"2"}`, "39adef0837"},
		{
			"完整 K 线参数",
			`{"stockId":"70000294","marketType":"2","type":"2","marketCode":"70","instrumentType":"9","subInstrumentType":"9008","_":"1780507278122"}`,
			"45671b44c1",
		},
		{
			"仅时间戳不同的同一组参数",
			`{"stockId":"70000294","marketType":"2","type":"2","marketCode":"70","instrumentType":"9","subInstrumentType":"9008","_":"1780600000000"}`,
			"7feb1b1665",
		},
		{
			"使用下划线命名的参数",
			`{"stock_id":"76033806042898","market_type":"1","type":"0","subType":"0","_":"1781592384789"}`,
			"055b0960ec",
		},
	}
	for _, tc := range cases {
		if got := Sign(tc.input); got != tc.want {
			t.Errorf("%s：token 期望 %s，实际 %s", tc.name, tc.want, got)
		}
	}
}

// TestSignParams 验证「参数列表 → 签名输入 → token」整条链路，以及键序保真。
func TestSignParams(t *testing.T) {
	params := Params(
		"stockId", "70000294",
		"marketType", "2",
		"type", "2",
		"marketCode", "70",
		"instrumentType", "9",
		"subInstrumentType", "9008",
		"_", "1780507278122",
	)
	want := `{"stockId":"70000294","marketType":"2","type":"2","marketCode":"70","instrumentType":"9","subInstrumentType":"9008","_":"1780507278122"}`
	if got := SignInput(params); got != want {
		t.Fatalf("签名输入不符：\n期望 %s\n实际 %s", want, got)
	}
	if got := SignParams(params); got != "45671b44c1" {
		t.Fatalf("token 期望 45671b44c1，实际 %s", got)
	}
}

// TestSignParamsOrderSensitive 键序不同即应得到不同 token，且查询串不得被重排。
func TestSignParamsOrderSensitive(t *testing.T) {
	original := Params(
		"stockId", "70000294",
		"marketType", "2",
		"type", "2",
		"marketCode", "70",
		"instrumentType", "9",
		"subInstrumentType", "9008",
		"_", "1780507278122",
	)
	swapped := Params(
		"stockId", "70000294",
		"type", "2",
		"marketType", "2",
		"marketCode", "70",
		"instrumentType", "9",
		"subInstrumentType", "9008",
		"_", "1780507278122",
	)
	if SignParams(swapped) == SignParams(original) {
		t.Fatal("调换键序后 token 不应相同")
	}

	query := SignedQuery(original)
	wantPrefix := "stockId=70000294&marketType=2&type=2&marketCode=70"
	if !strings.HasPrefix(query, wantPrefix) {
		t.Fatalf("查询串键序不符：%s", query)
	}
	if strings.Index(query, "_=") < strings.Index(query, "subInstrumentType=") {
		t.Fatalf("时间戳参数应位于最后：%s", query)
	}
}

// TestParams 参数个数为奇数时忽略末尾落单的名称。
func TestParams(t *testing.T) {
	params := Params("a", "1", "b", "2", "c")
	if len(params) != 2 {
		t.Fatalf("期望 2 个参数，实际 %d：%+v", len(params), params)
	}
	if params[0] != (Param{Name: "a", Value: "1"}) || params[1] != (Param{Name: "b", Value: "2"}) {
		t.Fatalf("参数内容不符：%+v", params)
	}
	if got := SignInput(nil); got != "" {
		t.Fatalf("空参数列表的签名输入应为空串，实际 %q", got)
	}
}

// TestSignInputEscapes 签名输入必须做 JSON 最小转义，非 ASCII 字符保持原样。
func TestSignInputEscapes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"双引号", `a"b`, `{"k":"a\"b"}`},
		{"反斜杠", `a\b`, `{"k":"a\\b"}`},
		{"换行", "a\nb", `{"k":"a\nb"}`},
		{"回车", "a\rb", `{"k":"a\rb"}`},
		{"制表符", "a\tb", `{"k":"a\tb"}`},
		{"中文不转义", "中国", `{"k":"中国"}`},
	}
	for _, tc := range cases {
		got := SignInput([]Param{{Name: "k", Value: tc.in}})
		if got != tc.want {
			t.Errorf("%s：期望 %s，实际 %s", tc.name, tc.want, got)
		}
	}
}

// TestSignedQueryEscapes 查询串必须对键与值做 URL 转义。
func TestSignedQueryEscapes(t *testing.T) {
	query := SignedQuery([]Param{
		{Name: "k", Value: "a b&c=d/中"},
		{Name: "x", Value: "+"},
	})
	want := "k=a+b%26c%3Dd%2F%E4%B8%AD&x=%2B"
	if query != want {
		t.Fatalf("查询串期望 %s，实际 %s", want, query)
	}
}
