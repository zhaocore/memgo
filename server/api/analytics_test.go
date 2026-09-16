package api

// analytics 纯函数测试: 补零桶序列与一位小数舍入。

import (
	"testing"
	"time"
)

// TestFillDailyBuckets: 连续 days 天、今天含内、缺数据日补零、乱序输入无害。
func TestFillDailyBuckets(t *testing.T) {
	today := time.Date(2026, 9, 16, 10, 30, 0, 0, time.UTC)
	got := fillDailyBuckets(3, map[string]int{"2026-09-15": 4, "2026-09-16": 7}, today)
	if len(got) != 3 {
		t.Fatalf("应输出 3 个桶, 得 %d", len(got))
	}
	wantDates := []string{"2026-09-14", "2026-09-15", "2026-09-16"}
	wantCounts := []int{0, 4, 7}
	for i, bucket := range got {
		if bucket["date"] != wantDates[i] || bucket["count"] != wantCounts[i] {
			t.Errorf("桶 %d 应为 %s=%d, 得 %v", i, wantDates[i], wantCounts[i], bucket)
		}
	}
}

// TestFillDailyBucketsSingleDay: days=1 只含今天。
func TestFillDailyBucketsSingleDay(t *testing.T) {
	today := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	got := fillDailyBuckets(1, nil, today)
	if len(got) != 1 || got[0]["date"] != "2026-09-16" || got[0]["count"] != 0 {
		t.Errorf("单日窗口应为今天且补零: %v", got)
	}
}

// TestRound1: 一位小数舍入 (显示口径)。
func TestRound1(t *testing.T) {
	cases := map[float64]float64{99.76: 99.8, 142.44: 142.4, 0: 0, 100.0: 100}
	for in, want := range cases {
		if got := round1(in); got != want {
			t.Errorf("round1(%v) 应为 %v, 得 %v", in, want, got)
		}
	}
}
