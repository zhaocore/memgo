package memory

// 事件端口测试: ADD/UPDATE/DELETE/CATEGORIZE 各落点 emit, nil sink 零影响。

import (
	"strings"
	"testing"

	"github.com/zhao-core/memgo/core/config"
)

// captureSink 捕获事件 (测试桩)。
type captureSink struct{ events []Event }

func (c *captureSink) Emit(ev Event) { c.events = append(c.events, ev) }

func (c *captureSink) types() []string {
	out := make([]string, 0, len(c.events))
	for _, ev := range c.events {
		out = append(out, ev.Type)
	}
	return out
}

// TestAddEmitsAddAndCategorize: infer=true + 目录生效 → 每条新记忆 ADD + CATEGORIZE。
func TestAddEmitsAddAndCategorize(t *testing.T) {
	sink := &captureSink{}
	llm := &fakeLLM{queue: []string{extractionResponse, categoryClassResp("food", "travel")}}
	m := newCategoryTestMemory(t, llm, &config.MemoryConfig{
		Version: "v1.1",
		VectorStore: config.ProviderConfig{Provider: "pgvector", Config: map[string]any{
			"collection_name": "memories",
		}},
		LLM:      config.ProviderConfig{Provider: "openai"},
		Embedder: config.ProviderConfig{Provider: "openai"},
		CustomCategories: []config.Category{
			{Name: "food", Description: "dining"},
			{Name: "travel", Description: "trips"},
		},
	})
	m.SetEventSink(sink)
	results, err := m.Add([]map[string]any{{"role": "user", "content": "I love ramen"}}, AddParams{UserID: "alice", Infer: true})
	if err != nil {
		t.Fatalf("add 失败: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("期望 2 条, 得 %d", len(results))
	}
	got := sink.types()
	if len(got) != 4 || got[0] != EventAdd || got[1] != EventCategorize || got[2] != EventAdd || got[3] != EventCategorize {
		t.Fatalf("事件序列应为 ADD,CATEGORIZE,ADD,CATEGORIZE: %v", got)
	}
	if sink.events[0].Data == "" || sink.events[0].MemoryID != results[0]["id"] {
		t.Errorf("ADD 事件应含记忆 id 与内容: %+v", sink.events[0])
	}
	if sink.events[1].Category != "food" {
		t.Errorf("CATEGORIZE 事件应含命中分类: %+v", sink.events[1])
	}
}

// TestAddInferFalseEmitsAddOnly: infer=false / 功能关闭 → 仅 ADD, 无 CATEGORIZE。
func TestAddInferFalseEmitsAddOnly(t *testing.T) {
	sink := &captureSink{}
	llm := &fakeLLM{response: extractionResponse}
	m := newCategoryTestMemory(t, llm, nil)
	m.SetEventSink(sink)
	_, err := m.Add([]map[string]any{
		{"role": "user", "content": "I live in Berlin"},
		{"role": "assistant", "content": "Noted"},
	}, AddParams{UserID: "alice", Infer: false})
	if err != nil {
		t.Fatalf("add 失败: %v", err)
	}
	for i, ev := range sink.events {
		if ev.Type != EventAdd {
			t.Errorf("infer=false 只应有 ADD, 第 %d 个事件为 %s", i, ev.Type)
		}
		if strings.Contains(ev.Data, "must be skipped") {
			t.Errorf("system 消息不应产生事件")
		}
	}
	if len(sink.events) != 2 {
		t.Fatalf("期望 2 个 ADD 事件, 得 %d", len(sink.events))
	}
}

// TestUpdateAndDeleteEmit: Update → UPDATE(新文本), Delete → DELETE(删除时当前文本)。
func TestUpdateAndDeleteEmit(t *testing.T) {
	sink := &captureSink{}
	llm := &fakeLLM{response: extractionResponse}
	m := newCategoryTestMemory(t, llm, nil)
	m.SetEventSink(sink)
	results, err := m.Add([]map[string]any{{"role": "user", "content": "I live in Berlin"}},
		AddParams{UserID: "alice", Infer: false})
	if err != nil {
		t.Fatalf("add 失败: %v", err)
	}
	id := results[0]["id"].(string)
	if _, err := m.Update(UpdateParams{MemoryID: id, Text: ptrString("I live in Munich")}); err != nil {
		t.Fatalf("update 失败: %v", err)
	}
	if _, err := m.Delete(id); err != nil {
		t.Fatalf("delete 失败: %v", err)
	}
	if len(sink.events) != 3 {
		t.Fatalf("期望 ADD+UPDATE+DELETE 三个事件, 得 %v", sink.types())
	}
	if sink.events[1].Type != EventUpdate || sink.events[1].Data != "I live in Munich" {
		t.Errorf("UPDATE 事件应携带新文本: %+v", sink.events[1])
	}
	if sink.events[2].Type != EventDelete || sink.events[2].Data != "I live in Munich" {
		// DELETE 携带删除时的当前文本 (Update 已改写为 Munich)。
		t.Errorf("DELETE 事件应携带删除前文本: %+v", sink.events[2])
	}
}

func ptrString(s string) *string { return &s }
