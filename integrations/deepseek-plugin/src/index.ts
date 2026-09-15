/**
 * deepseek-plugin: MemGo 长期记忆, 作为 DeepSeek Harness (Cordis) 原生插件。
 *
 * 注册两个 agent 可调工具, 由直连 MemGo OSS server 的内置客户端驱动:
 *   - `search_memory` 召回与查询相关的事实
 *   - `add_memory` 为后续会话存储一条事实
 *
 * 插件是导出 `apply(ctx, config)` 的 Cordis 模块。声明 `inject = ['tools']`
 * 会挂起插件直到 harness 工具注册表就绪; 通过 `ctx.tools.register(...)`
 * 注册的工具在插件卸载时自动注销(Cordis 可逆副作用)。
 *
 * 与托管平台版差异: 不再依赖托管 SDK(替换为 src/client.ts), 同步写入、
 * X-API-Key 鉴权、无平台事件轮询与 custom_categories。
 */
import type { Context } from "@deepseek-ai/cordis";
import { defineTool } from "@deepseek-ai/dsh-tools";
import { MemoryClient } from "./client.ts";
import { formatMemoryList, formatAddResult } from "./formatting.ts";
import { truncateOutput } from "./output.ts";
import { resolveSearchFilters, resolveAddParams } from "./scoping.ts";
import { captureEvent, errorKind } from "./telemetry.ts";

export const name = "memgo";
export const inject = ["tools"];

const DEFAULT_SEARCH_LIMIT = 10;

export interface Config {
  /** MemGo API key。缺省读 MEMGO_API_KEY 环境变量。 */
  apiKey?: string;
  /** 拥有这些记忆的默认实体(MemGo user 范围)。 */
  userId: string;
  /** MemGo OSS server 根地址; 缺省读 MEMGO_BASE_URL, 再回退到默认地址。 */
  host?: string;
}

// 两个工具都返回单条文本字符串, 渲染一致, 抽成共享声明避免逐工具重复。
const textOutput = {
  schema: { type: "string" } as const,
  render: (_args: unknown, value: string) => [
    { type: "text" as const, text: value },
  ],
};

// 两个工具共享的可选单次调用范围参数。一次 harness 安装可为多个实体服务,
// 模型可用这些参数在单次调用中覆盖挂载默认值(见 scoping.ts)。
const scopeParams = {
  userId: {
    type: "string",
    description:
      "Entity that owns the memory. Defaults to the plugin's configured userId; set this only to read or write another user's memories.",
  },
  agentId: {
    type: "string",
    description: "Optional agent scope, to partition memories by agent.",
  },
  runId: {
    type: "string",
    description: "Optional run/session scope, to partition memories by session.",
  },
} as const;

export function apply(ctx: Context, config: Config): void {
  const apiKey = config.apiKey ?? process.env.MEMGO_API_KEY;
  if (!apiKey) {
    throw new Error("deepseek-plugin: 请设置 config.apiKey 或 MEMGO_API_KEY 环境变量");
  }
  const userId = config.userId;
  if (!userId) {
    throw new Error("deepseek-plugin: config.userId 为必填");
  }

  const client = new MemoryClient({
    apiKey,
    userId,
    ...(config.host ? { host: config.host } : {}),
  });

  captureEvent("deepseek.plugin.mounted", { has_host: Boolean(config.host) }, client);

  // 召回。OSS 的 search 要求范围放在 `filters` 内。
  ctx.tools.register(
    defineTool({
      name: "search_memory",
      description:
        "Search the user's long-term MemGo memory for facts relevant to a query. Use proactively before answering anything that may depend on what the user told you earlier.",
      parameters: {
        query: { type: "string", description: "What to recall.", required: true },
        limit: {
          type: "integer",
          description: `Max results to return (default ${DEFAULT_SEARCH_LIMIT}).`,
        },
        ...scopeParams,
      },
      output: textOutput,
      async execute({ query, limit, userId: u, agentId, runId }) {
        const filters = resolveSearchFilters({ userId: u, agentId, runId }, userId);
        const topK = limit && limit > 0 ? limit : DEFAULT_SEARCH_LIMIT;
        const started = Date.now();
        try {
          const results = await client.search(query, filters, topK);
          captureEvent(
            "deepseek.tool.search_memory",
            {
              success: true,
              duration_ms: Date.now() - started,
              top_k: topK,
              result_count: results.length,
              query_chars: query.length,
              scope_overridden: Boolean(u && u !== userId),
              has_agent_id: Boolean(agentId),
              has_run_id: Boolean(runId),
            },
            client,
          );
          return truncateOutput(formatMemoryList(results));
        } catch (err) {
          captureEvent(
            "deepseek.tool.search_memory",
            {
              success: false,
              duration_ms: Date.now() - started,
              top_k: topK,
              error_kind: errorKind(err),
            },
            client,
          );
          return `search_memory failed: ${err instanceof Error ? err.message : String(err)}`;
        }
      },
    }),
  );

  // 写入。OSS 同步返回提取后的记忆, 写入即完成, 返回后可立即搜索。
  ctx.tools.register(
    defineTool({
      name: "add_memory",
      description:
        "Store a fact in the user's long-term MemGo memory for later sessions. The write is synchronous: the fact is searchable once this tool returns.",
      parameters: {
        text: { type: "string", description: "The fact to remember.", required: true },
        ...scopeParams,
      },
      output: textOutput,
      async execute({ text, userId: u, agentId, runId }) {
        const addParams = resolveAddParams({ userId: u, agentId, runId }, userId);
        const started = Date.now();
        try {
          const results = await client.add(text, addParams);
          captureEvent(
            "deepseek.tool.add_memory",
            {
              success: true,
              duration_ms: Date.now() - started,
              text_chars: text.length,
              memory_count: results.length,
              scope_overridden: Boolean(u && u !== userId),
              has_agent_id: Boolean(agentId),
              has_run_id: Boolean(runId),
            },
            client,
          );
          return truncateOutput(formatAddResult(results));
        } catch (err) {
          captureEvent(
            "deepseek.tool.add_memory",
            {
              success: false,
              duration_ms: Date.now() - started,
              text_chars: text.length,
              error_kind: errorKind(err),
            },
            client,
          );
          return `add_memory failed: ${err instanceof Error ? err.message : String(err)}`;
        }
      },
    }),
  );
}
