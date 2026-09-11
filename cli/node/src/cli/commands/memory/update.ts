import { _validateExpires } from '../../../application/memory/expiration.js';
import { parseObject } from '../../../application/memory/input.js';
import type { JsonObject } from '../../../backend/json.js';
import type { Backend } from '../../../backend/types.js';
import { printError, printSuccess, timedStatus } from '../../../output/branding.js';
import { formatAgentEnvelope, formatJson } from '../../../output/format.js';
import { exit } from '../../../runtime/exit.js';
import { setCurrentCommand } from '../../../runtime/state.js';

/**
 * 校验更新字段后修改指定记忆。
 * @param backend - 已装配的后端连接器。
 * @param memoryId - 目标记忆 ID。
 * @param text - 待解析或提交的文本。
 * @param opts - 本次操作选项。
 * @param opts.metadata - 记忆元数据的 JSON 文本。
 * @param opts.expires - 记忆到期日期，格式为 YYYY-MM-DD。
 * @param opts.timestamp - 业务时间戳。
 * @param opts.output - 终端输出格式。
 */
export async function cmdUpdate(
  backend: Backend,
  memoryId: string,
  text: string | undefined,
  opts: {
    metadata?: string;
    expires?: string;
    timestamp?: number;
    output: string;
  }
): Promise<void> {
  setCurrentCommand('update');
  let meta: JsonObject | undefined;
  if (opts.metadata) {
    try {
      meta = parseObject(opts.metadata);
    } catch {
      printError('Invalid JSON in --metadata.');
      exit(1);
    }
  }

  if (opts.expires) _validateExpires(opts.expires);

  const start = performance.now();
  let result: JsonObject;
  try {
    result = await timedStatus('Updating memory...', async () => {
      return backend.update(memoryId, text, meta, {
        expirationDate: opts.expires,
        timestamp: opts.timestamp,
      });
    });
  } catch (e) {
    printError(e instanceof Error ? e.message : String(e));
    exit(1);
  }
  const elapsed = (performance.now() - start) / 1000;

  if (opts.output === 'agent') {
    formatAgentEnvelope({
      command: 'update',
      data: result,
      durationMs: Math.round(elapsed * 1000),
    });
  } else if (opts.output === 'json') {
    formatJson(result);
  } else if (opts.output !== 'quiet') {
    printSuccess(`Memory ${memoryId.slice(0, 8)} updated (${elapsed.toFixed(2)}s)`);
  }
}
