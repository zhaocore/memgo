import type { Command } from 'commander';
import { setAgentMode } from '../../../runtime/state.js';

import { getBackendOnly } from '../../context.js';

/**
 * 注册 memory/get 命令，保持参数和输出合同。
 * @param program - 本次装配的 Commander 根命令。
 */
export function registerMemoryGet(program: Command): void {
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
    .command('get <memoryId>')
    .description('Get a specific memory by ID.')
    .option('-o, --output <format>', 'Output: text, json.', 'text')
    .option('--api-key <key>', 'Override API key.')
    .option('--base-url <url>', 'Override API base URL.')
    .addHelpText(
      'after',
      '\nExamples:\n  $ memgo get abc-123-def-456\n  $ memgo get abc-123-def-456 -o json'
    )
    .action(async (memoryId, opts) => {
      const { cmdGet } = await import('../../../cli/commands/memory/get.js');
      const isAgent = checkAgentMode();
      const backend = await getBackendOnly(opts.apiKey, opts.baseUrl);
      const output = isAgent ? 'agent' : opts.output;
      await cmdGet(backend, memoryId, { output });
    });
}
