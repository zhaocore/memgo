# Pipeline mechanics

Full step-by-step for `memgo-integrate`. Read this when you are executing a
step. The one-line-per-step overview and every non-negotiable rule live in
`../SKILL.md`, which is loaded on every run; this file is loaded on demand.

Verbatim subagent system prompts for steps 8 and 10 are in
[`subagent-prompts.md`](subagent-prompts.md).

## 1. Language detection

| Signal | Track |
|---|---|
| `package.json` + TypeScript config | Node / TypeScript |
| `package.json` (no TS config) | Node / JavaScript |
| `pyproject.toml` or `requirements.txt` | Python |
| `go.mod` | Go |

Monorepo with both, ask which subdirectory to operate in, then recurse.

## 2. Repo comprehension: what does this repo do, and where is the backend?

Before any decision (goal, plan), understand the repo enough to
locate *where in the backend* the integration belongs. This is not
fit-surveying, the user already decided MemGo fits. This is mechanics: you
cannot write a plan without knowing what files matter.

Read, in order, with a token budget. Do not scan the whole tree.

1. `README.md` (root) plus the first page of any `README_*.md` variants.
2. `CONTRIBUTING.md` / `AGENTS.md` / `CLAUDE.md` at root if present. These
   often spell out architecture and entry points.
3. `package.json` / `pyproject.toml` / `go.mod` scripts and entry points.
4. The layout of the top two directory levels, not recursive.
5. Key config files: `docker-compose.yml`, `Dockerfile`, `Makefile`,
   `langgraph.json`, `next.config.*`, `nuxt.config.*`.

Produce `.memgo-integration/repo-summary.md`:

    # Repo comprehension

    **What this repo does:** <one paragraph in plain English. Who is
    the end user? What does the app do for them? What LLM / agent
    behavior is central? Do not list dependencies, describe behavior.>

    **Architecture at a glance:**
      - Backend: <path(s), framework, primary entry point>
      - Frontend: <path(s) if any, framework, for context only; no
        integration here>
      - Agent loop / orchestration: <LangGraph? custom? none?>
      - Existing memory/session/state systems: <name them, these are
        what step 6 Coexistence must preserve>

    **Candidate backend integration surfaces** (ranked, best first):
      1. `<backend-file>:<line_range>` <function> <one-sentence
         reason this is where write/read could slot in without
         replacing anything existing>
      2. ...
      3. ...

    **Not a fit here:** <list anything the skill considered but ruled
    out, e.g. "frontend chat component: client-side, excluded by
    backend-only rule"; "existing memory subsystem X: would require
    replacement, excluded by additive principle">

    **Sources read:** <list the files actually opened, with line counts,
    so reviewers can verify coverage.>

Show the user the rendered summary and ask: *"Is this understanding correct?
Which of the candidate surfaces (1, 2, 3 ...) should step 3 forward target?"*

Gate rules:

- No backend surface found, exit code 1. Preconditions should already have
  caught frontend-only repos; reaching this point means a subtler miss (for
  example the "backend" is actually just a static build). Do not force a fit.
- Every candidate surface would require replacing an existing memory or
  session system, exit code 1 with the additive-principle rationale. The user
  can point at a non-conflicting location manually and re-run.
- User corrections update `repo-summary.md` and re-confirm. Max 3 rounds,
  beyond that exit code 1.

The user's chosen surface index is baked into `product.json` as
`preferred_site` and referenced by steps 5 and 6.

## 3. Server selection: remote vs local

MemGo 只有 OSS 面、没有托管平台;步骤 3 决定打哪个 server。先读本 skill
Canonical sources 里的 OSS API 契约(`integrations/README.md`),再按启发式
推荐。问,但从不空白问。

- 目标仓库已有 `MEMGO_BASE_URL` / `MEMGO_API_KEY` 环境变量,或团队已部署
  MemGo server → 推荐 **remote**(`https://memgo.wxget.com`,可用
  `MEMGO_BASE_URL` 覆盖)。
- 目标仓库带 `docker-compose.yml`(postgres/pgvector + qdrant),或本地开发
  走 `make up` + `make bootstrap` 起栈 → 推荐 **local**
  (`http://localhost:8888`)。
