package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// golden 路径 (相对模块根)。
const parityGoldenPath = "../../tests/contract/cli_parity_golden.json"

// allowedMissing 显式列明 Go 侧缺席项 (计划 P3 验收: 不得静默缺失)。
var allowedMissing = map[string]string{
	"agent-rush": "AGENTRUSH 游戏命令 (交互+平台专属), 计划 R10 尽力项, 未迁移",
}

// TestCLIParityWithPython: 三向 parity golden —— Go 不得有矩阵外命令/选项; 缺席须列明。
func TestCLIParityWithPython(t *testing.T) {
	raw, err := os.ReadFile(parityGoldenPath)
	if err != nil {
		t.Fatalf("parity golden 缺失: %v", err)
	}
	var golden struct {
		Commands map[string]struct {
			Arguments   []string       `json:"arguments"`
			Options     []string       `json:"options"`
			Subcommands map[string]any `json:"subcommands"`
		} `json:"commands"`
	}
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatalf("golden 解析失败: %v", err)
	}

	root := NewRootCmd()
	goCmds := map[string]*cobra.Command{}
	for _, c := range root.Commands() {
		goCmds[c.Name()] = c
	}
	// SetHelpCommand 登记的 help 在 Execute 前不在 Commands(): 显式纳入
	goCmds["help"] = newHelpCmd()

	// 1) Go 不得有矩阵外命令
	for name := range goCmds {
		if _, ok := golden.Commands[name]; !ok {
			t.Errorf("矩阵外命令: %s (上游双 CLI 不存在)", name)
		}
	}
	// 2) 缺席项必须显式列明
	for name := range golden.Commands {
		if _, ok := goCmds[name]; !ok {
			if _, allowed := allowedMissing[name]; !allowed {
				t.Errorf("缺席命令未列明: %s", name)
			}
		}
	}
	// 3) 共有命令: 选项集须包含 golden 全集 (Go 可为空缺? 不行 —— 逐项比对)
	for name, spec := range golden.Commands {
		cmd, ok := goCmds[name]
		if !ok {
			continue
		}
		goOpts := map[string]bool{}
		goArgs := map[string]bool{}
		collectParams(cmd, goOpts, goArgs)
		for _, opt := range spec.Options {
			if !goOpts[opt] {
				t.Errorf("命令 %s 缺选项: %s", name, opt)
			}
		}
		for _, arg := range spec.Arguments {
			if !goArgs[arg] {
				// cobra 不携带参数名元数据: 以 Use 字段包含性校验
				if !strings.Contains(cmd.Use, "<"+arg+">") && !strings.Contains(cmd.Use, "["+arg+"]") {
					t.Errorf("命令 %s 缺参数: %s (Use=%q)", name, arg, cmd.Use)
				}
			}
		}
		// 子命令 (config/entity/event)
		if len(spec.Subcommands) > 0 {
			for sub := range spec.Subcommands {
				if cmd.Commands() == nil || findSub(cmd, sub) == nil {
					t.Errorf("命令 %s 缺子命令: %s", name, sub)
				}
			}
		}
	}
}

// collectParams 收集命令选项串与 Use 中的参数占位。
func collectParams(cmd *cobra.Command, opts map[string]bool, args map[string]bool) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		opts["--"+f.Name] = true
		if f.Shorthand != "" {
			opts["-"+f.Shorthand] = true
		}
	})
	for _, name := range []string{"text", "query", "memory_id", "key", "value", "file_path", "entity_type", "event_id", "name"} {
		if strings.Contains(cmd.Use, "<"+name+">") || strings.Contains(cmd.Use, "["+name+"]") {
			args[name] = true
		}
	}
}

func findSub(cmd *cobra.Command, name string) *cobra.Command {
	for _, c := range cmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}
