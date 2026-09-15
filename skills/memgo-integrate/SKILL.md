---
name: memgo-integrate
description: >
  Integrate MemGo into an existing repository using a goal-driven, TDD pipeline.
  Detects the repo's language automatically. MemGo is a self-hosted memory
  server with an OSS API surface (no managed platform). Writes failing tests
  before any implementation. Produces a local feature branch plus
  `.memgo-integration/` artifacts.
  TRIGGER when: user says "integrate memgo", "add memgo to this repo", "wire
  memgo into <repo>", or asks how to add memory to an existing project.
  DO NOT TRIGGER when: the user wants general CLI usage (use the `memgo` CLI
  directly).
license: Apache-2.0
metadata:
  author: memgo
  version: "0.1.0"
  category: ai-memory
  tags: "memory, integration, tdd, oss"
---

# memgo-integrate

Wire MemGo into an existing repo with a goal-driven, test-first pipeline.
验证内置本 skill(起 server → 写入 → 搜索 → 断言命中),不依赖外部验证 skill。

## Canonical sources (read before deciding anything)

The skill MUST read these before step 6 and cite them in `plan.md`. They are
the ground truth — do not rely on ambient knowledge of the MemGo API.

- **OSS API 契约(权威)**: 本仓库 `integrations/README.md`(相对本 skill 目录
  `../../integrations/README.md`)。端点、鉴权(`X-API-Key`)、实体范围
  (`user_id`/`agent_id`/`run_id`,无 `app_id`/`custom_categories`)、同步写入语义。
- **server 起栈 / CLI / 环境变量**: 本仓库 `README.md`(`make up`/`make
  bootstrap`、`bin/memgo` CLI、`MEMGO_BASE_URL` 等)。
- **现成调用范例**: 本仓库 `integrations/` 下各接入实现(claude-code-plugin /
  codex / deepseek-plugin / mcp),直接复用其请求/响应形状,不要重新发明。
- **请求/响应形状基准**: `tests/contract/` 下的契约测试脚本。

OSS API 契约速查(仍以 `integrations/README.md` 为准):

```
POST   {base}/memories     # 写入 (同步)
       body: {messages:[{role,content}], user_id?, agent_id?, run_id?, metadata?, infer?}
       响应: {"results":[{id,memory,user_id,agent_id,run_id,metadata,created_at,updated_at}]}
       memory 字段 = 提取后的记忆文本

POST   {base}/search        # 搜索 (同步)
       body: {query, filters:{user_id?, agent_id?, run_id?}, top_k?}
       filters 至少含一个 user_id/agent_id/run_id
       响应: {"results":[{...同上}]}
```

鉴权头 `X-API-Key: <key>`;`MEMGO_BASE_URL` 默认 `https://memgo.wxget.com`,
本地起栈用 `http://localhost:8888`。写入为同步语义:`POST /memories` 直接返回
`results`,无事件轮询。

## Integration principles (non-negotiable)

The true goal of this skill is to produce a **PR the maintainers can accept
without argument**. That rules out anything invasive.

1. **Additive, not replacing.** If the target repo already has a memory
   system, a session store, a user-context layer, or anything named
   `Memory` / `memory_*`, MemGo sits **alongside** it, not in place of it.
   The existing system keeps working unchanged.
2. **Opt-in by default.** Gate all new MemGo code behind a feature flag
   (env var like `MEMGO_ENABLED=1`, a config key, or a strategy selector).
   With the flag unset, behavior is the repo's original behavior,
   byte-for-byte.
3. **No breakage.** No removed exports, no renamed public functions,
   no changed method signatures, no modified existing tests, no changed
   behavior of existing tests. All pre-existing tests must pass unchanged
   both with the flag set and unset.
4. **No new dependency surface.** MemGo 是 HTTP 服务,用目标仓库已有的 HTTP
   client(Go `net/http`,Python `requests`/`httpx`,Node `fetch`/`axios`)直连
   OSS API。不加新 SDK、新向量库、新 provider 依赖。
5. **Separable commits.** Code, tests, and config/docs land in separate
   commits so reviewers can cherry-pick.
6. **The null hypothesis wins.** If no additive, gated fit exists after
   step 6 (plan), exit with code 1 and a rationale. A bad PR is worse
   than no PR.
