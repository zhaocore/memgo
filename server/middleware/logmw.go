package middleware

import (
	"net/http"
	"time"

	"github.com/zhao-core/memgo/server/auth"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// LogRequest 包装: 计时 + 跳过清单 + 后台落库。
func LogRequest(writer *LogWriter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: 500}
		start := time.Now()
		next.ServeHTTP(rec, r)
		if !ShouldLog(r.Method, r.URL.Path) {
			return
		}
		latencyMS := float64(time.Since(start).Microseconds()) / 1000.0
		authType := "none"
		if c := auth.FromContext(r.Context()); c != nil {
			authType = c.AuthType
		}
		writer.Emit(r.Method, r.URL.Path, rec.status, latencyMS, authType)
	})
}
