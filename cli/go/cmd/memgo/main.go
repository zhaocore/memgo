// memgo CLI 入口 (cobra; 第三实现, 上游 python/node CLI 零改动)。
package main

import (
	"os"

	cli "github.com/zhao-core/memgo/cli/go"
)

func main() {
	args := os.Args[1:]
	cleaned := args
	if isInitIn(args) {
		// init 的 --agent 是子命令语义 (Agent Mode bootstrap), 从全局摘除
		cleaned = nil
		for _, a := range args {
			if a == "--agent" {
				continue
			}
			cleaned = append(cleaned, a)
		}
	}
	os.Exit(cli.Execute(cleaned))
}

func isInitIn(args []string) bool {
	for _, a := range args {
		if a == "init" {
			return true
		}
	}
	return false
}