7. **Backend only.** MemGo integration lives in server-side code. API keys,
   memory scope, and user-identity resolution are not safe client-side.
   If the repo has both backend and frontend, the call sites live in
   backend files. Frontend-only repos are rejected at preconditions.

Enforced at four gates: **preconditions** (reject frontend-only repos
and repos where additive fit is impossible), **step 2 comprehension**
(confirm a backend exists and name candidate surfaces), **step 6 plan
review** (reject plans that mutate existing exports or name client-side
call sites), and **step 10 self-healing loop** (refuse to "fix" principle
violations — surface them instead).

## Integration style (no published skill library)

MemGo 无发布版 SDK skill;接入一律直连 OSS API(HTTP),不要仿照托管平台的 SDK
调用形态。写调用点前,先读本仓库 `integrations/` 下同栈接入的现有实现,复制其
请求/响应形状。

| Detected in target repo | Copy call-site pattern from |
|---|---|
| Python 后端 (FastAPI/Flask/Django/agent 框架) | `integrations/claude-code-plugin/` 或 `integrations/codex/` 的 Python 接入 |
| Node / TS 后端 | `integrations/deepseek-plugin/` (TS) |
| 需要 MCP 形态 | `integrations/mcp/` |
| 无现成同栈范例 | 按 Canonical sources 里的 OSS API 契约手写 HTTP 调用 |

目标仓库没有 LLM 调用点、或已有自己的记忆/会话系统时,先按 Integration
principles 判断是否值得接入,不要硬塞。

## Preconditions

Refuse to start unless ALL of the following are true:

- Current working directory is inside a git repository with a clean index
  (no uncommitted changes). Protects the user's work — every edit lands on
  a feature branch, not on top of in-progress changes.
- Repo has a detectable language (`package.json` / `pyproject.toml` /
  `requirements.txt` / `go.mod`). No language → exit cleanly with a written rationale.
- Repo has a **backend**. Detected by: a `backend/` or `server/` or `api/`
  directory; a Python package with FastAPI/Flask/Django/Starlette; a Node
  package with Express/Fastify/Koa/NestJS/Next-API-routes; a Go module with
  an HTTP server; an agent-loop framework (LangGraph, LangChain, LlamaIndex,
  Agno). Frontend-only repos (pure React/Vue/Svelte SPAs, static sites,
  mobile-only) → exit with code 1 and a rationale. MemGo is not installed
  client-side.
- The user has already decided MemGo fits this repo. This skill does NOT
  survey the codebase to justify fit — bring a concrete goal. (Step 2
  *does* read the repo to understand what it does and locate backend
  integration surfaces; that is mechanics, not fit-justification.)

Exit with a written rationale if any precondition fails. Do not try to
"make it work anyway."

## Pipeline

Ten steps. Full mechanics, document templates, and gate rules are in
[`references/pipeline.md`](references/pipeline.md). Read that file when you
start executing a step; the summary below is only for routing.

| # | Step | Gate |
|---|---|---|
| 1 | **Language detection.** `package.json` / `pyproject.toml` / `requirements.txt` / `go.mod`. Monorepo, ask which subdirectory. | |
| 2 | **Repo comprehension.** Budgeted read of README, contributor docs, entry points, top two directory levels. Produces `repo-summary.md` with ranked backend surfaces. | User confirms the summary and picks a surface. No backend surface, exit 1. |
| 3 | **Server selection.** remote (`https://memgo.wxget.com`) vs local (`make up`, :8888), recommended from signals, never asked blank. | Locked into `goal.md`, never re-decided. |
| 4 | **Server + key check.** `MEMGO_BASE_URL` (默认 `https://memgo.wxget.com`) + `MEMGO_API_KEY`(必填),探活 `GET /memories`。客户端无需 `OPENAI_API_KEY`(LLM/embedder 由 server 配置)。 | CI mode with a missing key or unreachable server, exit 2. |
| 5 | **Goal doc.** `goal.md`: what gets stored, when it is retrieved, why, server, API scope, out of scope. | **Hard gate.** Explicit approval required. 3 rejections, exit 3. |
| 6 | **Integration plan.** Scoped grep for call sites and identity source. `plan.md`: write/read patterns, scoping, call sites, HTTP client, preserved behavior, coexistence, feature flag, sources, E2E recipe. | **Hard gate.** No plausible additive call site or 3 rejections, exit 5. |
| 7 | **Tests first.** Failing write and read tests in the repo's native framework, assertion shapes lifted from the OSS API 契约 (`POST /memories`, `POST /search`). Must be importable with `MEMGO_API_KEY` unset. | Tests must fail. If they pass, they are wrong. |
| 8 | **Implementation.** Fresh-context subagent, prompt in [`references/subagent-prompts.md`](references/subagent-prompts.md), returns a diff reviewed against `plan.md` and `goal.md`. | 3 review loops, then exit 4. |
| 9 | **Commit and handoff.** Branch `memgo-integrate/<slug>`, separable commits: config, module, wiring, tests. | `--no-heal` stops here. |
| 10 | **Self-healing loop.** 原生测试(flag on/off)+ E2E(起 server → 写入 → 搜索 → 断言命中),categorizes the failure, spawns a bounded remediation subagent, reverts on regression. | Pre-existing test failure, **stop**, exit 6. Never "fix" it. |

