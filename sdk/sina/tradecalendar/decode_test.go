package tradecalendar

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// calendarEncoded 是 2026-11-02 抓取的真实编码串（469 个字符）。
const calendarEncoded = "" +
	"LC/AAApNDXCw6mHbaPgkryxXv10eAJP1LW0SD39aT7+NV44Xba3PxCgTdrp5BkYVAc11hWvg0c/19UAc7jNtHQyWBAu2xmGu" +
	"ZI1NVAc3FepphjnTBw1X4hmGu+ypVAcvFenpBXPqCc6F4ZmGueLFwbIN8QTDXPsCc1FepphjvOoCc8FepphjvcgFO3CP00wx" +
	"XXWhrkUdZrIJpw9X3ThrlEp6hlGc88Kcem0VeFpZM46VV4MrTC2KScKc811U4aLXUdlzINc9lTrwFW3T52KPj0mDueVFuUR1" +
	"RtiEoCXfdgFOOSGRXnUhrXWhb0kt6Rk2pU44JV4SrTyU9wSDHPwCnXdP1FuiUM44r7qwdKqcYrIZpw1DqgrlU5IrHRawxjrw" +
	"BaqcbrIt9gr3UhDtOpyVNjEnCHPnC3royNWvi0gjHXBXYdRlLbFpdJFueSFcqkK30sSDO+68K46IVOwVkaBX/"

// calendarBody 是完整响应文本。
const calendarBody = `var datelist="` + calendarEncoded + `";var KLC_TD_SH=datelist;`

// mustParse 解析样本，失败即终止测试。
func mustParse(t *testing.T) *TradeCalendar {
	t.Helper()
	cal, err := Parse(calendarBody)
	if err != nil {
		t.Fatalf("解析样本失败: %v", err)
	}
	return cal
}

// TestParseSample 校验样本的交易日数量、首尾日期与抽样日期。
func TestParseSample(t *testing.T) {
	cal := mustParse(t)

	if got := cal.Len(); got != 8796 {
		t.Fatalf("交易日数量期望 8796，实际 %d", got)
	}
	first, ok := cal.First()
	if !ok || first.Format("2006-01-02") != "1990-12-19" {
		t.Fatalf("首个交易日期望 1990-12-19，实际 %v", first)
	}
	last, ok := cal.Last()
	if !ok || last.Format("2006-01-02") != "2026-12-31" {
		t.Fatalf("末个交易日期望 2026-12-31，实际 %v", last)
	}

	// 抽样：开市首日、样本抓取当日、当年最后一个交易日。
	for _, day := range []string{"1990-12-19", "2026-11-02", "2026-12-31"} {
		d := parseDay(t, day)
		if !cal.IsTradeDay(d) {
			t.Errorf("%s 应为交易日", day)
		}
	}
}

// TestParseSampleNonTradeDays 周末、法定节假日与数据源缺口都应判为非交易日。
func TestParseSampleNonTradeDays(t *testing.T) {
	cal := mustParse(t)
	cases := []struct {
		day string
		why string
	}{
		{"2026-10-01", "国庆节"},
		{"2026-10-02", "国庆节"},
		{"2027-01-01", "元旦（超出数据范围）"},
		{"2026-11-07", "周六"},
		{"2026-11-08", "周日"},
		{"2000-01-01", "元旦"},
		{"1992-05-04", "数据源未收录"},
	}
	for _, c := range cases {
		if cal.IsTradeDay(parseDay(t, c.day)) {
			t.Errorf("%s（%s）不应判为交易日", c.day, c.why)
		}
	}
}

// TestParseDatesAreAscending 日期必须严格升序且不含周末。
func TestParseDatesAreAscending(t *testing.T) {
	cal := mustParse(t)
	for i := 1; i < cal.Len(); i++ {
		if !cal.Dates[i].After(cal.Dates[i-1]) {
			t.Fatalf("第 %d 项未严格递增：%v 之后是 %v", i, cal.Dates[i-1], cal.Dates[i])
		}
	}
	for _, d := range cal.Dates {
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			t.Fatalf("出现周末日期：%v", d)
		}
		if d.Hour() != 0 || d.Minute() != 0 || d.Second() != 0 {
			t.Fatalf("日期应为 UTC 零点：%v", d)
		}
	}
}

// TestParseNoData 缺少 datelist 声明时返回 ErrNoData。
func TestParseNoData(t *testing.T) {
	cases := map[string]string{
		"空响应":    "",
		"其他内容":   "<html>forbidden</html>",
		"引号内为空":  `var datelist="";var KLC_TD_SH=datelist;`,
		"变量名不同":  `var datelist2="abc";`,
		"没有等号声明": "datelist abc",
	}
	for name, body := range cases {
		_, err := Parse(body)
		if !errors.Is(err, ErrNoData) {
			t.Errorf("%s：期望 ErrNoData，实际 %v", name, err)
		}
	}
}

// TestParseChecksumMismatch 头部校验和非 0/1 时返回 ErrChecksum。
func TestParseChecksumMismatch(t *testing.T) {
	// 第 3 个字符承载头部的第二个字段，原值为 '/'（63），改成 '9'（61）后校验和为 2。
	broken := "LC9" + calendarEncoded[3:]
	_, err := Parse(`var datelist="` + broken + `";`)
	if !errors.Is(err, ErrChecksum) {
		t.Fatalf("期望 ErrChecksum，实际 %v", err)
	}
}

