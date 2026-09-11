import type { JsonObject } from '../../backend/json.js';
import { exit } from '../../runtime/exit.js';
/** 实体管理命令。 */

import readline from 'node:readline';
import Table from 'cli-table3';
import type { Backend } from '../../backend/types.js';
import { colors, printError, printInfo, printSuccess, timedStatus } from '../../output/branding.js';
import { formatAgentEnvelope, formatJson } from '../../output/format.js';
import { setCurrentCommand } from '../../runtime/state.js';

const { accent, dim } = colors;

const VALID_TYPES = new Set(['users', 'agents', 'apps', 'runs']);

/**
 * 列出指定类型的实体并按选定格式输出。
 * @param backend - 已装配的后端连接器。
 * @param entityType - 需要列出的实体类型。
 * @param opts - 本次操作选项。
 * @param opts.output - 终端输出格式。
 */
export async function cmdEntitiesList(
  backend: Backend,
  entityType: string,
  opts: { output: string }
): Promise<void> {
  setCurrentCommand('entity list');
  if (!VALID_TYPES.has(entityType)) {
    printError(`Invalid entity type: ${entityType}. Use: ${[...VALID_TYPES].join(', ')}`);
    exit(1);
  }

  const start = performance.now();
  let results: JsonObject[];
  try {
    results = await timedStatus(`Fetching ${entityType}...`, async () => {
      return backend.entities(entityType);
    });
  } catch (e) {
    printError(
      e instanceof Error ? e.message : String(e),
      'This feature may require the memgo Platform.'
    );
    exit(1);
  }
  const elapsed = (performance.now() - start) / 1000;

  if (opts.output === 'agent' || opts.output === 'json') {
    formatAgentEnvelope({
      command: 'entity list',
      data: results,
      count: results.length,
      durationMs: Math.round(elapsed * 1000),
    });
    return;
  }

  if (!results.length) {
    printInfo(`No ${entityType} found.`);
    return;
  }

  const table = new Table({
    head: [accent('Name / ID'), accent('Created')],
    style: { head: [], border: [] },
  });

  for (const entity of results) {
    const name = String(entity.name ?? entity.id ?? '—');
    const created = String(entity.created_at ?? '—').slice(0, 10);
    table.push([name, created]);
  }

  console.log();
  console.log(table.toString());
  console.log(`  ${dim(`${results.length} ${entityType} (${elapsed.toFixed(2)}s)`)}`);
  console.log();
}

/**
 * 确认删除范围后删除实体及其记忆，预演时仅展示范围。
 * @param backend - 已装配的后端连接器。
 * @param opts - 本次操作选项。
 * @param opts.userId - 用户 ID。
 * @param opts.agentId - 代理 ID。
 * @param opts.appId - 应用 ID。
 * @param opts.runId - 会话 ID。
 * @param opts.dryRun - 仅展示删除目标，不执行删除。
 * @param opts.force - 跳过已有配置或删除操作的交互确认。
 * @param opts.output - 终端输出格式。
 */
export async function cmdEntitiesDelete(
  backend: Backend,
  opts: {
    userId?: string;
    agentId?: string;
    appId?: string;
    runId?: string;
    dryRun?: boolean;
    force: boolean;
    output: string;
  }
): Promise<void> {
  setCurrentCommand('entity delete');
  const { isAgentMode } = await import('../../runtime/state.js');
  if (isAgentMode() && !opts.force) {
    printError('Destructive operation requires --force in agent mode.');
    exit(1);
  }
  if (!opts.userId && !opts.agentId && !opts.appId && !opts.runId) {
    printError('Provide at least one of --user-id, --agent-id, --app-id, --run-id.');
    exit(1);
  }

  const scopeParts: string[] = [];
  if (opts.userId) scopeParts.push(`user=${opts.userId}`);
  if (opts.agentId) scopeParts.push(`agent=${opts.agentId}`);
  if (opts.appId) scopeParts.push(`app=${opts.appId}`);
  if (opts.runId) scopeParts.push(`run=${opts.runId}`);
  const scope = scopeParts.join(', ');

  if (opts.dryRun) {
    printInfo(`Would delete entity ${scope} and all its memories.`);
    printInfo('No changes made.');
    return;
  }

  if (!opts.force) {
    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
    });
    const answer = await new Promise<string>((resolve) => {
      rl.question(
        `\n  \u26a0  Delete entity ${scope} AND all its memories? This cannot be undone. [y/N] `,
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
    result = await timedStatus('Deleting entity...', async () => {
      return backend.deleteEntities({
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
      command: 'entity delete',
      data: { deleted: true },
      durationMs: Math.round(elapsed * 1000),
    });
  } else if (opts.output === 'json') {
    formatJson(result);
  } else if (opts.output !== 'quiet') {
    printSuccess(`Entity deleted with all memories (${elapsed.toFixed(2)}s)`);
  }
}
