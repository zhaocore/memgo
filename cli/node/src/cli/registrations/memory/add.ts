import type { Command } from 'commander';
import { setAgentMode } from '../../../runtime/state.js';

import { resolveIds } from '../../../application/entity-ids.js';
import { getBackendAndConfig } from '../../context.js';

/**
 * 注册 memory/add 命令，保持参数和输出合同。
 * @param program - 本次装配的 Commander 根命令。
 */
export function registerMemoryAdd(program: Command): void {
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
    .command('add [text]')
    .description('Add a memory from text, messages, file, or stdin.')
    .option('-u, --user-id <id>', 'Scope to user.')
    .option('--agent-id <id>', 'Scope to agent.')
    .option('--app-id <id>', 'Scope to app.')
    .option('--run-id <id>', 'Scope to run.')
    .option('--messages <json>', 'Conversation messages as JSON.')
    .option('-f, --file <path>', 'Read messages from JSON file.')
    .option('-m, --metadata <json>', 'Custom metadata as JSON.')
    .option('--immutable', 'Prevent future updates.', false)
    .option('--no-infer', 'Skip inference, store raw.')
    .option('--expires <date>', 'Expiration date (YYYY-MM-DD).')
    .option('--categories <value>', 'Not supported on add, use --custom-categories instead.')
    .option('--custom-instructions <text>', 'Custom instructions for fact extraction.')
    .option(
      '--agent-custom-instructions <text>',
      'Extraction instructions for agent-scoped memories, overriding the project setting.'
    )
    .option(
      '--custom-categories <json>',
      'Custom categories as a JSON array of {name: description} objects.'
    )
    .option('--structured-data-schema <json>', 'Schema for structured data extraction, as JSON.')
    .option('--timestamp <unix>', 'Unix timestamp for the memory.', (v) => Number.parseInt(v))
    .option('-o, --output <format>', 'Output format: text, json, quiet.', 'text')
    .option('--api-key <key>', 'Override API key.')
    .option('--base-url <url>', 'Override API base URL.')
    .addHelpText(
      'after',
      '\nExamples:\n  $ memgo add "I prefer dark mode" --user-id alice\n  $ echo "text" | memgo add -u alice\n  $ memgo add --file msgs.json -u alice -o json'
    )
    .action(async (text, opts) => {
      const { cmdAdd } = await import('../../../cli/commands/memory/add.js');
      const isAgent = checkAgentMode();
      const { backend, config } = await getBackendAndConfig(opts.apiKey, opts.baseUrl);
      const ids = resolveIds(config, opts);
      const output = isAgent ? 'agent' : opts.output;
      await cmdAdd(backend, text, { ...ids, ...opts, output });
    });
}
