import type { Command } from 'commander';
import { setAgentMode } from '../../runtime/state.js';
import { richFormatHelp } from '../help.js';

import { getBackendOnly } from '../context.js';

/**
 * 注册 events 命令，保持参数和输出合同。
 * @param program - 本次装配的 Commander 根命令。
 */
export function registerEvents(program: Command): void {
  /**
   * 读取全局选项并同步本次调用的代理输出模式。
   * @returns 本次调用是否采用代理输出模式。
   */
  function checkAgentMode(): boolean {
    const value = !!(program.opts().json || program.opts().agent);
    setAgentMode(value);
    return value;
  }

  const eventCmd = program
    .command('event')
    .description('Inspect background processing events.')
    .addHelpCommand(false)
    .configureHelp({ formatHelp: richFormatHelp });

  eventCmd
    .command('list')
    .description('List recent background processing events.')
    .option('-o, --output <format>', 'Output: table, json.', 'table')
    .option('--api-key <key>', 'Override API key.')
    .option('--base-url <url>', 'Override API base URL.')
    .addHelpText('after', '\nExamples:\n  $ memgo event list\n  $ memgo event list -o json')
    .action(async (opts) => {
      const { cmdEventList } = await import('../commands/events.js');
      const isAgent = checkAgentMode();
      const backend = await getBackendOnly(opts.apiKey, opts.baseUrl);
      const output = isAgent ? 'agent' : opts.output;
      await cmdEventList(backend, { output });
    });

  eventCmd
    .command('status <eventId>')
    .description('Check the status of a specific background event.')
    .option('-o, --output <format>', 'Output: text, json.', 'text')
    .option('--api-key <key>', 'Override API key.')
    .option('--base-url <url>', 'Override API base URL.')
    .addHelpText(
      'after',
      '\nExamples:\n  $ memgo event status <event-id>\n  $ memgo event status <event-id> -o json'
    )
    .action(async (eventId, opts) => {
      const { cmdEventStatus } = await import('../commands/events.js');
      const isAgent = checkAgentMode();
      const backend = await getBackendOnly(opts.apiKey, opts.baseUrl);
      const output = isAgent ? 'agent' : opts.output;
      await cmdEventStatus(backend, eventId, { output });
    });
}
