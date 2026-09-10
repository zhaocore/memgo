// Package errpkg: 上游错误分类与错误信封 (对齐 server/errors.py)。
package errpkg

import (
	"net/http"
)

// Code 全集 (doc-02 §1.3)。
const (
	CodeProviderAuth      = "provider_auth_failed"
	CodeProviderRateLimit = "provider_rate_limited"
	CodeProviderTimeout   = "provider_timeout"
	CodeProviderUnavail   = "provider_unavailable"
	CodeProviderBadReq    = "provider_bad_request"
	CodeDatastoreUnavail  = "datastore_unavailable"
	CodeVectorUnavail     = "vector_store_unavailable"
	CodeUnknown           = "unknown"
)

// CodeDetail 对齐 _classify_one 的 detail 文案 (逐字, golden 锁定)。
var CodeDetail = map[string]string{
	CodeProviderAuth:      "Provider rejected the request (authentication). Check your LLM provider API key on the Configuration page.",
	CodeProviderRateLimit: "Provider rate limit hit. Retry shortly.",
	CodeProviderTimeout:   "Provider timed out. Retry shortly.",
	CodeProviderUnavail:   "Provider is unreachable or returned a server error.",
	CodeProviderBadReq:    "Provider rejected the request as malformed.",
	CodeDatastoreUnavail:  "The memory database is unreachable.",
	CodeVectorUnavail:     "The vector store is unreachable or returned an error.",
	CodeUnknown:           "Upstream provider error.",
}

// UpstreamError 携带 502 信封三要素。
type UpstreamError struct {
	Code      string
	Detail    string
	RequestID string
}

func (e *UpstreamError) Error() string { return e.Detail }

// Status 实现 HTTP 语义。
func (e *UpstreamError) Status() int { return http.StatusBadGateway }

// New 构造 (detail 按 code 查表)。
func New(code, requestID string) *UpstreamError {
	detail, ok := CodeDetail[code]
	if !ok {
		code = CodeUnknown
		detail = CodeDetail[CodeUnknown]
	}
	return &UpstreamError{Code: code, Detail: detail, RequestID: requestID}
}

// StatusCarrier 允许任意错误暴露上游 HTTP 状态 (llm.StatusError 实现)。
type StatusCarrier interface {
	error
	Status() int
}