- 无强信号,默认推荐 **remote**:部署在 `https://memgo.wxget.com`,开箱即用。

Example:

> 目标仓库没有本机起栈痕迹。我推荐 **remote**(https://memgo.wxget.com,
> `X-API-Key` 鉴权)。覆盖为本机 `make up` 起栈?

Bake the choice into the goal doc in step 5 (`Server: remote | local`)。Do not
re-decide later.

## 4. Server reachability + API key check

| 项 | 值 |
|---|---|
| Base URL | `MEMGO_BASE_URL`(默认 `https://memgo.wxget.com`;local 用 `http://localhost:8888`) |
| API key | `MEMGO_API_KEY`(必填) |
| 鉴权头 | `X-API-Key: <key>`(不是 `Authorization: Token`) |
| 探活 | `GET {base}/memories?user_id=<probe>` 带 `X-API-Key`;200 或 401 都说明 server 就绪(区分鉴权错误),connection refused/timeout 说明没起 |

- `MEMGO_API_KEY` 在 env,直接进入探活。
- server 不可达(local):按本仓库 README 起栈 `make up` + `make bootstrap`
  (seed: setup-status → register → 建 API key),或用 `make run-local` 连外部
  Postgres;再探活。
- server 不可达(remote):向用户要正确的 `MEMGO_BASE_URL`,不要猜。
- Missing key 且 **CI mode**(`MEMGO_INTEGRATE_CI=1`),exit code 2 with the
  name of the missing key。

Never echo key values into `trace.jsonl`。Persist to `.env` only with explicit
user consent, and append `.env` to `.gitignore` if it is not there already.

客户端不需要 `OPENAI_API_KEY`:LLM 与 embedder 由 MemGo server 端配置
(`MEMGO_DEFAULT_LLM_MODEL` / `MEMGO_DEFAULT_EMBEDDER_MODEL`),接入端只发
HTTP,不接触模型配置。

## 5. Goal doc, the hard gate

Write `.memgo-integration/goal.md` and **require user approval before step 6**.

    # MemGo Integration Goal

    **What gets stored:** <one sentence. User utterances? Extracted
    preferences? A specific domain fact like "dietary restrictions"?>

    **When it gets retrieved:** <one sentence. On each user turn? Before a
    specific tool call? At session start?>

    **Why:** <one sentence, the user-visible behavior change. "Assistant
    remembers previous orders across sessions," not "we added memory.">

    **Server:** remote | local  (locked from step 3, do not change)

    **API scope:** user_id=<人> · agent_id=<仓库 slug> · run_id=<会话>。仅这三
    个维度;无 app_id、无 custom_categories。

    **Out of scope:** <anything explicitly excluded: "no graph memory,"
    "no multimodal," "no migration from existing store">

Rules:

- The user must approve explicitly. If they edit the doc, reload and
  re-confirm.
- `goal.md` is the contract the test suite is written against. Never rewrite
  it after step 6 starts.
- Max 3 rejection rounds. On the 4th, exit code 3 with the rejection notes:
  the integration is not well-specified enough to proceed.

## 6. Integration plan, where and how (hard gate)

`goal.md` is what and why. This step produces where and how, and gets explicit
sign-off before any code is written.

Do a **scoped** read of the repo, no wide survey:

- Grep for the LLM call sites that match the goal (`openai.chat.`,
  `anthropic.messages.`, `model.generateContent`, `ChatOpenAI`, `createLLM`,
  Go 里直接调 LLM 的地方)。
- Grep for the user-identity source (`req.user`, `session.user`, `auth()`,
  `ctx.userId`, cookies)。
- Check for conflicts,例如仓库里已有同名 `memgo`/memory 模块。

