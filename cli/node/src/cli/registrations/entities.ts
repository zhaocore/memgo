import type { Command } from 'commander';
import { setAgentMode } from '../../runtime/state.js';
import { richFormatHelp } from '../help.js';

import { getBackendOnly } from '../context.js';

/**
 * 注册 entities 命令，保持参数和输出合同。
 * @param program - 本次装配的 Commander 根命令。
 */
export function registerEntities(program: Command): void {
  /**
   * 读取全局选项并同步本次调用的代理输出模式。
   * @returns 本次调用是否采用代理输出模式。
   */
  function checkAgentMode(): boolean {
    const value = !!(program.opts().json || program.opts().agent);
    setAgentMode(value);
    return value;
  }

  const entityCmd = program
    .command('entity')
    .description('Manage entities.')
    .addHelpCommand(false)
    .configureHelp({ formatHelp: richFormatHelp });

  entityCmd
    .command('list <entityType>')
    .description('List all entities of a given type.')
    .option('-o, --output <format>', 'Output: table, json.', 'table')
    .option('--api-key <key>', 'Override API key.')
    .option('--base-url <url>', 'Override API base URL.')
    .addHelpText(
      'after',
      '\nExamples:\n  $ memgo entity list users\n  $ memgo entity list agents -o json'
    )
    .action(async (entityType, opts) => {
      const { cmdEntitiesList } = await import('../commands/entities.js');
      const isAgent = checkAgentMode();
      const backend = await getBackendOnly(opts.apiKey, opts.baseUrl);
      const output = isAgent ? 'agent' : opts.output;
      await cmdEntitiesList(backend, entityType, { output });
    });

  entityCmd
    .command('delete')
    .description('Delete an entity and ALL its memories (cascade).')
    .option('--dry-run', 'Show what would be deleted without deleting.', false)
    .option('-u, --user-id <id>', 'Scope to user.')
    .option('--agent-id <id>', 'Scope to agent.')
    .option('--app-id <id>', 'Scope to app.')
    .option('--run-id <id>', 'Scope to run.')
    .option('--force', 'Skip confirmation.', false)
    .option('-o, --output <format>', 'Output: text, json, quiet.', 'text')
    .option('--api-key <key>', 'Override API key.')
    .option('--base-url <url>', 'Override API base URL.')
    .addHelpText(
      'after',
      '\nExamples:\n  $ memgo entity delete --user-id alice --force\n  $ memgo entity delete --user-id alice --dry-run'
    )
    .action(async (opts) => {
      const { cmdEntitiesDelete } = await import('../commands/entities.js');
      const isAgent = checkAgentMode();
      const backend = await getBackendOnly(opts.apiKey, opts.baseUrl);
      const output = isAgent ? 'agent' : opts.output;
      await cmdEntitiesDelete(backend, { ...opts, output });
    });
}
