import type { JsonObject } from '../../../backend/json.js';
import type { Backend } from '../../../backend/types.js';
import { printError, printInfo, timedStatus } from '../../../output/branding.js';
import {
  formatAgentEnvelope,
  formatMemoriesTable,
  formatMemoriesText,
  printResultSummary,
} from '../../../output/format.js';
import { exit } from '../../../runtime/exit.js';
import { setCurrentCommand } from '../../../runtime/state.js';

/**
 * 按实体范围与分页条件列出记忆。
 * @param backend - 已装配的后端连接器。
 * @param opts - 本次操作选项。
 * @param opts.userId - 用户 ID。
 * @param opts.agentId - 代理 ID。
 * @param opts.appId - 应用 ID。
 * @param opts.runId - 会话 ID。
 * @param opts.page - 分页页码。
 * @param opts.pageSize - 每页条目数。
 * @param opts.category - 分类筛选条件。
 * @param opts.after - 创建日期下界。
 * @param opts.before - 创建日期上界。
 * @param opts.showExpired - 包含已过期记忆的选项。
 * @param opts.latestOnly - 只返回最新版本的选项。
 * @param opts.output - 终端输出格式。
 */
export async function cmdList(
  backend: Backend,
  opts: {
    userId?: string;
    agentId?: string;
    appId?: string;
    runId?: string;
    page: number;
    pageSize: number;
    category?: string;
    after?: string;
    before?: string;
    showExpired?: boolean;
    latestOnly?: boolean;
    output: string;
  }
): Promise<void> {
  setCurrentCommand('list');
  if (opts.pageSize < 1) {
    printError('--page-size must be >= 1.');
    exit(1);
  }
  if (opts.page < 1) {
    printError('--page must be >= 1.');
    exit(1);
  }

  const start = performance.now();
  let results: JsonObject[];
  try {
    results = await timedStatus('Listing memories...', async () => {
      return backend.listMemories({
        userId: opts.userId,
        agentId: opts.agentId,
        appId: opts.appId,
        runId: opts.runId,
        page: opts.page,
        pageSize: opts.pageSize,
        category: opts.category,
        after: opts.after,
        before: opts.before,
        showExpired: opts.showExpired,
        latestOnly: opts.latestOnly,
      });
    });
  } catch (e) {
    printError(e instanceof Error ? e.message : String(e));
    exit(1);
  }
  const elapsed = (performance.now() - start) / 1000;

  if (opts.output === 'quiet') return;

  if (opts.output === 'agent' || opts.output === 'json') {
    const scope: Record<string, string | undefined> = {
      user_id: opts.userId,
      agent_id: opts.agentId,
      app_id: opts.appId,
      run_id: opts.runId,
    };
    formatAgentEnvelope({
      command: 'list',
      data: results,
      scope,
      count: results.length,
      durationMs: Math.round(elapsed * 1000),
    });
  } else if (opts.output === 'table') {
    if (results.length > 0) {
      formatMemoriesTable(results, {});
      printResultSummary({
        count: results.length,
        durationSecs: elapsed,
        page: opts.page,
        scopeIds: { user_id: opts.userId, agent_id: opts.agentId },
      });
    } else {
      console.log();
      printInfo('No memories found.');
      console.log();
    }
  } else {
    if (results.length > 0) {
      formatMemoriesText(results, 'memories');
      printResultSummary({
        count: results.length,
        durationSecs: elapsed,
        page: opts.page,
        scopeIds: { user_id: opts.userId, agent_id: opts.agentId },
      });
    } else {
      console.log();
      printInfo('No memories found.');
      console.log();
    }
  }
}
