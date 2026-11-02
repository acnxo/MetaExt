package tradecalendar

import (
	"sort"
	"time"
)

// TradeCalendar 是一段 A 股交易日历。
//
// Dates 按日期升序排列，每项都是 UTC 零点，只表示日历日期。字段导出便于调用方
// 直接遍历，但不应修改顺序，否则二分查询会失效。
type TradeCalendar struct {
	Dates []time.Time
}

// Len 返回交易日数量。
func (c *TradeCalendar) Len() int {
	if c == nil {
		return 0
	}
	return len(c.Dates)
}

// First 返回最早的一个交易日，日历为空时第二个返回值为 false。
func (c *TradeCalendar) First() (time.Time, bool) {
	if c.Len() == 0 {
		return time.Time{}, false
	}
	return c.Dates[0], true
}

// Last 返回最晚的一个交易日，日历为空时第二个返回值为 false。
func (c *TradeCalendar) Last() (time.Time, bool) {
	if c.Len() == 0 {
		return time.Time{}, false
	}
	return c.Dates[len(c.Dates)-1], true
}

// IsTradeDay 判断给定日期是否为交易日。
//
// 只比较入参显示的年月日，不做时区换算：同一时刻用不同时区表示时，结果可能不同。
// 非交易日包括周末、法定节假日以及数据源未收录的日期。
func (c *TradeCalendar) IsTradeDay(t time.Time) bool {
	if c.Len() == 0 {
		return false
	}
	year, month, day := t.Date()
	target := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	idx := sort.Search(len(c.Dates), func(i int) bool {
		return !c.Dates[i].Before(target)
	})
	return idx < len(c.Dates) && c.Dates[idx].Equal(target)
}
