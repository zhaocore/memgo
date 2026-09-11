import type { Command } from 'commander';
import { setAgentMode } from '../../../runtime/state.js';

import { resolveIds } from '../../../application/entity-ids.js';
import { getBackendAndConfig } from '../../context.js';

/**
 * 注册 memory/list 命令，保持参数和输出合同。
 * @param program - 本次装配的 Commander 根命令。
 */
export function registerMemoryList(program: Command): void {
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
    .command('list')
    .description('List memories with optional filters.')
    .option('-u, --user-id <id>', 'Filter by user.')
    .option('--agent-id <id>', 'Filter by agent.')
    .option('--app-id <id>', 'Filter by app.')
    .option('--run-id <id>', 'Filter by run.')
    .option('--page <n>', 'Page number.', (v) => Number.parseInt(v), 1)
    .option('--page-size <n>', 'Results per page.', (v) => Number.parseInt(v), 100)
    .option('--category <name>', 'Filter by category.')
    .option('--after <date>', 'Created after (YYYY-MM-DD).')
    .option('--before <date>', 'Created before (YYYY-MM-DD).')
    .option('--show-expired', 'Include expired memories.', false)
    .option('--latest-only', 'Only return the latest version of each memory.', false)
    .option('-o, --output <format>', 'Output: text, json, table.', 'table')
    .option('--api-key <key>', 'Override API key.')
    .option('--base-url <url>', 'Override API base URL.')
    .addHelpText(
      'after',
      '\nExamples:\n  $ memgo list -u alice\n  $ memgo list --category prefs --after 2024-01-01 -o json'
    )
    .action(async (opts) => {
      const { cmdList } = await import('../../../cli/commands/memory/list.js');
      const isAgent = checkAgentMode();
      const { backend, config } = await getBackendAndConfig(opts.apiKey, opts.baseUrl);
      const ids = resolveIds(config, opts);
      const output = isAgent ? 'agent' : opts.output;
      await cmdList(backend, {
        ...ids,
        page: opts.page,
        pageSize: opts.pageSize,
        category: opts.category,
        after: opts.after,
        before: opts.before,
        showExpired: opts.showExpired,
        latestOnly: opts.latestOnly,
        output,
      });
    });
}
