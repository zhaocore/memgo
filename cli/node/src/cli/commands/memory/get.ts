import type { JsonObject } from '../../../backend/json.js';
import type { Backend } from '../../../backend/types.js';
import { printError, timedStatus } from '../../../output/branding.js';
import { formatAgentEnvelope, formatSingleMemory } from '../../../output/format.js';
import { exit } from '../../../runtime/exit.js';
import { setCurrentCommand } from '../../../runtime/state.js';

/**
 * 获取指定记忆并按输出格式展示。
 * @param backend - 已装配的后端连接器。
 * @param memoryId - 目标记忆 ID。
 * @param opts - 本次操作选项。
 * @param opts.output - 终端输出格式。
 */
export async function cmdGet(
  backend: Backend,
  memoryId: string,
  opts: { output: string }
): Promise<void> {
  setCurrentCommand('get');
  let result: JsonObject;
  try {
    result = await timedStatus('Fetching memory...', async () => {
      return backend.get(memoryId);
    });
  } catch (e) {
    printError(e instanceof Error ? e.message : String(e));
    exit(1);
  }

  if (opts.output === 'agent') {
    formatAgentEnvelope({ command: 'get', data: result });
  } else {
    formatSingleMemory(result, opts.output);
  }
}
