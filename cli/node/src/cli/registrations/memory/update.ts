import fs from 'node:fs';
import type { Command } from 'commander';
import { setAgentMode, stdinIsPiped } from '../../../runtime/state.js';

import { getBackendOnly } from '../../context.js';

/**
 * 注册 memory/update 命令，保持参数和输出合同。
 * @param program - 本次装配的 Commander 根命令。
 */
export function registerMemoryUpdate(program: Command): void {
  /**
   * 读取全局选项并同步本次调用的代理输出模式。
   * @returns 本次调用是否采用代理输出模式。
   */
  function checkAgentMode(): boolean {
    const value = !!(program.opts().json || program.opts().agent);
    setAgentMode(value);
    return value;
  }

  program
    .command('update <memoryId> [text]')
    .description("Update a memory's text or metadata.")
    .option('-m, --metadata <json>', 'Update metadata (JSON).')
    .option('--expires <date>', 'Expiration date (YYYY-MM-DD).')
    .option('--timestamp <unix>', 'Unix timestamp for the memory.', (v) => Number.parseInt(v))
    .option('-o, --output <format>', 'Output: text, json, quiet.', 'text')
    .option('--api-key <key>', 'Override API key.')
    .option('--base-url <url>', 'Override API base URL.')
    .addHelpText(
      'after',
      '\nExamples:\n  $ memgo update abc-123 "new text"\n  $ memgo update abc-123 --metadata \'{"key":"val"}\'\n  $ echo "new text" | memgo update abc-123'
    )
    .action(async (memoryId, text, opts) => {
      let resolvedText = text;
      if (!resolvedText && stdinIsPiped()) {
        resolvedText = fs.readFileSync(0, 'utf-8').trim();
      }
      const { cmdUpdate } = await import('../../../cli/commands/memory/update.js');
      const isAgent = checkAgentMode();
      const backend = await getBackendOnly(opts.apiKey, opts.baseUrl);
      const output = isAgent ? 'agent' : opts.output;
      await cmdUpdate(backend, memoryId, resolvedText, {
        metadata: opts.metadata,
        expires: opts.expires,
        timestamp: opts.timestamp,
        output,
      });
    });
}
