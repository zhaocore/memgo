// Package webhook: 记忆事件 → 注册端点的异步投递器 (实现 core/memory.EventSink 端口)。
// 投递为尽力而为: 失败记告警不重试阻塞业务, 注册表为空时零开销。
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/zhao-core/memgo/core/memory"
	"github.com/zhao-core/memgo/server/store"
)

// 事件类型映射: core 事件 → webhook 订阅词表 (参考 custom-categories 同源文档词表)。
var coreToWebhookEvent = map[string]string{
	memory.EventAdd:        "memory_add",
	memory.EventUpdate:     "memory_update",
	memory.EventDelete:     "memory_delete",
	memory.EventCategorize: "memory_categorize",
}

// AllEventTypes 全部可订阅事件 (服务端校验与默认值用)。
func AllEventTypes() []string {
	return []string{"memory_add", "memory_update", "memory_delete", "memory_categorize"}
}

// ValidEventType 校验订阅事件名。
func ValidEventType(t string) bool {
	for _, known := range AllEventTypes() {
		if known == t {
			return true
		}
	}
	return false
}

// Dispatcher 事件投递器。
type Dispatcher struct {
	store  *store.Store
	client *http.Client
	// listHooks 端点列表来源 (测试注入; nil 时用 store.ListWebhooks)。
	listHooks func(ctx context.Context) ([]store.Webhook, error)
	// AttemptTimeout 单次 HTTP 尝试超时 (测试可覆盖)。
	AttemptTimeout time.Duration
}

// NewDispatcher 构造 (client 为 nil 时用默认 10s 超时)。
func NewDispatcher(st *store.Store) *Dispatcher {
	return &Dispatcher{
		store:          st,
		client:         &http.Client{Timeout: 10 * time.Second},
		AttemptTimeout: 10 * time.Second,
	}
}

// Emit 实现 memory.EventSink: 异步投递, 不阻塞记忆写路径。
func (d *Dispatcher) Emit(ev memory.Event) {
	webhookType, ok := coreToWebhookEvent[ev.Type]
	if !ok {
		return
	}
	payload := buildPayload(ev)
	go d.deliver(webhookType, payload)
}

// buildPayload 对齐文档: 记忆事件含 ID/内容/事件类型; CATEGORIZE 含记忆 ID/类型/命中分类。
func buildPayload(ev memory.Event) map[string]any {
	if ev.Type == memory.EventCategorize {
		return map[string]any{
			"event":      "CATEGORIZE",
			"memory_id":  ev.MemoryID,
			"categories": []string{ev.Category},
		}
	}
	return map[string]any{
		"event":     ev.Type,
		"memory_id": ev.MemoryID,
		"data":      ev.Data,
	}
}

// deliver 列出匹配订阅的端点逐个投递 (尽力而为: 失败仅告警)。
func (d *Dispatcher) deliver(webhookType string, payload map[string]any) {
	ctx, cancel := context.WithTimeout(context.Background(), d.AttemptTimeout)
	defer cancel()
	list := d.listHooks
	if list == nil {
		list = d.store.ListWebhooks
	}
	hooks, err := list(ctx)
	if err != nil {
		log.Printf("webhook 投递前列表失败: %v", err)
		return
	}
	for _, hook := range hooks {
		if !subscribed(hook.EventTypes, webhookType) {
			continue
		}
		if err := d.post(hook.URL, webhookType, payload); err != nil {
			log.Printf("webhook 投递失败 (url=%s event=%s): %v", hook.URL, webhookType, err)
		}
	}
}

// subscribed 判断端点是否订阅该事件 (未配置 event_types = 订阅全部)。
func subscribed(types []string, event string) bool {
	if len(types) == 0 {
		return true
	}
	for _, t := range types {
		if t == event {
			return true
		}
	}
	return false
}

// post 单次投递 (独立超时; 失败返回错误由调用方告警)。
func (d *Dispatcher) post(url, webhookType string, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook payload 序列化失败: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), d.AttemptTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook 请求构造失败 (url=%s): %w", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Memgo-Event", webhookType)
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook 请求失败 (url=%s): %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook 端点返回非 2xx (url=%s status=%d)", url, resp.StatusCode)
	}
	return nil
}
