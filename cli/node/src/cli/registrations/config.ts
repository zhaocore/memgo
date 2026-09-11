import type { Command } from 'commander';
import { setAgentMode } from '../../runtime/state.js';

/**
 * 注册 config 命令，保持参数和输出合同。
 * @param program - 本次装配的 Commander 根命令。
 */
export function registerConfig(program: Command): void {
  /**
   * 读取全局选项并同步本次调用的代理输出模式。
   * @returns 本次调用是否采用代理输出模式。
   */
  function checkAgentMode(): boolean {
    const value = !!(program.opts().json || program.opts().agent);
    setAgentMode(value);
    return value;
  }

  const configCmd = program
    .command('config')
    .description('Manage memgo configuration.')
    .addHelpCommand(false);

  configCmd
    .command('show')
    .description('Display current configuration (secrets redacted).')
    .option('-o, --output <format>', 'Output: text, json.', 'text')
    .addHelpText('after', '\nExamples:\n  $ memgo config show\n  $ memgo config show -o json')
    .action(async (opts) => {
      const { cmdConfigShow } = await import('../commands/config.js');
      const isAgent = checkAgentMode();
      const output = isAgent ? 'agent' : opts.output;
      cmdConfigShow({ output });
    });

  configCmd
    .command('get <key>')
    .description('Get a configuration value.')
    .addHelpText(
      'after',
      '\nExamples:\n  $ memgo config get platform.api_key\n  $ memgo config get defaults.user_id'
    )
    .action(async (key) => {
      const { cmdConfigGet } = await import('../commands/config.js');
      checkAgentMode();
      cmdConfigGet(key);
    });

  configCmd
    .command('set <key> <value>')
    .description('Set a configuration value.')
    .addHelpText(
      'after',
      '\nExamples:\n  $ memgo config set defaults.user_id alice\n  $ memgo config set platform.base_url https://api.memgo.ai'
    )
    .action(async (key, value) => {
      const { cmdConfigSet } = await import('../commands/config.js');
      checkAgentMode();
      cmdConfigSet(key, value);
    });
}
