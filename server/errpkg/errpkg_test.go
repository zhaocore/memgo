package errpkg

import (
	"errors"
	"fmt"
	"testing"

	"github.com/zhao-core/memgo/core/llm"
)

// TestClassifyStatusMatrix: 502 code 归因表 (黑盒不可测的 provider_timeout 在此覆盖)。
func TestClassifyStatusMatrix(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{llm.NewStatusError(401, "x"), CodeProviderAuth},
		{llm.NewStatusError(403, "x"), CodeProviderAuth},
		{llm.NewStatusError(429, "x"), CodeProviderRateLimit},
		{llm.NewStatusError(400, "x"), CodeProviderBadReq},
		{llm.NewStatusError(422, "x"), CodeProviderBadReq},
		{llm.NewStatusError(500, "x"), CodeProviderUnavail},
		{llm.NewStatusError(503, "x"), CodeProviderUnavail},
		{llm.NewStatusError(0, "x"), CodeProviderUnavail},
		{fmt.Errorf("pgvector: 检索失败: %w", errors.New("conn refused")), CodeVectorUnavail},
		{fmt.Errorf("appdb: 查询失败: %w", errors.New("conn refused")), CodeDatastoreUnavail},
		{errors.New("mystery"), CodeUnknown},
	}
	for i, c := range cases {
		if got := Classify(c.err); got != c.want {
			t.Errorf("case %d: 期望 %s 得 %s", i, c.want, got)
		}
	}
	// wrapped StatusError 沿链归因
	wrapped := fmt.Errorf("add 流水线: %w", llm.NewStatusError(429, "rate"))
	if got := Classify(wrapped); got != CodeProviderRateLimit {
		t.Errorf("wrapped 归因失败: %s", got)
	}
	// provider_timeout: 黑盒不可测, 分类器必须能产出
	if CodeDetail[CodeProviderTimeout] == "" {
		t.Errorf("provider_timeout 文案缺失")
	}
}

// TestDetailTextsLocked: golden 锁定的 detail 文案逐字校验 (防手滑改文案)。
func TestDetailTextsLocked(t *testing.T) {
	want := map[string]string{
		CodeProviderAuth:      "Provider rejected the request (authentication). Check your LLM provider API key on the Configuration page.",
		CodeProviderRateLimit: "Provider rate limit hit. Retry shortly.",
		CodeProviderUnavail:   "Provider is unreachable or returned a server error.",
		CodeProviderBadReq:    "Provider rejected the request as malformed.",
	}
	for code, text := range want {
		if CodeDetail[code] != text {
			t.Errorf("code %s detail 文案漂移: %q", code, CodeDetail[code])
		}
	}
}
