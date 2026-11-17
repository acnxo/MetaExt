package finscope

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"
)

// TestLive 是对真实站点的联调用例，默认跳过。
//
// 用法（等风控解除、或换到不被风控的出口后执行）：
//
//	FINSCOPE_LIVE=1 go test ./component/quote/baidu/finscope -run TestLive -v -count=1
//
// 只对环境变量开关生效，CI 与日常测试都不会发起外部请求。
// 通过后请把 verification.go 里对应接口的 Status 改成 StatusVerified，
// 并在 Note 中写清核对结论。
func TestLive(t *testing.T) {
	if os.Getenv("FINSCOPE_LIVE") == "" {
		t.Skip("真实联调默认跳过：设置 FINSCOPE_LIVE=1 才发起外部请求")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	client := New(Config{})

	// 两个已验证接口：失败即视为回归。
	t.Run("GetConstituents", func(t *testing.T) {
		page, err := client.GetConstituents(ctx, GetConstituentsRequest{
			FinanceType: FinanceTypeBlock,
			Market:      MarketA,
			Code:        "GN2309",
			SortKey:     SortByMarketValue,
			SortType:    SortDesc,
			Limit:       5,
		})
		if err != nil {
			t.Fatalf("调用失败：%v", err)
		}
		t.Logf("北交所上市企业首屏 %d 条，首条 %s %s 市值 %.0f 元",
			len(page.Items), page.Items[0].Code, page.Items[0].Name, page.Items[0].RawData.MarketValue)
	})

	t.Run("Suggest", func(t *testing.T) {
		items, err := client.FindSuggest(ctx, "北证50", ItemTypeIndex)
		if err != nil {
			t.Fatalf("调用失败：%v", err)
		}
		if len(items) == 0 {
			t.Fatal("未匹配到北证50")
		}
		price, _ := items[0].PriceValue()
		ratio, _ := items[0].RatioPercent()
		t.Logf("%s %s 点位 %.2f 涨跌幅 %.2f%%", items[0].Code, items[0].Name, price, ratio)
	})

	// 五个被风控挡住的接口：仍返回风控就记为跳过，通了就把样本打出来。
	blocked := []struct {
		name  string
		check func() (any, error)
	}{
		{"GetQuoteKLine", func() (any, error) {
			return client.GetQuoteKLine(ctx, GetQuoteKLineRequest{Code: "899050", KType: KTypeDay, Count: 5})
		}},
		{"GetQuoteFiveDay", func() (any, error) {
			return client.GetQuoteFiveDay(ctx, GetQuoteFiveDayRequest{Code: "399001"})
		}},
		{"GetQuoteForexMinute", func() (any, error) {
			return client.GetQuoteForexMinute(ctx, GetQuoteForexMinuteRequest{Code: "USDCNY"})
		}},
		{"GetQuoteFuturesMinute", func() (any, error) {
			return client.GetQuoteFuturesMinute(ctx, GetQuoteFuturesMinuteRequest{Code: "AU888"})
		}},
		{"GetQuoteFuturesFiveDay", func() (any, error) {
			return client.GetQuoteFuturesFiveDay(ctx, GetQuoteFuturesFiveDayRequest{Code: "AU888"})
		}},
	}
	for _, item := range blocked {
		t.Run(item.name, func(t *testing.T) {
			result, err := item.check()
			if err != nil {
				if IsRiskControl(err) {
					t.Skipf("仍被站点风控挡住，保持 blocked：%v", err)
				}
				t.Fatalf("非风控错误，需要排查：%v", err)
			}
			// 通了：把原始结构打出来，便于据此补结构化解析并更新 Status。
			raw, _ := json.Marshal(result)
			t.Logf("已取得真实响应，请据此补解析并更新 verification.go 状态：%s", truncate(string(raw), 800))
		})
	}
}

// TestEndpointsStatus 校验状态清单本身：已验证与待复测的接口集合不能写错。
func TestEndpointsStatus(t *testing.T) {
	all := Endpoints()
	if len(all) != 7 {
		t.Fatalf("接口清单应有 7 项，实际 %d", len(all))
	}
	blocked := BlockedEndpoints()
	names := make([]string, 0, len(blocked))
	for _, endpoint := range blocked {
		names = append(names, endpoint.Method)
	}
	want := []string{
		"GetQuoteKLine", "GetQuoteFiveDay", "GetQuoteForexMinute",
		"GetQuoteFuturesMinute", "GetQuoteFuturesFiveDay",
	}
	if len(names) != len(want) {
		t.Fatalf("待复测接口 = %v，期望 %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("待复测接口[%d] = %q，期望 %q", i, names[i], want[i])
		}
	}
	// 每个被阻塞的接口都必须写明复测线索，避免后续无从下手。
	for _, endpoint := range blocked {
		if endpoint.Note == "" {
			t.Errorf("%s 缺少备注", endpoint.Method)
		}
		if endpoint.Status != StatusBlocked {
			t.Errorf("%s 状态 = %q", endpoint.Method, endpoint.Status)
		}
	}
}

// TestBlockedErrorIsRiskControl 确保风控错误可被识别，避免上层误判为空数据。
func TestBlockedErrorIsRiskControl(t *testing.T) {
	err := &APIError{Path: DefaultQuotePath, Code: CodeRiskControl, Message: "hit risk"}
	if !IsRiskControl(err) {
		t.Fatal("风控错误应被 IsRiskControl 识别")
	}
	if errors.Is(err, context.Canceled) {
		t.Error("风控错误不应等同于上下文取消")
	}
}
