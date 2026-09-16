package api

// export 纯函数测试: 条目映射 (可选键按需出现) 与 CSV 转义 (逗号/引号/换行)。

import "testing"

// TestExportItemFromPayload: data/scope/时间落位; 可选键仅在非空时出现。
func TestExportItemFromPayload(t *testing.T) {
	full := exportItemFromPayload("m1", map[string]any{
		"data": "I like tea", "user_id": "alice", "agent_id": "bot",
		"run_id": "r1", "category": "food", "created_at": "2026-09-16T00:00:00.000000+00:00",
		"hash": "ignored", "text_lemmatized": "ignored",
	})
	if full["content"] != "I like tea" || full["user_id"] != "alice" || full["id"] != "m1" {
		t.Errorf("基础字段映射不符: %v", full)
	}
	if full["agent_id"] != "bot" || full["run_id"] != "r1" || full["category"] != "food" {
		t.Errorf("可选字段应保留: %v", full)
	}
	if _, has := full["hash"]; has {
		t.Errorf("metadata 内部字段不应导出")
	}
	minimal := exportItemFromPayload("m2", map[string]any{"data": "x", "created_at": "t"})
	for _, opt := range []string{"agent_id", "run_id", "category"} {
		if _, has := minimal[opt]; has {
			t.Errorf("空可选键 %s 不应出现", opt)
		}
	}
}

// TestBuildExportCSV: 表头行 + content 含逗号/引号/换行时的引号转义 + nil → 空串。
func TestBuildExportCSV(t *testing.T) {
	csvOut := buildExportCSV([]map[string]any{
		{
			"id": "m1", "content": "likes \"dark\" mode, a lot\nreally", "user_id": "alice",
			"agent_id": nil, "run_id": nil, "category": nil, "created_at": "t0",
		},
	})
	// encoding/csv 最小引用: 仅含逗号/引号/换行的字段加引号。
	want := "id,content,user_id,agent_id,run_id,category,created_at\n" +
		`m1,"likes ""dark"" mode, a lot` + "\n" + `really",alice,,,,t0` + "\n"
	if csvOut != want {
		t.Errorf("CSV 输出不符:\n got %q\nwant %q", csvOut, want)
	}
}
