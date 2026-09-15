# MemGo 快速开始

两分钟把 MemGo 跑起来并增删查。前提是一个可访问的 server + 一个 API key。

## 前置

- Docker(本地起栈用 `make`)或一个已部署的 MemGo server(如 `https://memgo.wxget.com`)
- 环境变量:

  ```bash
  export MEMGO_API_KEY="..."
  export MEMGO_BASE_URL="https://memgo.wxget.com"
  ```

## 1. 起 server

```bash
export POSTGRES_PASSWORD=... JWT_SECRET=$(openssl rand -base64 48) OPENAI_API_KEY=sk-...
make up            # memgo-server(:8888) + pgvector(:8432)
make bootstrap     # setup-status → register → 建 API key
make health        # 健康三探
make down          # 停栈清卷
```

## 2. curl 增删查

```bash
# 写入(同步,立即完成)
curl -X POST $MEMGO_BASE_URL/memories \
  -H "X-API-Key: $MEMGO_API_KEY" -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user", "content": "我吃素,对坚果过敏"},
      {"role": "assistant", "content": "记住了。"}
    ],
    "user_id": "user123"
  }'

# 搜索
curl -X POST $MEMGO_BASE_URL/search \
  -H "X-API-Key: $MEMGO_API_KEY" -H "Content-Type: application/json" \
  -d '{"query": "饮食偏好", "filters": {"user_id": "user123"}}'

# 列表
curl "$MEMGO_BASE_URL/memories?user_id=user123" -H "X-API-Key: $MEMGO_API_KEY"

# 更新 / 删除
curl -X PUT "$MEMGO_BASE_URL/memories/<id>" -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" -d '{"text": "现在纯素,不吃坚果"}'
curl -X DELETE "$MEMGO_BASE_URL/memories/<id>" -H "X-API-Key: $MEMGO_API_KEY"
```

写入响应:

```json
{
  "results": [
    {"id": "ea925981-...", "memory": "用户吃素,对坚果过敏。", "event": "ADD"}
  ]
}
```

## 3. Go 示例

### 方式 A:直接调 OSS HTTP(最简单,标准库即可)

```go
// example_http.go —— 任意 Go 程序直连 MemGo server
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

const baseURL = "https://memgo.wxget.com" // 或 env MEMGO_BASE_URL

func call(method, path string, body any, out any) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, baseURL+path, bytes.NewReader(b))
	req.Header.Set("X-API-Key", os.Getenv("MEMGO_API_KEY"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func main() {
	// 写入
	type addResp struct {
		Results []map[string]any `json:"results"`
	}
	var added addResp
	err := call("POST", "/memories", map[string]any{
		"messages": []map[string]string{{"role": "user", "content": "我吃素,对坚果过敏"}},
		"user_id":  "user123",
	}, &added)
	if err != nil {
		panic(err)
	}
	fmt.Println("写入:", added.Results)

	// 搜索
	type searchResp struct {
		Results []map[string]any `json:"results"`
	}
	var res searchResp
	err = call("POST", "/search", map[string]any{
		"query":   "饮食偏好",
		"filters": map[string]string{"user_id": "user123"},
	}, &res)
	if err != nil {
		panic(err)
	}
	for _, m := range res.Results {
		fmt.Println("命中:", m["memory"])
	}
}
```

### 方式 B:内嵌 Go core 库(不经网络,依赖注入)

```go
import (
	"github.com/zhao-core/memgo/core/config"
	"github.com/zhao-core/memgo/core/embedder"
	"github.com/zhao-core/memgo/core/history"
	"github.com/zhao-core/memgo/core/llm"
	"github.com/zhao-core/memgo/core/memory"
	"github.com/zhao-core/memgo/core/vectorstore"
)

// 组装:config + pgvector + embedder + llm + sqlite 历史库 + 实体抽取器
cfg, _ := config.Load(nil) // 或 config.MemoryConfig 手动构建
vec, _ := vectorstore.NewPGVector(vectorstore.PGVectorConfig{ /* 连接参数 */ })
emb, _ := embedder.NewOpenAI(embedder.OpenAIConfig{ /* api key */ })
lang, _ := llm.NewOpenAI(llm.DefaultOpenAIConfig()) // 或 anthropic/gemini
hist, _ := history.NewManager(cfg.HistoryDBPath)

m := memory.New(cfg, vec, emb, lang, hist, nil)

// 写入
_, _ = m.Add([]map[string]any{
	{"role": "user", "content": "我吃素,对坚果过敏"},
}, memory.AddParams{UserID: "user123"})

// 搜索
res, _ := m.Search(memory.SearchParams{
	Query:   "饮食偏好",
	Filters: map[string]any{"user_id": "user123"},
})
```

## 4. CLI

```bash
go build -o bin/memgo ./cli/go/cmd/memgo
bin/memgo init --api-key "$MEMGO_API_KEY" --user-id alice
bin/memgo add "我喜欢爬山" -u alice
bin/memgo search "户外" -u alice -o table
bin/memgo get <id>
bin/memgo list -u alice
bin/memgo update <id> "更新后的文本"
bin/memgo delete <id>
```

打自托管 server 时保证 `MEMGO_BASE_URL` 指向它即可。

## 5. Python / Node 直连(无官方 SDK)

MemGo **没有**官方 Python/TS SDK —— 直连 OSS HTTP 即可,所有 HTTP 客户端都行。

**Python(标准库):**

```python
import os, json, urllib.request

def api(method: str, path: str, body: dict | None = None) -> dict:
    req = urllib.request.Request(
        os.environ["MEMGO_BASE_URL"] + path,
        data=json.dumps(body).encode() if body else None,
        method=method,
        headers={
            "X-API-Key": os.environ["MEMGO_API_KEY"],
            "Content-Type": "application/json",
        },
    )
    with urllib.request.urlopen(req) as r:
        return json.load(r)

api("POST", "/memories", {
    "messages": [{"role": "user", "content": "我吃素,对坚果过敏"}],
    "user_id": "alice",
})
results = api("POST", "/search", {"query": "饮食", "filters": {"user_id": "alice"}})
print(results["results"])
```

**Node(TypeScript,无依赖):**

```typescript
const base = process.env.MEMGO_BASE_URL!;
const key = process.env.MEMGO_API_KEY!;

async function api<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(base + path, {
    method,
    headers: { "X-API-Key": key, "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
  return res.json() as Promise<T>;
}

await api("POST", "/memories", {
  messages: [{ role: "user", content: "我吃素,对坚果过敏" }],
  user_id: "alice",
});
const results = await api<{ results: Array<{ memory: string }> }>("POST", "/search", {
  query: "饮食",
  filters: { user_id: "alice" },
});
```

## 下一步

- [SDK Guide](sdk-guide.md) —— Go core 库全部方法与 HTTP 调用对照
- [API Reference](api-reference.md) —— 端点 / filters / 记忆对象
- [Integration Patterns](integration-patterns.md) —— MCP / CLI / 编码 agent / HTTP 直连
