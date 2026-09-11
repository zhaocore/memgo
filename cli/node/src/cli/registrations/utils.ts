import type { Command } from 'commander';
import { colors } from '../../output/branding.js';
import { setAgentMode } from '../../runtime/state.js';
import { CLI_VERSION } from '../../version.js';

import { resolveIds } from '../../application/entity-ids.js';
import { getBackendAndConfig } from '../context.js';

/**
 * 注册 utils 命令，保持参数和输出合同。
 * @param program - 本次装配的 Commander 根命令。
 */
export function registerUtils(program: Command): void {
  /**
   * 读取全局选项并同步本次调用的代理输出模式。
   * @returns 本次调用是否采用代理输出模式。
   */
  function checkAgentMode(): boolean {
    const value = !!(program.opts().json || program.opts().agent);
    setAgentMode(value);
    return value;
  }
  /** 输出当前 CLI 版本。 */
  function printVersion(): void {
    console.log(`  ${colors.brand('◆ MemGo')} CLI v${CLI_VERSION}`);
  }

  program
    .command('status')
    .description('Check connectivity and authentication.')
    .option('-o, --output <format>', 'Output: text, json.', 'text')
    .option('--api-key <key>', 'Override API key.')
    .option('--base-url <url>', 'Override API base URL.')
    .addHelpText('after', '\nExamples:\n  $ memgo status\n  $ memgo status -o json')
    .action(async (opts) => {
      const { cmdStatus } = await import('../commands/utils.js');
      const isAgent = checkAgentMode();
      const { backend, config } = await getBackendAndConfig(opts.apiKey, opts.baseUrl);
      const output = isAgent ? 'agent' : opts.output;
      await cmdStatus(backend, {
        userId: config.defaults.userId || undefined,
        agentId: config.defaults.agentId || undefined,
        output,
      });
    });

  program
    .command('version')
    .description('Show version and exit.')
    .action(() => {
      printVersion();
    });

  program
    .command('import <filePath>')
    .description('Import memories from a JSON file.')
    .option('-u, --user-id <id>', 'Override user ID.')
    .option('--agent-id <id>', 'Override agent ID.')
    .option('-o, --output <format>', 'Output: text, json.', 'text')
    .option('--api-key <key>', 'Override API key.')
    .option('--base-url <url>', 'Override API base URL.')
    .addHelpText(
      'after',
      '\nExamples:\n  $ memgo import data.json --user-id alice\n  $ memgo import data.json -u alice -o json'
    )
    .action(async (filePath, opts) => {
      const { cmdImport } = await import('../commands/utils.js');
      const isAgent = checkAgentMode();
      const { backend, config } = await getBackendAndConfig(opts.apiKey, opts.baseUrl);
      const ids = resolveIds(config, opts);
      const output = isAgent ? 'agent' : opts.output;
      await cmdImport(backend, filePath, {
        userId: ids.userId,
        agentId: ids.agentId,
        output,
      });
    });
}
