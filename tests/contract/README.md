# 契约套件 (P0)

黑盒 HTTP 套件: 同一套用例打 Python server (采基线) 与 Go server (验目标)。
合同基准: `memgo/architecture/doc-02` (逐端点) + `doc-03 §3.3` (鉴权回退链)。

## 结构

- `conftest.sh` — HTTP/断言/归一化/golden 助手 (bash 3.2 兼容)
- `run.sh` — 编排器: 前置检查、stub 自动拉起、按序执行套件
- `stub/` — OpenAI 兼容打桩 server (Go, stdlib): 固定 LLM/embedding 响应 + `/_control` 故障注入
- `compose.contract.yaml` — Podman 契约栈: pgvector + Python server (:18888) + AUTH_DISABLED 实例 (:18889)
- `goldens/` — 从 Python server 采集的响应快照 (归一化后)

## 运行

```bash
# 1. 起栈 (全新库)
podman compose -f tests/contract/compose.contract.yaml down -v
podman compose -f tests/contract/compose.contract.yaml up -d --build

# 2. 采基线 (一次性, 冻结 goldens/)
./tests/contract/run.sh --capture

# 3. 回归 (Python) / 验 Go (CONTRACT_BASE_URL 指向 Go server)
./tests/contract/run.sh
CONTRACT_BASE_URL=http://localhost:8888 ./tests/contract/run.sh --impl go
```

## 确定性策略

- `infer=false` 全确定性 (stub embedder 恒定向量 → score 恒 1.0)
- `infer=true` 走 stub LLM 固定响应, 结构性断言
- 归一化抹平: uuid/时间戳/token/key/hash/latency/score(4 位)
- 限流用例依赖 1 分钟窗口, 偶发失败重跑即可

## 已知盲区 (黑盒不可测)

| 项 | 原因 | Go 侧对策 |
|----|------|-----------|
| `provider_timeout` code | openai SDK 客户端默认 600s 超时 | errpkg 分类器单测 |
| 409 email 冲突 | registration closed, 无法建第二用户 | store/handler 单测 |
| 403 分支 (role!=admin) | 单 admin 拓扑不可达 (registration closed) | require_admin/模式B 判定逻辑单测 |
| POST /configure 多 worker 广播 | 单实例部署前提 | 文档记录 (doc-03 §5.2 缺陷不移植) |

## 实测合同发现 (doc-02 与源码行为不符处, Go 以实测为准)

1. **PUT `{"text": null}` → 400**, 非 doc-02 §2.4 所述"显式置空" —— SDK `update(data=None)` 抛 ValueError。
   fields_set 语义实际生效面: 只传 `metadata` 不动内容、`{"expiration_date": null}` 清除、`{}` → 400
   `"At least one of text, metadata, or expiration_date must be provided."`。
2. **doc-02 §8 矩阵 user-key 403 格在单 admin 部署不可达**: require_admin/模式B 判定是
   `role != admin AND auth_type not in {admin_api_key, disabled}` —— 用户 key 解析出的唯一用户 role 恒 admin。
3. **bootstrap 哨兵只覆盖 require_admin 面**: `verify_auth` 对 ADMIN_API_KEY 返回 None,
   空库时 require_auth (如 POST /api-keys) → 401; require_admin (如 GET /requests) → _BOOTSTRAP_ADMIN 哨兵放行。
4. **infer=false 重复 add 无 hash 去重** → 恒 ADD (NONE 事件来自 infer=true 决策链)。
5. **POST /search threshold>1 → 400** (SDK ValueError), 非空结果集。
