import { Command } from 'commander';
import { captureEvent } from '../integrations/telemetry.js';
import { colors } from '../output/branding.js';
import { exit } from '../runtime/exit.js';
import { setAgentMode, setCurrentCommand } from '../runtime/state.js';
import { warnOptional } from '../runtime/warning.js';
import { CLI_VERSION } from '../version.js';
import { richFormatHelp } from './help.js';

import { registerAgentRush } from './registrations/agent-rush.js';
import { registerConfig } from './registrations/config.js';
import { registerEntities } from './registrations/entities.js';
import { registerEvents } from './registrations/events.js';
import { registerHelp } from './registrations/help.js';
import { registerIdentify } from './registrations/identify.js';
import { registerInit } from './registrations/init.js';
import { registerMemoryAdd } from './registrations/memory/add.js';
import { registerMemoryDelete } from './registrations/memory/delete.js';
import { registerMemoryGet } from './registrations/memory/get.js';
import { registerMemoryList } from './registrations/memory/list.js';
import { registerMemorySearch } from './registrations/memory/search.js';
import { registerMemoryUpdate } from './registrations/memory/update.js';
import { registerUtils } from './registrations/utils.js';
import { registerWhoami } from './registrations/whoami.js';

/**
 * 为单次 CLI 调用创建独立的命令树。
 * @returns 独立的 Commander 命令树。
 */
export function createProgram(): Command {
  const program = new Command();
  /** 输出当前 CLI 版本。 */
  function printVersion(): void {
    console.log(`  ${colors.brand('◆ MemGo')} CLI v${CLI_VERSION}`);
  }

  program
    .name('memgo')
    .exitOverride()
    .description(`◆ MemGo CLI v${CLI_VERSION} · Node.js SDK\n\nThe Memory Layer for AI Agents`)
    // 子命令后的选项归子命令，避免 init --agent 被全局 JSON 别名吞掉。

    .enablePositionalOptions()
    .option('--version', 'Show version and exit.')
    .on('option:version', () => {
      printVersion();
      exit(0);
    })
    .option('--json', 'Output as JSON for agent/programmatic use.')
    .option(
      '--agent',
      'Output as JSON for agent/programmatic use. (alias: --json) Place BEFORE the subcommand: `memgo --agent <cmd>`. On `init`, `memgo init --agent` is the Agent Mode bootstrap flag instead.'
    )
    .usage('<command> [options]')
    .helpOption('--help', 'Show this message and exit.')
    .addHelpCommand(false)
    .configureHelp({ formatHelp: richFormatHelp });

  program.hook('preAction', (_thisCommand, actionCommand) => {
    try {
      setAgentMode(!!(program.opts().json || program.opts().agent || actionCommand.opts().json));
      const commandName = actionCommand.name();
      const parentName = actionCommand.parent?.name();
      const fullCommand =
        parentName && parentName !== 'memgo' ? `${parentName}.${commandName}` : commandName;
      // 记录当前命令，确保 JSON 错误信封包含命令名。

      setCurrentCommand(fullCommand);
      // init 自行发送完整遥测属性，此处跳过以免重复计数。

      if (fullCommand === 'init') return;
      const isAgent = !!(program.opts().json || program.opts().agent);
      captureEvent(
        `cli.${fullCommand}`,
        {
          command: fullCommand,
          is_agent: isAgent,
        },
        undefined
      );
    } catch {
      warnOptional('program');
    }
  });

  registerInit(program);
  registerIdentify(program);
  registerWhoami(program);
  registerAgentRush(program);
  registerMemoryAdd(program);
  registerMemorySearch(program);
  registerMemoryGet(program);
  registerMemoryList(program);
  registerMemoryUpdate(program);
  registerMemoryDelete(program);
  registerConfig(program);
  registerEntities(program);
  registerEvents(program);
  registerUtils(program);
  registerHelp(program);
  return program;
}
