package llm

import "fmt"

// StatusError 携带上游 HTTP 状态, 供 server/errpkg 做 502 code 归因。
type StatusError struct {
	Status int
	Msg    string
}

func (e *StatusError) Error() string { return e.Msg }

// NewStatusError 构造。
func NewStatusError(status int, format string, args ...any) *StatusError {
	return &StatusError{Status: status, Msg: fmt.Sprintf(format, args...)}
}