Then write `.memgo-integration/plan.md`:

    # MemGo Integration Plan

    **Write pattern:** <one sentence, e.g. "After each assistant reply,
    POST {base}/memories with messages=[user_msg, assistant_msg],
    user_id=<source>, agent_id=<repo-slug>.">

    **Read pattern:** <one sentence, e.g. "Before building the LLM prompt,
    POST {base}/search with query=latest_user_msg,
    filters={user_id:<source>, agent_id:<repo-slug>}, top_k=5 and inject
    results as a system message.">

    **User identifier source:** <code path, e.g. `req.auth.userId`,
    `session.user.email`, `ctx.params.user_id`. If none, ask the user.>

    **Session scoping:**
      - user_id: <source>
      - agent_id: <static slug | null>
      - run_id:   <source | null>
    (仅这三个维度;无 app_id、无 custom_categories)

    **Write call site:** `<file:line_range>` inside `<function>`
    **Read call site:**  `<file:line_range>` inside `<function>`

    **HTTP client:** <目标仓库已有的, 如 Go `net/http`, Python
    `requests`/`httpx`, Node `fetch`/`axios`>; 不新增依赖。

    **Preserved behavior:** <list the existing repo behaviors that must
    keep working after this edit, e.g. "existing OpenAI streaming still
    works," "existing Redis session store still used," "existing tests
    still pass unchanged.">

    **Coexistence:** <one bullet per existing system the integration sits
    alongside. Name the files/classes. Example: "The existing
    `agents/memory/storage.py` MemoryStorage class remains untouched and
    keeps its LangGraph SummarizationEvent flow. MemGo is added as a
    parallel long-term-facts store, in a new file, invoked only when
    MEMGO_ENABLED=1 is set.">

    **Feature flag:** <the exact mechanism and the default. Required.
    Example: `env MEMGO_ENABLED=1`, default unset / off; `config.memgo.enabled`,
    default false. With the flag in its default state, the repo must
    behave exactly like `main`.>

    **Sources consulted:** <minimum 2 sources from "Canonical sources" in
    SKILL.md that informed this plan. At least one `integrations/README.md`
    URL or path and one existing `integrations/` 接入实现。Cite the specific
    section or heading.>

    **E2E recipe:** <如何端到端驱动 app。纯库无运行入口则跳过并警告。>

        server:              <"remote" | "local">
        base_url:            <https://memgo.wxget.com | http://localhost:8888>
        api_key_env:         <MEMGO_API_KEY>
        start:               <local: shell command 起 server, 如 make up + make
                              bootstrap; remote: no-op>
        ready_probe:         <one of: url=<URL> status=<code>  /
                              log="<substring to wait for>"  /
                              sleep=<seconds, last resort>>
        write_call:          <command that triggers the MemGo write path
                              exactly once; 60s runtime or less>
        read_call:           <command that triggers the MemGo read path,
                              typically a fresh session / new request>
        read_assert:         <substring, regex, or jsonpath=<expr>=<value>
                              that MUST appear in read_call's output for
                              the E2E to pass. Derived from goal.md's
                              "What gets stored.">

    **Rejected alternatives:** <briefly, 1 or 2 bullets. Patterns the skill
    considered but did not pick, and why. Helps the user decide.>

写入为同步语义:`POST /memories` 直接返回 `results` 即完成,无事件轮询、无
`write_async_wait_ms`。验证即 起 server → 写入 → 搜索 → 断言命中。

Rules:

- Show the user the proposed call sites with 10 lines of context around each
  before asking for approval.
- If no plausible call site exists for either write or read, exit code 5 and
  ask the user to name the files manually. That is the "no fit here" signal,
  do not guess.
- Max 3 rejection rounds on the plan. On the 4th, exit code 5 with the last
  plan and the user's notes.
- If the user edits `plan.md` by hand, reload and re-confirm.

`plan.md`, not `goal.md`, is the contract the subagent implements against in
step 8.

## 7. Tests first (TDD)

The main agent writes failing tests against `goal.md` in the repo's native
test framework:

| Track | Default framework |
|---|---|
| Python | `pytest` |
| TypeScript | `vitest` if detected, else `jest` |
| JavaScript | same |
| Go | `go test` |

Test assertion shapes must match the **OSS API 契约**(`integrations/README.md`
的 `/memories` 与 `/search` 请求/响应):
- 请求 body 形状取自契约,不手搓。写调用点应发起 `POST {base}/memories`,
  body 含 `messages` 数组与正确的 `user_id`/`agent_id`/`run_id`,携带
  `X-API-Key` 头;读调用点应发起 `POST {base}/search`。

