package cnatradecalendar

import (
	"context"
	"sort"
	"time"
)

// shanghaiOpenDate 是沪市交易日历的起点，即上交所开市首日。
//
// 数据源 klc_td_sh.txt 以该日为起点；深市资源 klc_td_sz.txt 的起点更晚，
// 见 shenzhenOpenDate。
const (
	shanghaiOpenDate = "1990-12-19"
	shenzhenOpenDate = "1991-01-02"
)

// sourceGaps 是数据源未收录、但当天确有交易的日期。
//
// 1992-05-04 是周一，正常应有交易，站点日历文件中缺失；补齐后与其它来源一致。
var sourceGaps = []time.Time{
	time.Date(1992, 5, 4, 0, 0, 0, 0, time.UTC),
}

// ShanghaiOpenDay 返回沪市交易日历的起点，即上交所开市首日（1990-12-19）。
//
// 本包基于沪市资源：GetTradeCalendar 与 GetCNATradeCalendar 返回的日历都从该日开始。
func ShanghaiOpenDay() time.Time {
	day, _ := time.Parse("2006-01-02", shanghaiOpenDate)
	return day
}

// ShenzhenOpenDay 返回深市交易日历的起点（1991-01-02）。
//
// 深交所开市晚于上交所，两市在 1990-12-19 至 1991-01-01 之间的交易日并不相同。
// 判断深市标的时，不能直接用本包返回的沪市日历，需要按该差异自行处理。
func ShenzhenOpenDay() time.Time {
	day, _ := time.Parse("2006-01-02", shenzhenOpenDate)
	return day
}

// GetCNATradeCalendar 拉取 A 股交易日历，并补齐数据源已知缺失的交易日。
//
// 与 GetTradeCalendar 的关系：后者如实返回数据源内容，可能带缺口；本方法在其基础上
// 补上 sourceGaps，得到完整的沪市日历，适合直接用于判断「某天是否为交易日」。
//
// 对外建议优先使用本方法。注意该日历以上交所为准：深交所开市晚于沪市起点，
// 两市的交易日集合在早期并不完全一致，判断深市标的时需按 ShenzhenOpenDay 的差异
// 自行处理。
func (c *Client) GetCNATradeCalendar(ctx context.Context) (*TradeCalendar, error) {
	cal, err := c.GetTradeCalendar(ctx)
	if err != nil {
		return nil, err
	}
	return withSourceGaps(cal), nil
}

// withSourceGaps 返回补齐缺口后的新日历，不修改入参。
//
// 若日历中已存在该日期则不重复插入；补齐后仍保持升序。
func withSourceGaps(cal *TradeCalendar) *TradeCalendar {
	if cal == nil {
		cal = &TradeCalendar{}
	}
	dates := make([]time.Time, len(cal.Dates), len(cal.Dates)+len(sourceGaps))
	copy(dates, cal.Dates)

	for _, day := range sourceGaps {
		idx := sort.Search(len(dates), func(i int) bool {
			return !dates[i].Before(day)
		})
		if idx < len(dates) && dates[idx].Equal(day) {
			continue
		}
		dates = append(dates, time.Time{})
		copy(dates[idx+1:], dates[idx:])
		dates[idx] = day
	}
	return &TradeCalendar{Dates: dates}
}
