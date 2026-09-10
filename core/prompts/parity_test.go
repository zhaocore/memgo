package prompts

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// parityManifest 列出本包应与 Python 版逐字节一致的常量名 (gen_prompts.py 生成)。
var parityManifest []string

func init() {
	if err := json.Unmarshal([]byte(parityManifestJSON), &parityManifest); err != nil {
		panic(err)
	}
}

// TestPromptParityWithPython: prompts 与 Python 版 diff = 0 (计划 P1 验收项)。
// 依赖 mem0 源码位置, 环境变量 MEM0_SOURCE 指向 mem0 仓库根; 未设则跳过。
func TestPromptParityWithPython(t *testing.T) {
	repo := os.Getenv("MEM0_SOURCE")
	if repo == "" {
		t.Skip("MEM0_SOURCE 未设, 跳过跨语言 parity 校验")
	}
	src := filepath.Join(repo, "mem0", "configs", "prompts.py")
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("prompts.py 不存在: %s", src)
	}
	out, err := exec.Command("python3", filepath.Join("..", "..", "tools", "dump_prompts.py"), src).Output()
	if err != nil {
		t.Fatalf("dump_prompts.py 执行失败: %v", err)
	}
	var pyConsts map[string]string
	if err := json.Unmarshal(out, &pyConsts); err != nil {
		t.Fatalf("解析 dump 输出失败: %v", err)
	}
	goConsts := allConstants()
	for _, name := range parityManifest {
		pyText, ok := pyConsts[name]
		if !ok {
			t.Errorf("Python 版缺少常量 %s", name)
			continue
		}
		goText, ok := goConsts[name]
		if !ok {
			t.Errorf("Go 版缺少常量 %s", name)
			continue
		}
		if pyText != goText {
			t.Errorf("常量 %s 与 Python 版不一致 (len py=%d go=%d)", name, len(pyText), len(goText))
		}
	}
}
