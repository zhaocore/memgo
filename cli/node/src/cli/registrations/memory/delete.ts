import type { Command } from 'commander';
import { printError } from '../../../output/branding.js';
import { exit } from '../../../runtime/exit.js';
import { setAgentMode } from '../../../runtime/state.js';

import { resolveIds } from '../../../application/entity-ids.js';
import { getBackendAndConfig, getBackendOnly } from '../../context.js';

/**
 * 注册 memory/delete 命令，保持参数和输出合同。
 * @param program - 本次装配的 Commander 根命令。
 */
export function registerMemoryDelete(program: Command): void {
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
    .command('delete [memoryId]')
    .description('Delete a memory, all memories matching a scope, or an entity.')
    .option('--all', 'Delete all memories matching scope filters.', false)
    .option('--entity', 'Delete the entity itself and all its memories (cascade).', false)
    .option('--project', 'With --all: delete ALL memories project-wide.', false)
    .option('--dry-run', 'Show what would be deleted without deleting.', false)
    .option('--force', 'Skip confirmation.', false)
    .option('--delete-linked', 'Also delete memories linked to this memory.', false)
    .option('-u, --user-id <id>', 'Scope to user.')
    .option('--agent-id <id>', 'Scope to agent.')
    .option('--app-id <id>', 'Scope to app.')
    .option('--run-id <id>', 'Scope to run.')
    .option('-o, --output <format>', 'Output: text, json, quiet.', 'text')
    .option('--api-key <key>', 'Override API key.')
    .option('--base-url <url>', 'Override API base URL.')
    .addHelpText(
      'after',
      [
        '\nExamples:',
        '  $ memgo delete abc-123-def-456              # single memory',
        '  $ memgo delete --all -u alice --force        # all memories for user',
        '  $ memgo delete --all --project --force       # project-wide wipe',
        '  $ memgo delete --entity -u alice --force     # entity + all its memories',
      ].join('\n')
    )
    .action(async (memoryId, opts) => {
      const isAgent = checkAgentMode();
      const output = isAgent ? 'agent' : opts.output;
      // 校验互斥删除参数。
      if (memoryId && opts.all) {
        printError('Cannot combine <memoryId> with --all. Use one or the other.');
        exit(1);
      }
      if (memoryId && opts.entity) {
        printError('Cannot combine <memoryId> with --entity. Use one or the other.');
        exit(1);
      }
      if (opts.all && opts.entity) {
        printError('Cannot combine --all with --entity. Use one or the other.');
        exit(1);
      }
      if (!memoryId && !opts.all && !opts.entity) {
        printError(
          'Specify a memory ID, --all, or --entity.\n' +
            '  memgo delete <id>              Delete a single memory\n' +
            '  memgo delete --all [scope]     Delete all memories matching scope\n' +
            '  memgo delete --entity [scope]  Delete an entity and all its memories'
        );
        exit(1);
      }

      // 删除单条记忆。
      if (memoryId) {
        const { cmdDelete } = await import('../../../cli/commands/memory/delete.js');
        const backend = await getBackendOnly(opts.apiKey, opts.baseUrl);
        await cmdDelete(backend, memoryId, {
          output,
          dryRun: opts.dryRun,
          force: opts.force,
          deleteLinked: opts.deleteLinked,
        });
        return;
      }

      // 删除范围内全部记忆。
      if (opts.all) {
        const { cmdDeleteAll } = await import('../../../cli/commands/memory/delete-all.js');
        const { backend, config } = await getBackendAndConfig(opts.apiKey, opts.baseUrl);
        const ids = opts.project
          ? {
            userId: undefined,
            agentId: undefined,
            appId: undefined,
            runId: undefined,
          }
          : resolveIds(config, opts);
        await cmdDeleteAll(backend, {
          force: opts.force,
          dryRun: opts.dryRun,
          all: opts.project,
          ...ids,
          output,
        });
        return;
      }

      // 删除实体及其记忆。
      if (opts.entity) {
        const { cmdEntitiesDelete } = await import('../../commands/entities.js');
        const backend = await getBackendOnly(opts.apiKey, opts.baseUrl);
        await cmdEntitiesDelete(backend, { ...opts, output });
        return;
      }
    });
}
