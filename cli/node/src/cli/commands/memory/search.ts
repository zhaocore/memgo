import { parseObject } from '../../../application/memory/input.js';
import type { JsonObject } from '../../../backend/json.js';
import type { Backend } from '../../../backend/types.js';
import { printError, printInfo, timedStatus } from '../../../output/branding.js';
import {
  formatAgentEnvelope,
  formatJson,
  formatMemoriesTable,
  formatMemoriesText,
  printResultSummary,
} from '../../../output/format.js';
import { exit } from '../../../runtime/exit.js';
import { setCurrentCommand } from '../../../runtime/state.js';

/**
 * 按查询与筛选条件搜索记忆并展示结果。
 * @param backend - 已装配的后端连接器。
 * @param query - 记忆搜索文本。
 * @param opts - 本次操作选项。
 * @param opts.userId - 用户 ID。
 * @param opts.agentId - 代理 ID。
 * @param opts.appId - 应用 ID。
 * @param opts.runId - 会话 ID。
 * @param opts.topK - 最大搜索结果数。
 * @param opts.threshold - 搜索匹配阈值。
 * @param opts.rerank - 启用结果重排的选项。
 * @param opts.keyword - 启用关键词搜索的选项。
 * @param opts.filterJson - 高级筛选条件的 JSON 文本。
 * @param opts.fields - 需要返回的字段。
 * @param opts.showExpired - 包含已过期记忆的选项。
 * @param opts.referenceDate - 搜索使用的参考日期。
 * @param opts.latestOnly - 只返回最新版本的选项。
 * @param opts.output - 终端输出格式。
 */
export async function cmdSearch(
  backend: Backend,
  query: string | undefined,
  opts: {
    userId?: string;
    agentId?: string;
    appId?: string;
    runId?: string;
    topK: number;
    threshold: number;
    rerank: boolean;
    keyword: boolean;
    filterJson?: string;
    fields?: string;
    showExpired?: boolean;
    referenceDate?: string;
    latestOnly?: boolean;
    output: string;
  }
): Promise<void> {
  setCurrentCommand('search');
  if (!query) {
    printError('No query provided. Pass a query argument or pipe via stdin.');
    exit(1);
  }

  let filters: JsonObject | undefined;
  if (opts.filterJson) {
    try {
      filters = parseObject(opts.filterJson);
    } catch {
      printError('Invalid JSON in --filter.');
      exit(1);
    }
  }

  const fieldList = opts.fields ? opts.fields.split(',').map((f) => f.trim()) : undefined;

  if (opts.topK < 1) {
    printError('--top-k must be >= 1.');
    exit(1);
  }
  if (opts.threshold < 0 || opts.threshold > 1) {
    printError('--threshold must be between 0.0 and 1.0.');
    exit(1);
  }

  const start = performance.now();
  let results: JsonObject[];
  try {
    results = await timedStatus('Searching memories...', async () => {
      // biome-ignore lint/style/noNonNullAssertion: 已在缺少参数时终止命令。
      return backend.search(query!, {
        userId: opts.userId,
        agentId: opts.agentId,
        appId: opts.appId,
        runId: opts.runId,
        topK: opts.topK,
        threshold: opts.threshold,
        rerank: opts.rerank,
        keyword: opts.keyword,
        filters,
        fields: fieldList,
        showExpired: opts.showExpired,
        referenceDate: opts.referenceDate,
        latestOnly: opts.latestOnly,
      });
    });
  } catch (e) {
    printError(e instanceof Error ? e.message : String(e));
    exit(1);
  }
  const elapsed = (performance.now() - start) / 1000;

  if (opts.output === 'quiet') return;

  if (opts.output === 'agent') {
    const scope: Record<string, string | undefined> = {
      user_id: opts.userId,
      agent_id: opts.agentId,
      app_id: opts.appId,
      run_id: opts.runId,
    };
    formatAgentEnvelope({
      command: 'search',
      data: results,
      scope,
      count: results.length,
      durationMs: Math.round(elapsed * 1000),
    });
    return;
  }

  if (opts.output === 'json') {
    formatJson(results);
  } else if (opts.output === 'table') {
    if (results.length > 0) {
      formatMemoriesTable(results, { showScore: true });
      printResultSummary({
        count: results.length,
        durationSecs: elapsed,
        scopeIds: { user_id: opts.userId, agent_id: opts.agentId },
      });
    } else {
      console.log();
      printInfo('No memories found matching your query.');
      console.log();
    }
  } else {
    if (results.length > 0) {
      formatMemoriesText(results, 'memories');
      printResultSummary({
        count: results.length,
        durationSecs: elapsed,
        scopeIds: { user_id: opts.userId, agent_id: opts.agentId },
      });
    } else {
      console.log();
      printInfo('No memories found matching your query.');
      console.log();
    }
  }
}