// TestParseInvalidEncoding 非法字符或提前截断时返回 ErrEncoding。
func TestParseInvalidEncoding(t *testing.T) {
	cases := map[string]string{
		"非法字符":     "LC/AAApNDXCw6mHbaPg===",
		"头部不完整":    "LC",
		"起始游标不完整":  "LC/AA",
		"截断到 6 位":  "LC/AAA",
		"截断到 20 位": calendarEncoded[:20],
	}
	for name, encoded := range cases {
		_, err := Parse(`var datelist="` + encoded + `";`)
		if !errors.Is(err, ErrEncoding) {
			t.Errorf("%s：期望 ErrEncoding，实际 %v", name, err)
		}
	}
}

// TestDecodeEmpty 直接解码空编码串时报编码错误。
func TestDecodeEmpty(t *testing.T) {
	if _, err := decode(""); !errors.Is(err, ErrEncoding) {
		t.Fatalf("期望 ErrEncoding，实际 %v", err)
	}
}

// TestParseCursorAbnormal 起止游标异常时返回 ErrEncoding。
func TestParseCursorAbnormal(t *testing.T) {
	// header 正常，起始游标 10、结束游标 0，结束小于起始。
	encoded := encodeFields(t, 0, 63, 10, 0)
	_, err := Parse(`var datelist="` + encoded + `";`)
	if !errors.Is(err, ErrEncoding) {
		t.Fatalf("期望 ErrEncoding，实际 %v", err)
	}
}

// TestParseMinimalCalendar 起止游标均为 0 时应得到 1 个日期（1990-12-19）。
func TestParseMinimalCalendar(t *testing.T) {
	encoded := encodeFields(t, 0, 63, 0, 0)
	cal, err := Parse(`var datelist="` + encoded + `";`)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if cal.Len() != 1 || cal.Dates[0].Format("2006-01-02") != "1990-12-19" {
		t.Fatalf("期望仅 1990-12-19，实际 %v", cal.Dates)
	}
}

// TestReadValueCrossing30Bits 跨 30 位边界的字段需要拼接，不能位移溢出。
func TestReadValueCrossing30Bits(t *testing.T) {
	d := newDecoder(strings.Repeat("/", 8)) // 每个字符都是 63，即 6 位全 1
	got, err := d.readValue(36)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	want := (1 << 36) - 1
	if got != want {
		t.Fatalf("读取 36 位期望 %d，实际 %d", want, got)
	}
}

// TestReadBitsNonPositiveLength 长度 <= 0 时字段记为 0，且不消耗位。
func TestReadBitsNonPositiveLength(t *testing.T) {
	d := newDecoder(calendarEncoded)
	values, err := d.readBits([]int{-6, 0})
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(values) != 2 || values[0] != 0 || values[1] != 0 {
		t.Fatalf("期望两个 0，实际 %v", values)
	}
	if d.pos != 0 || d.off != 0 {
		t.Fatalf("不应消耗位，实际 pos=%d off=%d", d.pos, d.off)
	}
}

// TestNextBitExhausted 位流末尾按 0 处理，不越界。
func TestNextBitExhausted(t *testing.T) {
	d := newDecoder("")
	if d.nextBit() {
		t.Fatal("空位流不应读出 1")
	}
	if !d.exhausted() {
		t.Fatal("空位流应处于末尾状态")
	}
}

// TestIndexedAlphabet 字符表映射与标准 Base64 一致。
func TestIndexedAlphabet(t *testing.T) {
	if indexOf('/') != 63 || indexOf('+') != 62 || indexOf('A') != 0 || indexOf('-') != -1 {
		t.Fatal("字符表映射不符")
	}
}

// parseDay 把 YYYY-MM-DD 解析为测试用日期。
func parseDay(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("解析日期 %s 失败: %v", s, err)
	}
	return d
}

// encodeFields 按解码规则把头部与两个游标打包成编码串，用于构造边界样本。
func encodeFields(t *testing.T, header12, header6, start, end int) string {
	t.Helper()
	var bitsArr []int
	appendBits := func(value, n int) {
		for i := 0; i < n; i++ {
			bitsArr = append(bitsArr, (value>>i)&1)
		}
	}
	appendBits(header12, 12)
	appendBits(header6, 6)
	appendBits(start, 18)
	appendBits(end, 18)

	var sb strings.Builder
	for i := 0; i < len(bitsArr); i += 6 {
		value := 0
		for j := 0; j < 6; j++ {
			value |= bitsArr[i+j] << j
		}
		sb.WriteByte(base64Alphabet[value])
	}
	return sb.String()
}

// TestReadValueTruncated 读取超长字段时数据不足要返回错误，而不是静默给 0。
func TestReadValueTruncated(t *testing.T) {
	d := newDecoder("A") // 仅 6 位，不足以读 36 位
	if _, err := d.readValue(36); !errors.Is(err, ErrEncoding) {
		t.Fatalf("期望 ErrEncoding，实际 %v", err)
	}
}
