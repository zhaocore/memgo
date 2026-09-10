package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewRootCmd 组装 memgo 命令树 (对齐 app.py 选项矩阵; parity golden 锁定)。
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "memgo",
		Short:         "◆ memgo CLI v" + Version + " · Go\n\n   The Memory Layer for AI Agents",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	// completion 不在上游矩阵内, 禁用; 自定义 help 顶替 cobra 默认
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetHelpCommand(newHelpCmd())
	root.PersistentFlags().BoolVarP(&flagJSON, "json", "", false, "Output as JSON for agent/programmatic use.")
	root.PersistentFlags().BoolVar(&flagAgent, "agent", false, "Alias of --json.")
	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if flagJSON || flagAgent {
			SetAgentMode(true)
		}
		if cmd.Name() != "init" {
			setCurrentCommand(cmd.Name())
		}
	}
	root.AddCommand(
		newAddCmd(), newSearchCmd(), newGetCmd(), newListCmd(),
		newUpdateCmd(), newDeleteCmd(), newImportCmd(),
		newConfigCmd(), newEntityCmd(), newEventCmd(),
		newInitCmd(), newIdentifyCmd(), newWhoamiCmd(),
		newStatusCmd(), newVersionCmd(),
	)
	return root
}

// 全局标志。
var (
	flagJSON  bool
	flagAgent bool
)

// Execute 入口 (main.go 调用)。
func Execute(args []string) int {
	root := NewRootCmd()
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		if err != errExit {
			printError(err.Error(), "")
		}
		if n := TakeNotice(); n != "" && !IsAgentMode() {
			fmt.Fprintf(stderrW(), "\n🔔 %s\n\n", n)
		}
		return 1
	}
	if n := TakeNotice(); n != "" && !IsAgentMode() {
		fmt.Fprintf(stderrW(), "\n🔔 %s\n\n", n)
	}
	return 0
}
