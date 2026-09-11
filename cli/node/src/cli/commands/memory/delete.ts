import type { JsonObject } from '../../../backend/json.js';
import type { Backend } from '../../../backend/types.js';
import { printError, printInfo, printSuccess, timedStatus } from '../../../output/branding.js';
import { formatAgentEnvelope, formatJson } from '../../../output/format.js';
import { exit } from '../../../runtime/exit.js';
import { setCurrentCommand } from '../../../runtime/state.js';

/**
 * 确认后删除单条记忆，预演时仅显示目标。
 * @param backend - 已装配的后端连接器。
 * @param memoryId - 目标记忆 ID。
 * @param opts - 本次操作选项。
 * @param opts.output - 终端输出格式。
 * @param opts.dryRun - 仅展示删除目标，不执行删除。
 * @param opts.force - 跳过已有配置或删除操作的交互确认。
 * @param opts.deleteLinked - 同时删除关联记忆的选项。
 */
export async function cmdDelete(
  backend: Backend,
  memoryId: string,
  opts: {
    output: string;
    dryRun?: boolean;
    force?: boolean;
    deleteLinked?: boolean;
  }
): Promise<void> {
  setCurrentCommand('delete');
  if (opts.dryRun) {
    let mem: JsonObject;
    try {
      mem = await backend.get(memoryId);
    } catch (e) {
      printError(e instanceof Error ? e.message : String(e));
      exit(1);
    }
    const text = (mem.memory ?? mem.text ?? '') as string;
    printInfo(`Would delete memory ${memoryId.slice(0, 8)}: ${text}`);
    printInfo('No changes made.');
    return;
  }

  const start = performance.now();
  let result: JsonObject;
  try {
    result = await timedStatus('Deleting...', async () => {
      return backend.delete(memoryId, { deleteLinked: opts.deleteLinked });
    });
  } catch (e) {
    printError(e instanceof Error ? e.message : String(e));
    exit(1);
  }
  const elapsed = (performance.now() - start) / 1000;

  if (opts.output === 'agent') {
    formatAgentEnvelope({
      command: 'delete',
      data: { id: memoryId, deleted: true },
      durationMs: Math.round(elapsed * 1000),
    });
  } else if (opts.output === 'json') {
    formatJson(result);
  } else if (opts.output !== 'quiet') {
    printSuccess(`Memory ${memoryId.slice(0, 8)} deleted (${elapsed.toFixed(2)}s)`);
  }
}
