package cnatradecalendar

import (
	"strings"
	"testing"
	"time"
)

// layoutWidth 是每个日期单元的固定宽度，日期不带连接符，"20060102" 为 8 个字符。
const layoutWidth = len("20060102")

// TestTradeCalendarWeeklyLayout 把整份日历按「一行一周、周一至周五五列」的格式
// 打印到测试日志，缺失的交易日用等宽空白占位，保证各列纵向对齐。
//
// 用 -v 运行可看到完整表格：
//
//	go test ./component/quote/sina/finance/cnatradecalendar -run TestTradeCalendarWeeklyLayout -v
func TestTradeCalendarWeeklyLayout(t *testing.T) {
	cal := mustParse(t)

	// 以所在周的周一为键分组，行内按下标 0~4 对应周一至周五。
	rows := make(map[time.Time]*[5]string)
	weeks := make([]time.Time, 0, cal.Len()/5+1)
	filled := 0

	for _, d := range cal.Dates {
		weekday := (int(d.Weekday()) + 6) % 7 // 周一为 0，周日为 6
		if weekday > 4 {
			t.Fatalf("交易日历中不应出现周末日期：%s", d.Format("20060102"))
		}
		monday := d.AddDate(0, 0, -weekday)
		row, ok := rows[monday]
		if !ok {
			row = &[5]string{}
			rows[monday] = row
			weeks = append(weeks, monday)
		}
		if row[weekday] != "" {
			t.Fatalf("%s 在同一周内重复出现", d.Format("20060102"))
		}
		row[weekday] = d.Format("20060102")
		filled++
	}

	if filled != cal.Len() {
		t.Fatalf("填入单元数 %d 与交易日数量 %d 不一致", filled, cal.Len())
	}
	var sb strings.Builder
	blank := strings.Repeat(" ", layoutWidth)
	for i, monday := range weeks {
		row := rows[monday]
		for col := 0; col < 5; col++ {
			if col > 0 {
				sb.WriteString("  ")
			}
			if row[col] == "" {
				sb.WriteString(blank)
				continue
			}
			sb.WriteString(row[col])
		}
		sb.WriteString("\n")
		if i == 0 {
			// 首周用于确认起始日是周三：周一、周二应为空位。
			if row[0] != "" || row[1] != "" || row[2] != "19901219" {
				t.Fatalf("首周应为「空 空 1990-12-19 ...」，实际 %v", *row)
			}
		}
		if i == len(weeks)-1 {
			// 末日 2026-12-31 是周四，周五应为空位。
			if row[3] != "20261231" || row[4] != "" {
				t.Fatalf("末周应以 2026-12-31 结尾且周五为空，实际 %v", *row)
			}
		}
	}

	// 每周一行，行数必须与周数一致，确保输出没有被截断或漏行。
	if lines := strings.Count(sb.String(), "\n"); lines != len(weeks) {
		t.Fatalf("输出 %d 行，期望每周一行共 %d 行", lines, len(weeks))
	}

	t.Logf("交易日历按周输出：共 %d 周、%d 个交易日，列顺序为周一至周五，缺位留空",
		len(weeks), cal.Len())
	t.Log("\n" + sb.String())
}
