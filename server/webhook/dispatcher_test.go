package webhook

// 投递器测试: payload 形状、订阅过滤、真实 httptest 投递 (X-Memgo-Event 头)。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/zhao-core/memgo/core/memory"
	"github.com/zhao-core/memgo/server/store"
)

func testDispatcher(hooks []store.Webhook) *Dispatcher {
	return &Dispatcher{
		listHooks:      func(context.Context) ([]store.Webhook, error) { return hooks, nil },
		client:         &http.Client{Timeout: 2 * time.Second},
		AttemptTimeout: 2 * time.Second,
	}
}

// TestSubscribedFilter: 空 event_types = 全订阅; 按名单过滤。
func TestSubscribedFilter(t *testing.T) {
	if !subscribed(nil, "memory_add") {
		t.Errorf("未配置 event_types 应订阅全部")
	}
	onlyAdd := []string{"memory_add"}
	if !subscribed(onlyAdd, "memory_add") || subscribed(onlyAdd, "memory_delete") {
		t.Errorf("订阅过滤不符: %v", onlyAdd)
	}
}

// TestBuildPayloadShape: 记忆事件 vs CATEGORIZE 事件 payload。
func TestBuildPayloadShape(t *testing.T) {
	mem := buildPayload(memory.Event{Type: memory.EventAdd, MemoryID: "m1", Data: "I like tea"})
	if mem["event"] != "ADD" || mem["memory_id"] != "m1" || mem["data"] != "I like tea" {
		t.Errorf("记忆事件 payload 不符: %v", mem)
	}
	cat := buildPayload(memory.Event{Type: memory.EventCategorize, MemoryID: "m1", Category: "food"})
	if cat["event"] != "CATEGORIZE" || cat["categories"].([]string)[0] != "food" {
		t.Errorf("CATEGORIZE payload 不符: %v", cat)
	}
	if _, has := cat["data"]; has {
		t.Errorf("CATEGORIZE 不应携带 data")
	}
}

// TestEmitDeliversToSubscribed: httptest 实打 — 订阅匹配端点收到投递, 不匹配的收不到。
func TestEmitDeliversToSubscribed(t *testing.T) {
	var mu sync.Mutex
	received := map[string]map[string]any{}
	got := make(chan string, 4)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		event := r.Header.Get("X-Memgo-Event")
		mu.Lock()
		received[event] = payload
		mu.Unlock()
		got <- event
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	// hook1 只订阅 memory_add; hook2 订阅无关事件。
	d := testDispatcher([]store.Webhook{
		{ID: "w1", URL: target.URL, EventTypes: []string{"memory_add"}},
		{ID: "w2", URL: target.URL, EventTypes: []string{"memory_delete"}},
	})
	d.Emit(memory.Event{Type: memory.EventAdd, MemoryID: "m1", Data: "I like tea"})
	d.Emit(memory.Event{Type: memory.EventCategorize, MemoryID: "m1", Category: "food"})

	select {
	case ev := <-got:
		mu.Lock()
		payload := received[ev]
		mu.Unlock()
		if ev != "memory_add" {
			t.Fatalf("应只投递 memory_add, 收到 %s", ev)
		}
		if payload["memory_id"] != "m1" {
			t.Errorf("payload 应含 memory_id: %v", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("投递超时, 未收到任何请求")
	}
	// 给第二个 (被过滤的) 事件留出误投递窗口。
	time.Sleep(100 * time.Millisecond)
	select {
	case ev := <-got:
		t.Errorf("CATEGORIZE 不应投递到只订阅 memory_add/delete 的端点, 收到 %s", ev)
	default:
	}
}

// TestPostNon2xxIsError: 端点 500 → 返回错误 (调用方告警)。
func TestPostNon2xxIsError(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer target.Close()
	d := testDispatcher(nil)
	if err := d.post(target.URL, "memory_add", map[string]any{"event": "ADD"}); err == nil {
		t.Errorf("非 2xx 必须返回错误")
	}
}
