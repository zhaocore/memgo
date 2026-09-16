package api

import (
	"math"
	"net/http"
	"time"

	"github.com/zhao-core/memgo/server/auth"
	"github.com/zhao-core/memgo/server/store"
)

// getAnalytics GET /analytics (admin; memgo 扩展: 记忆操作聚合, 数据源 request_logs)。
// 口径 = 记忆操作路径 (/memories*、/search); query 参数 days 窗口天数 (默认 7, 1-90)。
func (s *Server) getAnalytics(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	days, ok := validateQueryInt(w, r, "days", 7, 1, 90, true, true)
	if !ok {
		return
	}
	// 窗口起点 = 今天(UTC) 往前 days-1 天的 0 点 (今天含内)。
	today := time.Now().UTC().Truncate(24 * time.Hour)
	since := today.AddDate(0, 0, -(days - 1))
	summary, err := s.store.GetAnalyticsSummary(r.Context(), since)
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	counts, err := s.store.ListDailyOpCounts(r.Context(), since)
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	byDate := make(map[string]int, len(counts))
	for _, d := range counts {
		byDate[d.Date] = d.Count
	}
	series := fillDailyBuckets(days, byDate, today)
	writeJSON(w, http.StatusOK, map[string]any{
		"total_operations":     summary.TotalOps,
		"avg_latency_ms":       round1(summary.AvgLatencyMS),
		"success_rate":         round1(summary.SuccessPct),
		"operations_over_time": series,
	})
}

// fillDailyBuckets 补零成连续 days 天序列 (今天含内, 纯函数便于单测)。
func fillDailyBuckets(days int, byDate map[string]int, today time.Time) []map[string]any {
	out := make([]map[string]any, 0, days)
	for i := days - 1; i >= 0; i-- {
		day := today.AddDate(0, 0, -i).Format("2006-01-02")
		out = append(out, map[string]any{"date": day, "count": byDate[day]})
	}
	return out
}

// round1 保留一位小数 (dashboard 显示口径)。
func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