Minimum two test files, paths taken from `plan.md` call sites:

- `test_memgo_write.<ext>` asserts `POST /memories` is called at the write call
  site with the right payload shape (`messages` 数组) and the right `user_id`
  source.
- `test_memgo_read.<ext>` asserts `POST /search` runs before the read call site
  and the result is wired into the LLM prompt or response path.

Tests MUST be importable with `MEMGO_API_KEY` unset. This is the design
pressure that forces step 8's lazy HTTP client construction: eager module-level
init hits the network on import and breaks pre-existing test collection when
the key is missing or the server is down.

Run the tests. They **must fail**. If they pass before any implementation, the
tests are wrong. Rewrite them.

## 8. Implementation (subagent, fresh context)

Spawn a subagent with:

- **Inputs**: the repo, `goal.md`, `plan.md`, the two test files, the OSS API
  契约(`integrations/README.md`), and `plan.md` 里记的现成同栈接入实现路径。
- **No access** to the main agent's reasoning trace or scratchpad.
- **System prompt**: use the implementation prompt in
  [`subagent-prompts.md`](subagent-prompts.md) verbatim.

The subagent returns a diff. The main agent reviews it against `plan.md` (the
mechanical contract) and `goal.md` (the intent):

- Approved, apply the diff and commit.
- Rejected, return with specific actionable feedback, not "try again."
- Max 3 review loops. Beyond that, exit code 4 with the last diff and the
  reviewer feedback.

## 9. Commit and handoff

Create branch `memgo-integrate/<short-goal-slug>` and commit in **separate
commits** so reviewers can cherry-pick:

1. `memgo: add gated config`, feature flag / config change(如仅 env 触发可跳过)。
2. `memgo: add integration module`, the new files.
3. `memgo: wire into <call site>`, the call-site edits, still gated.
4. `memgo: add tests`, the new test files.

With `--no-heal`, print the verification 命令(E2E recipe 的 write/read/assert)
and exit. Otherwise proceed to step 10.

## 10. Self-healing loop (default ON, disable with `--no-heal`)

验证两步,全部通过即 done, exit 0:

1. **原生测试套件(flag 开/关)**: 跑仓库测试,断言 `MEMGO_ENABLED=1` 与 unset
   两种状态下全部通过且行为一致(non-invasiveness)。
2. **E2E 冒烟**: 按 `plan.md` 的 E2E recipe — local 先 `start` 起 server →
   `write_call` 触发写入 → `read_call` 触发读取 → 断言 `read_assert` 命中。

Otherwise loop:

1. **Categorize the failing check** and route:
   - `unit_tests`, wiring or assertion fix.
   - `e2e`, server 起栈 / flag 接线 / HTTP 调用形状 fix。
   - **Pre-existing test failure**(flag unset 下原测试就挂),**STOP**. This is a
     non-invasiveness violation. Do NOT attempt to fix it, that breaks
     principle 3. Exit code 6 with a rationale.

2. **Spawn a remediation subagent** with fresh context. Inputs: `plan.md`,
   `goal.md`, the last committed diff, and the relevant log for the failing
   category(测试输出 / E2E 日志)。Use the remediation
   prompt in [`subagent-prompts.md`](subagent-prompts.md) verbatim.

3. **Apply the diff** and commit on the same branch as
   `memgo-heal: <category> attempt <N>`. Do NOT amend earlier commits,
   reviewers need the heal trail.

4. **Re-run both verifications**:
   - pass, done, exit 0.
   - Same check still failing, increment the attempt counter and loop.
   - A *different* check now failing, that is a regression. Revert the heal
     commit (`git revert HEAD --no-edit`), record it in
     `.memgo-integration/heal-trace.md`, exit code 6.

5. **Bounded iterations.** Default 3 attempts per failing category, override
   with `--heal-max N` (hard cap 10). On exhaustion, exit code 6 with the full
   attempt trace: each diff, each scorecard, final log tail.

6. **Post-loop summary** written to `.memgo-integration/heal-trace.md`: which
   category failed, how many attempts, each diff's intent, final status, and
   on success the delta from the initial state to the final one.
