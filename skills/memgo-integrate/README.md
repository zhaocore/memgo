# memgo-integrate — Pipeline Skill

Wire [MemGo](../../README.md) into an existing repository end-to-end, using a goal-driven, test-first pipeline.

> **This is a pipeline skill, not a reference skill.** Invoke it as `/memgo-integrate` when you want your assistant to do the work of integrating MemGo into a target repo. For day-to-day CLI usage, use the [`memgo` CLI](../../README.md#cli-memgo) directly.
>
> MemGo 是自托管 OSS 记忆 server,只有 OSS API 面,无托管平台。接入一律直连
> OSS API(HTTP),契约见 [OSS API 契约](../../integrations/README.md)。

## What This Skill Does

When invoked, your assistant will:

- **Detect** the target repo's language and stack automatically
- **Select** the MemGo server — remote (`https://memgo.wxget.com`) or local
  (`make up`, `:8888`)
- **Write failing tests first** — no implementation until tests exist
- **Keep the integration additive and feature-flagged** — existing behavior stays byte-for-byte identical when the flag is unset
- **Produce a local feature branch** (`memgo-integrate/...`) and a `.memgo-integration/` directory of artifacts (`goal.md`, `plan.md`, `product.json`)
- **Verify in-skill**: 起 server → 写入 → 搜索 → 断言命中

## When to Use

Trigger phrases:

- "Integrate MemGo into this repo"
- "Add MemGo to my project"
- "Wire MemGo into `<repo>`"
- "How do I add memory to an existing project?"

Do **not** use this skill for general CLI usage (use the `memgo` CLI directly).

## Installation

本 skill 随 MemGo 仓库分发,目录 `skills/memgo-integrate/`:

```bash
# 本地仓库即已就位; 需要拷到其他 agent 环境时, 整目录复制即可
cp -r skills/memgo-integrate /path/to/your/agent/skills/
```

### Prerequisites

- 一个可达的 MemGo server: 远端 `https://memgo.wxget.com`, 或本地
  `make up && make bootstrap`(见仓库 [README](../../README.md))
- `MEMGO_API_KEY`(必填; 默认 Base URL `https://memgo.wxget.com`,
  `MEMGO_BASE_URL` 覆盖)
- 目标仓库可检出语言(Python / Node / TS / Go ...)且有后端
- 目标仓库默认分支干净工作树

## Workflow

```
/memgo-integrate          →  建 memgo-integrate/<slug> 分支,
                             写 .memgo-integration/ artifacts,
                             对着失败测试实现,
                             验证(内置): 原生测试(flag on/off)
                             + server 起栈 → 写入 → 搜索 → 断言命中
```

## Links

- [MemGo 仓库](../../README.md)
- [OSS API 契约](../../integrations/README.md)
- [Server 环境变量](../../README.md#环境变量表go-server)

## License

Apache-2.0