## Artifacts (all under `.memgo-integration/`)

| File | Purpose | Retention |
|---|---|---|
| `repo-summary.md` | Repo comprehension + candidate backend surfaces (step 2). | Keep across runs. |
| `goal.md` | Approved intent. Never rewritten after step 6. | Keep across runs. |
| `plan.md` | Approved mechanics (where, how, call sites, preserved behavior). | Keep across runs. |
| `trace.jsonl` | Every tool call, decision, and subagent exchange this run. | Overwritten per run. |
| `diff.patch` | The committed integration as a reviewable patch. | Overwritten per run. |
| `heal-trace.md` | Per-attempt record of the self-healing loop (step 10). | Overwritten per run. |
| `product.json` | `{"product": "oss", "server": "remote"\|"local", "language": "...", "memgo_version": "...", "write_site": "file:line", "read_site": "file:line", "feature_flag": "MEMGO_ENABLED"}`. | Overwritten per run. |

`.memgo-integration/` is added to `.gitignore` on first run. Nothing is
written outside this directory and the repo's source tree.

## Modes

| Mode | Trigger | Behavior |
|---|---|---|
| Interactive (default) | TTY present, `MEMGO_INTEGRATE_CI` unset | Asks for keys, confirms goal doc, shows recommendations. |
| CI | `MEMGO_INTEGRATE_CI=1` | Requires key + reachable server, requires `--server`, auto-approves goal doc from `goal.md` if present, fails fast otherwise. |

## Invocation

    /memgo-integrate                            # interactive, heal ON
    /memgo-integrate --no-heal                  # stop after commit; manual verify
    /memgo-integrate --heal-max 5               # cap heal attempts per category (default 3)
    /memgo-integrate --server local             # skip the server ask (remote|local)
    /memgo-integrate --ci                       # non-interactive (for test harness)

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success. Feature branch committed; verification (tests + E2E) passed. |
| 1 | Precondition failed (dirty repo, no detectable language, etc.). |
| 2 | Missing env key or unreachable server in CI mode. |
| 3 | Goal doc rejected 3+ times — integration is not well-specified. |
| 4 | Subagent review loop did not converge in 3 rounds. |
| 5 | Integration plan rejected 3+ times, or no plausible additive call site found. |
| 6 | Self-healing loop did not converge, detected a non-invasiveness violation, or a pre-existing test failed. |

## Explicitly out of scope

- Surveying the repo for fit points. Humans decide where MemGo helps before
  invoking this skill.
- Replacing any existing memory / session / state system. Always additive
  and feature-flagged; see "Integration principles."
- Modifying pre-existing tests, even to "fix" them under self-heal. Tests
  that fail after integration with the flag unset are a non-invasiveness
  violation, not a bug to patch.
- Server 部署/扩容:接入端只管调用;起栈用本仓库 `make up`/`make bootstrap` 或
  运维侧,不在目标仓库里管 server。
- LLM/embedder/向量库选择:由 MemGo server 端配置,客户端不接触。
- 托管平台习惯(事件轮询、`custom_categories`、`app_id`):OSS 面不存在,不要写。
- Switching branches, pushing, or opening PRs. Commits locally and stops
  (or enters the heal loop, still local).
- Data migration between stores.
