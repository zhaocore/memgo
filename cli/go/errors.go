package cli

import (
	"errors"
	"os"
)

// backendErr 后端错误 → 输出 + 退出码 (对齐 commands/utils.py 的错误映射)。
func backendErr(err error) error {
	var ae *APIError
	if errors.As(err, &ae) {
		switch ae.Kind {
		case "auth":
			printError(ae.Msg, "Run 'memgo init' or set MEM0_API_KEY environment variable.")
		case "not_found":
			printError(ae.Msg, "")
		default:
			printError(ae.Msg, "")
		}
		return errExit
	}
	printError(err.Error(), "")
	return errExit
}

// exitCodeOf 错误映射退出码。
func exitCodeOf(err error) int {
	if err == nil {
		return 0
	}
	return 1
}

// dieImmediate 直接退出 (帮助语境)。
func dieImmediate() { os.Exit(1) }
