package middleware

import (
	"crypto/rand"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/zhao-core/memgo/server/store"
)

// newUUID RFC4122 v4 (日志行主键)。
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// 跳过清单对齐 main.py SKIPPED_*。
var skippedPaths = map[string]bool{"/api/health": true, "/docs": true, "/redoc": true, "/openapi.json": true}

var skippedPrefixes = []string{"/requests"}

// ShouldLog 对齐 _should_log_request。
func ShouldLog(method, path string) bool {
	if method == "OPTIONS" {
		return false
	}
	if skippedPaths[path] {
		return false
	}
	for _, p := range skippedPrefixes {
		if strings.HasPrefix(path, p) {
			return false
		}
	}
	return true
}

// LogWriter 有界后台批量落库 (计划 P2: 旁路 writer 替代 run_in_executor)。
// 失败仅记日志, 不影响响应 (对齐 _persist_request_log 语义)。
type LogWriter struct {
	ch   chan store.RequestLog
	wg   sync.WaitGroup
	stop chan struct{}
}

// NewLogWriter 启动后台 goroutine。
func NewLogWriter(db *store.Store) *LogWriter {
	w := &LogWriter{ch: make(chan store.RequestLog, 1024), stop: make(chan struct{})}
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		for {
			select {
			case entry := <-w.ch:
				// 尽力排空一小批再写, 单条失败仅日志
				ctx, cancel := contextWithTimeout()
				_ = entry
				if err := db.InsertRequestLog(ctx, &entry); err != nil {
					log.Printf("request log 落库失败: %v", err)
				}
				cancel()
			case <-w.stop:
				return
			}
		}
	}()
	return w
}

// Emit 投递 (满则丢弃, 不阻塞响应); id 在此生成。
func (w *LogWriter) Emit(method, path string, status int, latencyMS float64, authType string) {
	entry := store.RequestLog{
		ID: newUUID(), Method: method, Path: path, StatusCode: status,
		LatencyMS: latencyMS, AuthType: authType,
	}
	select {
	case w.ch <- entry:
	default:
	}
}

// Close 停止后台写。
func (w *LogWriter) Close() {
	close(w.stop)
	w.wg.Wait()
}
