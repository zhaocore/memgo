import type { Command } from 'commander';
import { colors } from '../../output/branding.js';
import { CLI_VERSION } from '../../version.js';

/**
 * 注册 help 命令，保持参数和输出合同。
 * @param program - 本次装配的 Commander 根命令。
 */
export function registerHelp(program: Command): void {
  program
    .command('help')
    .description('Show help. Use --json for machine-readable output (for LLM agents).')
    .option('--json', 'Output machine-readable JSON for LLM agents.', false)
    .addHelpText('after', '\nExamples:\n  $ memgo help\n  $ memgo help --json')
    .action((opts) => {
      // 子命令与根命令的 JSON 选项都支持机器可读帮助。

      if (opts.json || program.opts().json || program.opts().agent) {
        console.log(
          JSON.stringify(
            {
              name: 'memgo',
              version: CLI_VERSION,
              description: 'The Memory Layer for AI Agents',
            },
            null,
            2
          )
        );
      } else {
        const { brand: b } = colors;
        console.log(
          `${b('◆ MemGo CLI')} v${CLI_VERSION} · Node.js SDK\n  The Memory Layer for AI Agents\n`
        );
        console.log('Usage: memgo <command> [OPTIONS]\n');
        console.log('Commands:');
        console.log('  add              Add a memory from text, messages, file, or stdin');
        console.log('  search           Query your memory store (semantic, keyword, hybrid)');
        console.log('  get              Get a specific memory by ID');
        console.log('  list             List memories with optional filters');
        console.log("  update           Update a memory's text or metadata");
        console.log('  delete           Delete a memory, all memories, or an entity');
        console.log('  import           Import memories from a JSON file');
        console.log('  config           Manage configuration (show, get, set)');
        console.log('  entity           Manage entities (list, delete)');
        console.log('  event            Inspect background events (list, status)');
        console.log('  init             Interactive setup wizard');
        console.log('  status           Check connectivity and authentication');
        console.log();
        console.log('  memgo <command> --help    Get help for a command');
        console.log('  memgo help --json         Machine-readable help (for LLM agents)');
        console.log();
      }
    });
}
