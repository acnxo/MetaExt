package cnatradecalendar

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestGetCNATradeCalendar 补齐缺口后应比原始日历多一天，且包含 1992-05-04。
func TestGetCNATradeCalendar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(calendarBody))
	}))
	defer server.Close()

	client := newTestClient(t, server, Config{})
	raw, err := client.GetTradeCalendar(context.Background())
	if err != nil {
		t.Fatalf("拉取原始日历失败: %v", err)
	}
	filled, err := client.GetCNATradeCalendar(context.Background())
	if err != nil {
		t.Fatalf("拉取补齐日历失败: %v", err)
	}

	if raw.IsTradeDay(parseDay(t, "1992-05-04")) {
		t.Fatal("原始日历不应包含 1992-05-04，用例前提不成立")
	}
	if !filled.IsTradeDay(parseDay(t, "1992-05-04")) {
		t.Fatal("补齐日历应包含 1992-05-04")
	}
	if got, want := filled.Len(), raw.Len()+1; got != want {
		t.Fatalf("补齐后数量期望 %d，实际 %d", want, got)
	}
	if raw.Len() != 8796 {
		t.Fatalf("原始日历数量期望 8796，实际 %d", raw.Len())
	}

	// 保持严格升序，且插入位置正确：1992-05-01 之后、1992-05-05 之前。
	for i := 1; i < filled.Len(); i++ {
		if !filled.Dates[i].After(filled.Dates[i-1]) {
			t.Fatalf("第 %d 项未严格递增", i)
		}
	}
	// 插入位置应为 1992-04-30 与 1992-05-05 之间。
	pos := -1
	for i, d := range filled.Dates {
		if d.Format("2006-01-02") == "1992-05-04" {
			pos = i
			break
		}
	}
	if pos <= 0 || pos >= filled.Len()-1 {
		t.Fatalf("1992-05-04 位置异常: %d", pos)
	}
	if got := filled.Dates[pos-1].Format("2006-01-02"); got != "1992-04-30" {
		t.Errorf("前一项期望 1992-04-30，实际 %s", got)
	}
	if got := filled.Dates[pos+1].Format("2006-01-02"); got != "1992-05-05" {
		t.Errorf("后一项期望 1992-05-05，实际 %s", got)
	}
	if filled.IsTradeDay(parseDay(t, "1992-05-02")) {
		t.Error("1992-05-02 是周六，不应为交易日")
	}
}

// TestWithSourceGaps 补齐逻辑不改入参、不重复插入、位置正确。
func TestWithSourceGaps(t *testing.T) {
	original := newCalendar(t, "1992-04-30", "1992-05-05", "1992-05-06")
	filled := withSourceGaps(original)

	if filled.Len() != 4 {
		t.Fatalf("补齐后应有 4 天，实际 %d", filled.Len())
	}
	want := []string{"1992-04-30", "1992-05-04", "1992-05-05", "1992-05-06"}
	for i, day := range want {
		if got := filled.Dates[i].Format("2006-01-02"); got != day {
			t.Errorf("第 %d 项期望 %s，实际 %s", i, day, got)
		}
	}
	if original.Len() != 3 {
		t.Fatalf("入参不应被修改，实际长度 %d", original.Len())
	}
	if original.IsTradeDay(parseDay(t, "1992-05-04")) {
		t.Fatal("入参不应被插入缺口日期")
	}
}

// TestWithSourceGapsAlreadyPresent 日历已含该日期时不重复插入。
func TestWithSourceGapsAlreadyPresent(t *testing.T) {
	original := newCalendar(t, "1992-05-04", "1992-05-05")
	filled := withSourceGaps(original)
	if filled.Len() != 2 {
		t.Fatalf("不应重复插入，实际 %d 天", filled.Len())
	}
	if got := filled.Dates[0].Format("2006-01-02"); got != "1992-05-04" {
		t.Fatalf("首项应为 1992-05-04，实际 %s", got)
	}
}

// TestWithSourceGapsEmpty 空日历与 nil 日历补齐后仍只含缺口日期。
func TestWithSourceGapsEmpty(t *testing.T) {
	for name, cal := range map[string]*TradeCalendar{"nil": nil, "空": {}} {
		filled := withSourceGaps(cal)
		if filled.Len() != len(sourceGaps) {
			t.Errorf("%s：期望补齐为 %d 天，实际 %d", name, len(sourceGaps), filled.Len())
		}
	}
}

// TestExchangeOpenDays 两市开市日应与其真实日期一致，且深市晚于沪市。
func TestExchangeOpenDays(t *testing.T) {
	shanghai, shenzhen := ShanghaiOpenDay(), ShenzhenOpenDay()
	if got := shanghai.Format("2006-01-02"); got != "1990-12-19" {
		t.Errorf("沪市开市日期望 1990-12-19，实际 %s", got)
	}
	if got := shenzhen.Format("2006-01-02"); got != "1991-01-02" {
		t.Errorf("深市开市日期望 1991-01-02，实际 %s", got)
	}
	if !shenzhen.After(shanghai) {
		t.Error("深市开市日应晚于沪市")
	}
	if shanghai.Location() != time.UTC || shenzhen.Location() != time.UTC {
		t.Error("开市日应为 UTC 零点")
	}
}

// TestGetCNATradeCalendarError 底层拉取失败时原样返回错误，不做补齐。
func TestGetCNATradeCalendarError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := newTestClient(t, server, Config{}).GetCNATradeCalendar(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusBadGateway {
		t.Fatalf("期望 HTTPStatus=502 的错误，实际 %v", err)
	}
}
