package tradecalendar

import (
	"testing"
	"time"
)

// newCalendar 用给定日期构造测试用日历。
func newCalendar(t *testing.T, days ...string) *TradeCalendar {
	t.Helper()
	dates := make([]time.Time, 0, len(days))
	for _, day := range days {
		dates = append(dates, parseDay(t, day))
	}
	return &TradeCalendar{Dates: dates}
}

// TestCalendarEmpty 空日历与 nil 日历的行为一致，不 panic。
func TestCalendarEmpty(t *testing.T) {
	for name, cal := range map[string]*TradeCalendar{"nil": nil, "空": {}} {
		if cal.Len() != 0 {
			t.Errorf("%s：Len 期望 0，实际 %d", name, cal.Len())
		}
		if _, ok := cal.First(); ok {
			t.Errorf("%s：First 应返回 false", name)
		}
		if _, ok := cal.Last(); ok {
			t.Errorf("%s：Last 应返回 false", name)
		}
		if cal.IsTradeDay(parseDay(t, "2026-11-02")) {
			t.Errorf("%s：不应判为交易日", name)
		}
	}
}

// TestCalendarFirstLast 首尾日期取自升序列表的两端。
func TestCalendarFirstLast(t *testing.T) {
	cal := newCalendar(t, "2026-11-02", "2026-11-03", "2026-11-04")
	first, ok := cal.First()
	if !ok || first.Format("2006-01-02") != "2026-11-02" {
		t.Fatalf("First 不符：%v %v", first, ok)
	}
	last, ok := cal.Last()
	if !ok || last.Format("2006-01-02") != "2026-11-04" {
		t.Fatalf("Last 不符：%v %v", last, ok)
	}
}

// TestCalendarIsTradeDay 命中、未命中与边界日期都要判断正确。
func TestCalendarIsTradeDay(t *testing.T) {
	cal := newCalendar(t, "2026-11-02", "2026-11-03", "2026-11-05")
	cases := map[string]bool{
		"2026-11-02": true,  // 首项
		"2026-11-03": true,  // 中间
		"2026-11-05": true,  // 末项
		"2026-11-04": false, // 中间缺口
		"2026-11-01": false, // 早于首项
		"2026-11-06": false, // 晚于末项
	}
	for day, want := range cases {
		if got := cal.IsTradeDay(parseDay(t, day)); got != want {
			t.Errorf("%s 期望 %v，实际 %v", day, want, got)
		}
	}
}

// TestCalendarIsTradeDayTimeOfDay 只比较年月日，忽略时分秒。
func TestCalendarIsTradeDayTimeOfDay(t *testing.T) {
	cal := newCalendar(t, "2026-11-02")
	withTime := time.Date(2026, 11, 2, 15, 30, 45, 123, time.UTC)
	if !cal.IsTradeDay(withTime) {
		t.Fatal("同一天的任意时刻都应判为交易日")
	}
}

// TestCalendarIsTradeDayZone 按入参显示的年月日判断，不做时区换算。
func TestCalendarIsTradeDayZone(t *testing.T) {
	cal := newCalendar(t, "2026-11-02")
	shanghai := time.FixedZone("CST", 8*3600)
	// 同一瞬间：UTC 的 11-01 20:00 等于东八区的 11-02 04:00。
	instant := time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)
	if cal.IsTradeDay(instant) {
		t.Error("按 UTC 表示应为 11-01，不是交易日")
	}
	if !cal.IsTradeDay(instant.In(shanghai)) {
		t.Error("按东八区表示应为 11-02，是交易日")
	}
}
