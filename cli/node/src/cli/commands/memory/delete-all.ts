import type { JsonObject } from '../../../backend/json.js';
import type { Backend } from '../../../backend/types.js';
import { printError, printInfo, printSuccess, timedStatus } from '../../../output/branding.js';
import { formatAgentEnvelope, formatJson } from '../../../output/format.js';
import { exit } from '../../../runtime/exit.js';
import { setCurrentCommand } from '../../../runtime/state.js';

/**
 * 校验删除范围和确认条件后批量删除记忆。
 * @param backend - 已装配的后端连接器。
 * @param opts - 本次操作选项。
 * @param opts.force - 跳过已有配置或删除操作的交互确认。
 * @param opts.dryRun - 仅展示删除目标，不执行删除。
 * @param opts.all - 请求项目范围的删除操作。
 * @param opts.userId - 用户 ID。
 * @param opts.agentId - 代理 ID。
 * @param opts.appId - 应用 ID。
 * @param opts.runId - 会话 ID。
 * @param opts.output - 终端输出格式。
 */
export async function cmdDeleteAll(
  backend: Backend,
  opts: {
    force: boolean;
    dryRun?: boolean;
    all?: boolean;
    userId?: string;
    agentId?: string;
    appId?: string;
    runId?: string;
    output: string;
  }
): Promise<void> {
  setCurrentCommand('delete-all');
  if (opts.all && opts.dryRun) {
    printError('Project-wide --dry-run is not supported; no memories were deleted.');
    exit(1);
  }
  const { isAgentMode } = await import('../../../runtime/state.js');
  if (isAgentMode() && !opts.force) {
    printError('Destructive operation requires --force in agent mode.');
    exit(1);
  }
  if (opts.all) {
    // 项目级删除使用通配实体标识。
    // 历史行为：项目级删除不支持预先计数；相关语义必须显式验收。

    if (!opts.force) {
      const readline = await import('node:readline');
      const rl = readline.createInterface({
        input: process.stdin,
        output: process.stdout,
      });
      const answer = await new Promise<string>((resolve) => {
        rl.question(
          '\n  \u26a0  Delete ALL memories across the ENTIRE project? This cannot be undone. [y/N] ',
          resolve
        );
      });
      rl.close();
      if (answer.toLowerCase() !== 'y') {
        printInfo('Cancelled.');
        exit(0);
      }
    }

    const start = performance.now();
    let result: JsonObject;
    try {
      result = await timedStatus('Deleting all memories project-wide...', async () => {
        return backend.delete(undefined, {
          all: true,
          userId: '*',
          agentId: '*',
          appId: '*',
          runId: '*',
        });
      });
    } catch (e) {
      printError(e instanceof Error ? e.message : String(e));
      exit(1);
    }
    const elapsed = (performance.now() - start) / 1000;

    if (opts.output === 'agent') {
      formatAgentEnvelope({
        command: 'delete-all',
        data: result,
        durationMs: Math.round(elapsed * 1000),
      });
    } else if (opts.output === 'json') {
      formatJson(result);
    } else if (opts.output !== 'quiet') {
      if (result.message) {
        printInfo('Deletion started. Memories will be removed in the background.');
      } else {
        printSuccess(`All project memories deleted (${elapsed.toFixed(2)}s)`);
      }
    }
    return;
  }

  if (opts.dryRun) {
    let memories: JsonObject[];
    try {
      memories = await backend.listMemories({
        userId: opts.userId,
        agentId: opts.agentId,
        appId: opts.appId,
        runId: opts.runId,
      });
    } catch (e) {
      printError(e instanceof Error ? e.message : String(e));
      exit(1);
    }
    printInfo(`Would delete ${memories.length} memories.`);
    printInfo('No changes made.');
    return;
  }

  if (!opts.force) {
    const scopeParts: string[] = [];
    if (opts.userId) scopeParts.push(`user=${opts.userId}`);
    if (opts.agentId) scopeParts.push(`agent=${opts.agentId}`);
    if (opts.appId) scopeParts.push(`app=${opts.appId}`);
    if (opts.runId) scopeParts.push(`run=${opts.runId}`);
    const scope = scopeParts.length > 0 ? scopeParts.join(', ') : 'ALL entities';

    const readline = await import('node:readline');
    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
    });
    const answer = await new Promise<string>((resolve) => {
      rl.question(
        `\n  \u26a0  Delete ALL memories for ${scope}? This cannot be undone. [y/N] `,
        resolve
      );
    });
    rl.close();
    if (answer.toLowerCase() !== 'y') {
      printInfo('Cancelled.');
      exit(0);
    }
  }

  const start = performance.now();
  let result: JsonObject;
  try {
    result = await timedStatus('Deleting all memories...', async () => {
      return backend.delete(undefined, {
        all: true,
        userId: opts.userId,
        agentId: opts.agentId,
        appId: opts.appId,
        runId: opts.runId,
      });
    });
  } catch (e) {
    printError(e instanceof Error ? e.message : String(e));
    exit(1);
  }
  const elapsed = (performance.now() - start) / 1000;

  if (opts.output === 'agent') {
    formatAgentEnvelope({
      command: 'delete-all',
      data: result,
      durationMs: Math.round(elapsed * 1000),
    });
  } else if (opts.output === 'json') {
    formatJson(result);
  } else if (opts.output !== 'quiet') {
    if (result.message) {
      printInfo('Deletion started. Memories will be removed in the background.');
    } else {
      printSuccess(`All matching memories deleted (${elapsed.toFixed(2)}s)`);
    }
  }
}
